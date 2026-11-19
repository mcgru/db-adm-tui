package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RoleMatrixReader = (*connection)(nil)

func (c *connection) RoleMatrix(ctx context.Context) (domain.RoleMatrix, error) {
	matrix := domain.RoleMatrix{Members: map[string]map[string]bool{}}
	users, err := c.matrixUsers(ctx)
	if err != nil {
		return domain.RoleMatrix{}, err
	}
	roles, err := c.matrixRoles(ctx, true, false)
	if err != nil {
		return domain.RoleMatrix{}, err
	}
	roleByKey := map[string]domain.RoleRef{}
	for _, role := range roles {
		ref := matrixRoleRef(role)
		matrix.Roles = append(matrix.Roles, ref)
		roleByKey[ref.Key()] = ref
		if !ref.Builtin {
			matrix.Principals = append(matrix.Principals, ref)
		}
	}
	for _, user := range users {
		ref := domain.RoleRef{
			Engine:   domain.EngineMongoDB,
			Database: user.DB,
			Name:     user.User + "@" + user.DB,
			Kind:     domain.KindUser,
		}
		matrix.Principals = append(matrix.Principals, ref)
		key := ref.Key()
		for _, role := range user.Roles {
			memberRole, ok := roleByKey[domain.RoleRef{
				Engine:   domain.EngineMongoDB,
				Database: role.DB,
				Name:     role.Role + "@" + role.DB,
				Kind:     domain.KindRole,
			}.Key()]
			if !ok {
				memberRole = domain.RoleRef{
					Engine:   domain.EngineMongoDB,
					Database: role.DB,
					Name:     role.Role + "@" + role.DB,
					Kind:     domain.KindRole,
				}
			}
			if matrix.Members[key] == nil {
				matrix.Members[key] = map[string]bool{}
			}
			matrix.Members[key][memberRole.Key()] = true
		}
	}
	for _, role := range roles {
		if role.IsBuiltin {
			continue
		}
		ref := matrixRoleRef(role)
		key := ref.Key()
		for _, inherited := range role.Roles {
			inheritedRef, ok := roleByKey[domain.RoleRef{
				Engine:   domain.EngineMongoDB,
				Database: inherited.DB,
				Name:     inherited.Role + "@" + inherited.DB,
				Kind:     domain.KindRole,
			}.Key()]
			if !ok {
				inheritedRef = domain.RoleRef{
					Engine:   domain.EngineMongoDB,
					Database: inherited.DB,
					Name:     inherited.Role + "@" + inherited.DB,
					Kind:     domain.KindRole,
				}
			}
			if matrix.Members[key] == nil {
				matrix.Members[key] = map[string]bool{}
			}
			matrix.Members[key][inheritedRef.Key()] = true
		}
	}
	sort.Slice(matrix.Principals, func(i, j int) bool {
		if matrix.Principals[i].Name == matrix.Principals[j].Name {
			return matrix.Principals[i].Kind < matrix.Principals[j].Kind
		}
		return matrix.Principals[i].Name < matrix.Principals[j].Name
	})
	sortMatrixRoles(matrix.Roles)
	return matrix, nil
}

func (c *connection) PrivilegeMatrix(ctx context.Context, objects []domain.ObjectRef) (domain.PrivilegeMatrix, error) {
	matrix := domain.PrivilegeMatrix{
		Objects: append([]domain.ObjectRef(nil), objects...),
		Cells:   map[string]map[string][]string{},
	}
	roles, err := c.matrixRoles(ctx, false, true)
	if err != nil {
		return domain.PrivilegeMatrix{}, err
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].DB != roles[j].DB {
			return roles[i].DB < roles[j].DB
		}
		return roles[i].Role < roles[j].Role
	})
	for _, role := range roles {
		principal := matrixRoleRef(role)
		matrix.Principals = append(matrix.Principals, principal)
		roleKey := principal.Key()
		for _, privilege := range role.Privileges {
			actions := uniqueSorted(privilege.Actions)
			for _, object := range objects {
				if !mongoMatrixPrivilegeMatches(privilege.Resource, object) {
					continue
				}
				if matrix.Cells[roleKey] == nil {
					matrix.Cells[roleKey] = map[string][]string{}
				}
				for _, action := range actions {
					matrix.Cells[roleKey][object.Key()] = appendUnique(
						matrix.Cells[roleKey][object.Key()], action)
				}
			}
		}
	}
	for principal := range matrix.Cells {
		for object := range matrix.Cells[principal] {
			sort.Strings(matrix.Cells[principal][object])
		}
	}
	return matrix, nil
}

type matrixUser struct {
	User  string      `bson:"user"`
	DB    string      `bson:"db"`
	Roles []mongoRole `bson:"roles"`
}

type matrixRole struct {
	Role       string           `bson:"role"`
	DB         string           `bson:"db"`
	IsBuiltin  bool             `bson:"isBuiltin"`
	Roles      []mongoRole      `bson:"roles"`
	Privileges []mongoPrivilege `bson:"privileges"`
}

func matrixRoleRef(role matrixRole) domain.RoleRef {
	return domain.RoleRef{
		Engine:   domain.EngineMongoDB,
		Database: role.DB,
		Name:     role.Role + "@" + role.DB,
		Kind:     domain.KindRole,
		Builtin:  role.IsBuiltin,
	}
}

func (c *connection) matrixUsers(ctx context.Context) ([]matrixUser, error) {
	var result struct {
		Users []matrixUser `bson:"users"`
	}
	err := c.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: 1},
		{Key: "forAllDBs", Value: true},
	}).Decode(&result)
	if err != nil {
		return nil, safeError("role-matrix", c.password, err)
	}
	return result.Users, nil
}

func (c *connection) matrixRoles(ctx context.Context, includeBuiltin, privileges bool) ([]matrixRole, error) {
	command := bson.D{
		{Key: "rolesInfo", Value: 1},
		{Key: "forAllDBs", Value: true},
		{Key: "showBuiltinRoles", Value: includeBuiltin},
	}
	if privileges {
		command = append(command, bson.E{Key: "showPrivileges", Value: true})
	}
	var result struct {
		Roles []matrixRole `bson:"roles"`
	}
	if err := c.client.Database("admin").RunCommand(ctx, command).Decode(&result); err != nil {
		return nil, safeError("role-matrix", c.password, err)
	}
	return result.Roles, nil
}

// mongoMatrixPrivilegeMatches returns true only for the exact resource cell
// being edited. Cluster/anyResource and database-wide privileges belong to
// other cells (or outside the object matrix), so they must not light up a
// collection cell that would then grant a different resource.
func mongoMatrixPrivilegeMatches(resource bson.D, ref domain.ObjectRef) bool {
	values := mongoResourceValues(resource)
	if values["anyResource"] == true || values["cluster"] == true {
		return false
	}
	database, _ := values["db"].(string)
	if database == "" || database != ref.Database {
		return false
	}
	collection, _ := values["collection"].(string)
	switch ref.Kind {
	case domain.KindDatabase:
		return collection == ""
	case domain.KindCollection, domain.KindTable:
		return collection == ref.Name
	default:
		return false
	}
}

func sortMatrixRoles(roles []domain.RoleRef) {
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Database != roles[j].Database {
			return roles[i].Database < roles[j].Database
		}
		return roles[i].Name < roles[j].Name
	})
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
