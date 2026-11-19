package mongo

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RoleAdmin = (*connection)(nil)

var mongoCollectionActions = map[string]bool{
	"find": true, "insert": true, "update": true, "remove": true,
	"createIndex": true, "dropIndex": true,
}

var mongoDatabaseActions = map[string]bool{
	"find": true, "insert": true, "update": true, "remove": true,
	"createCollection": true, "dropCollection": true,
	"createIndex": true, "dropIndex": true,
	"listCollections": true, "listIndexes": true,
}

func (c *connection) CreateRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	db, name, err := mongoRoleRef(role)
	if err != nil {
		return domain.MutationResult{}, err
	}
	statement := fmt.Sprintf("db.createRole({role:%q, privileges:[], roles:[]})", name)
	err = c.client.Database(db).RunCommand(ctx, bson.D{
		{Key: "createRole", Value: name},
		{Key: "privileges", Value: bson.A{}},
		{Key: "roles", Value: bson.A{}},
	}).Err()
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("role", c.password, err)
	}
	return domain.MutationResult{Statement: statement}, nil
}

func (c *connection) DropRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	db, name, err := mongoRoleRef(role)
	if err != nil {
		return domain.MutationResult{}, err
	}
	statement := fmt.Sprintf("db.dropRole(%q)", name)
	if err := c.client.Database(db).RunCommand(ctx, bson.D{{Key: "dropRole", Value: name}}).Err(); err != nil {
		return domain.MutationResult{Statement: statement}, safeError("role", c.password, err)
	}
	return domain.MutationResult{Statement: statement}, nil
}

func (c *connection) GrantRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	return c.mongoMembership(ctx, "grant", membership)
}

func (c *connection) RevokeRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	return c.mongoMembership(ctx, "revoke", membership)
}

func (c *connection) mongoMembership(ctx context.Context, verb string, membership domain.RoleMembership) (domain.MutationResult, error) {
	roleDB, roleName, err := mongoRoleRef(membership.Role)
	if err != nil {
		return domain.MutationResult{}, err
	}
	memberDB, memberName, err := mongoRoleRef(membership.Member)
	if err != nil {
		return domain.MutationResult{}, err
	}
	roleDoc := bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: roleDB}}
	var command string
	var value any = memberName
	if membership.Member.Kind == domain.KindRole {
		command = verb + "RolesFromRole"
		if verb == "grant" {
			command = "grantRolesToRole"
		}
	} else {
		command = verb + "RolesFromUser"
		if verb == "grant" {
			command = "grantRolesToUser"
		}
	}
	statement := fmt.Sprintf("%s.%s(%q, [%s@%s])", memberDB, command, memberName, roleName, roleDB)
	err = c.client.Database(memberDB).RunCommand(ctx, bson.D{
		{Key: command, Value: value},
		{Key: "roles", Value: bson.A{roleDoc}},
	}).Err()
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("role", c.password, err)
	}
	return domain.MutationResult{Statement: statement}, nil
}

func (c *connection) GrantPrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	return c.mongoPrivileges(ctx, "grantPrivilegesToRole", grant)
}

func (c *connection) RevokePrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	return c.mongoPrivileges(ctx, "revokePrivilegesFromRole", grant)
}

func (c *connection) mongoPrivileges(ctx context.Context, command string, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	db, role, err := mongoRoleRef(grant.Role)
	if err != nil {
		return domain.MutationResult{}, err
	}
	actions, resource, label, err := mongoGrant(grant)
	if err != nil {
		return domain.MutationResult{}, err
	}
	statement := fmt.Sprintf("db.%s(%q, [{resource:%s, actions:[%s]}])",
		command, role, label, strings.Join(actions, ", "))
	err = c.client.Database(db).RunCommand(ctx, bson.D{
		{Key: command, Value: role},
		{Key: "privileges", Value: bson.A{bson.D{
			{Key: "resource", Value: resource},
			{Key: "actions", Value: actions},
		}}},
	}).Err()
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("role", c.password, err)
	}
	return domain.MutationResult{Statement: statement}, nil
}

func mongoGrant(grant domain.PrivilegeGrant) ([]string, bson.D, string, error) {
	target := grant.Target
	switch target.Kind {
	case domain.KindDatabase:
		db := target.Database
		if db == "" {
			db = target.Name
		}
		if db == "" {
			return nil, nil, "", domain.NewError(domain.KindConfig, "role", "the database is empty")
		}
		actions, err := mongoActions(grant.Privileges, mongoDatabaseActions, "database")
		return actions, bson.D{{Key: "db", Value: db}}, "{db:" + db + "}", err
	case domain.KindCollection, domain.KindTable:
		db := target.Database
		if db == "" {
			db = target.Schema
		}
		if db == "" || target.Name == "" {
			return nil, nil, "", domain.NewError(domain.KindConfig, "role", "the collection is not fully qualified")
		}
		actions, err := mongoActions(grant.Privileges, mongoCollectionActions, "collection")
		return actions, bson.D{{Key: "db", Value: db}, {Key: "collection", Value: target.Name}},
			"{db:" + db + ", collection:" + target.Name + "}", err
	default:
		return nil, nil, "", domain.NewError(domain.KindConfig, "role", "unsupported MongoDB grant target")
	}
}

func mongoActions(values []string, allowed map[string]bool, kind string) ([]string, error) {
	if len(values) == 0 {
		return nil, domain.NewError(domain.KindConfig, "role", "choose at least one privilege")
	}
	seen := map[string]bool{}
	actions := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if !allowed[name] {
			return nil, domain.NewError(domain.KindConfig, "role",
				"unsupported "+kind+" action "+value)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		actions = append(actions, name)
	}
	return actions, nil
}

func mongoRoleRef(role domain.RoleRef) (string, string, error) {
	name := strings.TrimSpace(role.Name)
	db := strings.TrimSpace(role.Database)
	if index := strings.LastIndex(name, "@"); index >= 0 {
		if db == "" {
			db = name[index+1:]
		}
		name = name[:index]
	}
	if db == "" || name == "" {
		return "", "", domain.NewError(domain.KindConfig, "role", "the MongoDB role is not database-scoped")
	}
	return db, name, nil
}
