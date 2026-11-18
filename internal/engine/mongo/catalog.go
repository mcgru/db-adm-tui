package mongo

import (
	"context"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.Catalog = (*connection)(nil)

// Databases implements ports.Catalog.
func (c *connection) Databases(ctx context.Context) ([]domain.Node, error) {
	names, err := c.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	sort.Strings(names)
	nodes := make([]domain.Node, 0, len(names))
	for _, name := range names {
		nodes = append(nodes, databaseNode(name))
	}
	return nodes, nil
}

// Children implements ports.Catalog with collections and views. Access comes
// from the authenticated user's roles in connectionStatus.
func (c *connection) Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error) {
	if ref.Kind != domain.KindDatabase {
		return nil, domain.NewError(domain.KindConfig, "catalog", "the node has no catalog children")
	}
	cursor, err := c.client.Database(ref.Database).ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	var specs []struct {
		Name string `bson:"name"`
		Type string `bson:"type"`
	}
	if err := cursor.All(ctx, &specs); err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })

	access := c.accessFor(ctx, ref.Database)
	nodes := make([]domain.Node, 0, len(specs))
	for _, spec := range specs {
		kind := domain.KindCollection
		if strings.EqualFold(spec.Type, "view") {
			kind = domain.KindView
		}
		node := domain.Node{
			Ref: domain.ObjectRef{
				Engine:   domain.EngineMongoDB,
				Database: ref.Database,
				Name:     spec.Name,
				Kind:     kind,
			},
			Name:   spec.Name,
			Kind:   kind,
			Access: domain.AccessUnknown,
		}
		if kind.AccessMarker() {
			node.Access = access
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Principals implements ports.Catalog with database-scoped users and roles.
func (c *connection) Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error) {
	switch ref.Kind {
	case domain.KindUsers:
		return c.users(ctx)
	case domain.KindRoles:
		return c.roles(ctx)
	default:
		return nil, domain.NewError(domain.KindConfig, "catalog", "the node has no principals")
	}
}

func (c *connection) users(ctx context.Context) ([]domain.Principal, error) {
	var result struct {
		Users []struct {
			User  string `bson:"user"`
			DB    string `bson:"db"`
			Roles []struct {
				Role string `bson:"role"`
				DB   string `bson:"db"`
			} `bson:"roles"`
		} `bson:"users"`
	}
	err := c.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{{Key: "forAllDBs", Value: true}}},
	}).Decode(&result)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	principals := make([]domain.Principal, 0, len(result.Users))
	for _, user := range result.Users {
		attributes := make([]string, 0, len(user.Roles))
		for _, role := range user.Roles {
			attributes = append(attributes, role.Role+"@"+role.DB)
		}
		sort.Strings(attributes)
		principals = append(principals, domain.Principal{
			Name:       user.User + "@" + user.DB,
			Kind:       domain.KindUser,
			Database:   user.DB,
			Scope:      user.DB,
			Attributes: attributes,
		})
	}
	sortPrincipals(principals)
	return principals, nil
}

func (c *connection) roles(ctx context.Context) ([]domain.Principal, error) {
	var result struct {
		Roles []struct {
			Role      string `bson:"role"`
			DB        string `bson:"db"`
			IsBuiltin bool   `bson:"isBuiltin"`
			Roles     []struct {
				Role string `bson:"role"`
				DB   string `bson:"db"`
			} `bson:"roles"`
		} `bson:"roles"`
	}
	err := c.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "rolesInfo", Value: bson.D{{Key: "forAllDBs", Value: true}}},
		{Key: "showBuiltinRoles", Value: false},
	}).Decode(&result)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	var principals []domain.Principal
	for _, role := range result.Roles {
		if role.IsBuiltin {
			continue
		}
		attributes := make([]string, 0, len(role.Roles))
		for _, inherited := range role.Roles {
			attributes = append(attributes, "INHERITS "+inherited.Role+"@"+inherited.DB)
		}
		sort.Strings(attributes)
		principals = append(principals, domain.Principal{
			Name:       role.Role + "@" + role.DB,
			Kind:       domain.KindRole,
			Database:   role.DB,
			Scope:      role.DB,
			Attributes: attributes,
		})
	}
	sortPrincipals(principals)
	return principals, nil
}

// OrderKey implements ports.Catalog. MongoDB always has a stable _id order.
func (c *connection) OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error) {
	if ref.Kind != domain.KindCollection && ref.Kind != domain.KindView {
		return "", nil
	}
	return "_id", nil
}

type mongoAuthRole struct {
	role string
	db   string
}

type mongoAuth struct {
	loaded bool
	known  bool
	roles  []mongoAuthRole
}

// accessFor derives the marker for one database from the authenticated user's
// roles. An unauthenticated server or an unrecognized custom role yields
// unknown access, which omits the marker.
func (c *connection) accessFor(ctx context.Context, database string) domain.Access {
	roles, known := c.authRoles(ctx)
	if !known {
		return domain.AccessUnknown
	}
	read := false
	write := false
	unknown := false
	for _, role := range roles {
		state := classifyMongoRole(role, database)
		if !state.known {
			unknown = true
			continue
		}
		if state.write {
			write = true
		}
		if state.read {
			read = true
		}
	}
	if write {
		return domain.AccessFrom(true, true)
	}
	if read {
		return domain.AccessFrom(true, false)
	}
	if unknown {
		return domain.AccessUnknown
	}
	return domain.AccessNone
}

func (c *connection) authRoles(ctx context.Context) ([]mongoAuthRole, bool) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if c.auth.loaded {
		return c.auth.roles, c.auth.known
	}

	var result struct {
		AuthInfo struct {
			AuthenticatedUsers []struct {
				User string `bson:"user"`
				DB   string `bson:"db"`
			} `bson:"authenticatedUsers"`
			AuthenticatedUserRoles []struct {
				Role string `bson:"role"`
				DB   string `bson:"db"`
			} `bson:"authenticatedUserRoles"`
		} `bson:"authInfo"`
	}
	err := c.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "connectionStatus", Value: 1},
	}).Decode(&result)
	if err != nil || len(result.AuthInfo.AuthenticatedUsers) == 0 {
		return nil, false
	}
	c.auth.loaded = true
	roles := make([]mongoAuthRole, 0, len(result.AuthInfo.AuthenticatedUserRoles))
	for _, role := range result.AuthInfo.AuthenticatedUserRoles {
		roles = append(roles, mongoAuthRole{role: role.Role, db: role.DB})
	}
	c.auth.roles = roles
	c.auth.known = true
	return roles, true
}

type mongoRoleState struct {
	read  bool
	write bool
	known bool
}

func classifyMongoRole(role mongoAuthRole, database string) mongoRoleState {
	scoped := role.db == database
	switch role.role {
	case "root":
		return mongoRoleState{read: true, write: true, known: true}
	case "readWriteAnyDatabase":
		return mongoRoleState{read: true, write: true, known: true}
	case "readAnyDatabase":
		return mongoRoleState{read: true, known: true}
	case "readWrite", "dbOwner":
		if scoped {
			return mongoRoleState{read: true, write: true, known: true}
		}
		return mongoRoleState{known: true}
	case "read":
		if scoped {
			return mongoRoleState{read: true, known: true}
		}
		return mongoRoleState{known: true}
	case "dbAdmin", "userAdmin":
		return mongoRoleState{known: true}
	case "dbAdminAnyDatabase", "userAdminAnyDatabase",
		"clusterAdmin", "clusterManager", "clusterMonitor", "hostManager":
		return mongoRoleState{known: true}
	default:
		// A custom role may grant data access; do not claim "no access".
		return mongoRoleState{}
	}
}

func sortPrincipals(principals []domain.Principal) {
	sort.SliceStable(principals, func(i, j int) bool { return principals[i].Name < principals[j].Name })
}

func databaseNode(name string) domain.Node {
	return domain.Node{
		Ref: domain.ObjectRef{
			Engine:   domain.EngineMongoDB,
			Database: name,
			Name:     name,
			Kind:     domain.KindDatabase,
		},
		Name:       name,
		Kind:       domain.KindDatabase,
		Expandable: true,
	}
}
