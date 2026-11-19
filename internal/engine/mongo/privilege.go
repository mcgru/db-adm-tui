package mongo

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.PrivilegeReader = (*connection)(nil)

type mongoPrivilege struct {
	Resource bson.D   `bson:"resource"`
	Actions  []string `bson:"actions"`
}

type mongoRole struct {
	Role string `bson:"role"`
	DB   string `bson:"db"`
}

type mongoAuthInfo struct {
	AuthenticatedUsers []struct {
		User string `bson:"user"`
		DB   string `bson:"db"`
	} `bson:"authenticatedUsers"`
	AuthenticatedUserRoles      []mongoRole      `bson:"authenticatedUserRoles"`
	AuthenticatedUserPrivileges []mongoPrivilege `bson:"authenticatedUserPrivileges"`
}

type mongoConnectionStatus struct {
	AuthInfo mongoAuthInfo `bson:"authInfo"`
}

// Privileges reports native roles and inherited privileges from
// connectionStatus, usersInfo, and rolesInfo.
func (c *connection) Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error) {
	report := domain.PrivilegeReport{Scope: mongoPrivilegeScope(ref)}
	switch ref.Kind {
	case domain.KindUser, domain.KindRole:
		report.Principal = ref.Name
		user, db := splitMongoPrincipal(ref.Name, ref.Database)
		if ref.Kind == domain.KindRole {
			return c.rolePrivileges(ctx, report, user, db), nil
		}
		return c.userPrivileges(ctx, report, user, db), nil
	default:
		status, err := c.connectionStatus(ctx)
		if err != nil {
			return domain.PrivilegeReport{}, safeError("privileges", c.password, err)
		}
		if len(status.AuthInfo.AuthenticatedUsers) > 0 {
			user := status.AuthInfo.AuthenticatedUsers[0]
			report.Principal = user.User + "@" + user.DB
			report.Sections = append(report.Sections,
				domain.PrivilegeSection{Title: "roles", Lines: mongoRoleLines(status.AuthInfo.AuthenticatedUserRoles)})
			report.Sections = append(report.Sections,
				domain.PrivilegeSection{
					Title: "privileges",
					Lines: mongoPrivilegeLines(status.AuthInfo.AuthenticatedUserPrivileges, ref, true),
				})
		}
		return report, nil
	}
}

func mongoPrivilegeScope(ref domain.ObjectRef) string {
	switch ref.Kind {
	case domain.KindCollection, domain.KindTable:
		if ref.Database != "" {
			return "collection " + ref.Database + "." + ref.Name
		}
		return "collection " + ref.Name
	case domain.KindDatabase:
		return "database " + ref.Database
	case domain.KindUser, domain.KindRole:
		return string(ref.Kind) + " " + ref.Name
	case domain.KindSecurity:
		return "connection"
	default:
		return string(ref.Kind)
	}
}

func (c *connection) connectionStatus(ctx context.Context) (mongoConnectionStatus, error) {
	var result mongoConnectionStatus
	err := c.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "connectionStatus", Value: 1},
		{Key: "showPrivileges", Value: true},
	}).Decode(&result)
	return result, err
}

func (c *connection) userPrivileges(ctx context.Context, report domain.PrivilegeReport, user, db string) domain.PrivilegeReport {
	var result struct {
		Users []struct {
			User                string           `bson:"user"`
			DB                  string           `bson:"db"`
			Roles               []mongoRole      `bson:"roles"`
			InheritedRoles      []mongoRole      `bson:"inheritedRoles"`
			Privileges          []mongoPrivilege `bson:"privileges"`
			InheritedPrivileges []mongoPrivilege `bson:"inheritedPrivileges"`
		} `bson:"users"`
	}

	err := c.client.Database(db).RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{{Key: "user", Value: user}, {Key: "db", Value: db}}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&result)
	if err != nil {
		report.Sections = append(report.Sections, domain.PrivilegeSection{
			Title: "grants", Err: mongoPrivilegeError(c.password, err),
		})
		return report
	}
	if len(result.Users) == 0 {
		report.Sections = append(report.Sections, domain.PrivilegeSection{Title: "grants", Lines: nil})
		return report
	}
	entry := result.Users[0]
	report.Sections = append(report.Sections,
		domain.PrivilegeSection{Title: "roles", Lines: mongoRoleLines(entry.Roles)},
		domain.PrivilegeSection{Title: "inherited roles", Lines: mongoRoleLines(entry.InheritedRoles)},
		domain.PrivilegeSection{Title: "privileges", Lines: mongoPrivilegeLines(entry.Privileges, domain.ObjectRef{}, false)},
		domain.PrivilegeSection{Title: "inherited privileges", Lines: mongoPrivilegeLines(entry.InheritedPrivileges, domain.ObjectRef{}, false)},
	)
	return report
}

func (c *connection) rolePrivileges(ctx context.Context, report domain.PrivilegeReport, role, db string) domain.PrivilegeReport {
	var result struct {
		Roles []struct {
			Role                string           `bson:"role"`
			DB                  string           `bson:"db"`
			Privileges          []mongoPrivilege `bson:"privileges"`
			InheritedRoles      []mongoRole      `bson:"inheritedRoles"`
			InheritedPrivileges []mongoPrivilege `bson:"inheritedPrivileges"`
		} `bson:"roles"`
	}
	err := c.client.Database(db).RunCommand(ctx, bson.D{
		{Key: "rolesInfo", Value: bson.D{{Key: "role", Value: role}, {Key: "db", Value: db}}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&result)
	if err != nil {
		report.Sections = append(report.Sections, domain.PrivilegeSection{
			Title: "grants", Err: mongoPrivilegeError(c.password, err),
		})
		return report
	}
	if len(result.Roles) == 0 {
		report.Sections = append(report.Sections, domain.PrivilegeSection{Title: "grants", Lines: nil})
		return report
	}
	entry := result.Roles[0]
	report.Sections = append(report.Sections,
		domain.PrivilegeSection{Title: "privileges", Lines: mongoPrivilegeLines(entry.Privileges, domain.ObjectRef{}, false)},
		domain.PrivilegeSection{Title: "inherited roles", Lines: mongoRoleLines(entry.InheritedRoles)},
		domain.PrivilegeSection{Title: "inherited privileges", Lines: mongoPrivilegeLines(entry.InheritedPrivileges, domain.ObjectRef{}, false)},
	)
	return report
}

func splitMongoPrincipal(name, fallbackDB string) (string, string) {
	if index := strings.LastIndex(name, "@"); index >= 0 {
		return name[:index], name[index+1:]
	}
	if fallbackDB == "" {
		fallbackDB = "admin"
	}
	return name, fallbackDB
}

func mongoRoleLines(roles []mongoRole) []string {
	lines := make([]string, 0, len(roles))
	for _, role := range roles {
		lines = append(lines, role.Role+"@"+role.DB)
	}
	return lines
}

func mongoPrivilegeLines(privileges []mongoPrivilege, ref domain.ObjectRef, filterObject bool) []string {
	lines := make([]string, 0, len(privileges))
	for _, privilege := range privileges {
		if filterObject && !mongoPrivilegeMatches(privilege.Resource, ref) {
			continue
		}
		actions := strings.Join(privilege.Actions, ", ")
		lines = append(lines, fmt.Sprintf("%s on %s", actions, mongoResourceString(privilege.Resource)))
	}
	return lines
}

func mongoPrivilegeMatches(resource bson.D, ref domain.ObjectRef) bool {
	values := mongoResourceValues(resource)
	if values["anyResource"] == true || values["cluster"] == true {
		return true
	}
	db, _ := values["db"].(string)
	if ref.Database != "" && db != "" && db != ref.Database {
		return false
	}
	collection, _ := values["collection"].(string)
	if ref.Kind == domain.KindCollection || ref.Kind == domain.KindTable {
		if collection != "" && collection != ref.Name {
			return false
		}
	}
	return true
}

func mongoResourceString(resource bson.D) string {
	values := mongoResourceValues(resource)
	if values["cluster"] == true {
		return "cluster"
	}
	if values["anyResource"] == true {
		return "anyResource"
	}
	db, _ := values["db"].(string)
	collection, _ := values["collection"].(string)
	switch {
	case db != "" && collection != "":
		return db + "." + collection
	case db != "":
		return "db " + db
	default:
		return "resource"
	}
}

func mongoResourceValues(resource bson.D) map[string]any {
	values := make(map[string]any, len(resource))
	for _, element := range resource {
		switch value := element.Value.(type) {
		case bool:
			values[element.Key] = value
		case int32:
			values[element.Key] = value != 0
		case int64:
			values[element.Key] = value != 0
		case string:
			values[element.Key] = value
		default:
			values[element.Key] = element.Value
		}
	}
	return values
}

func mongoPrivilegeError(password string, err error) string {
	safe := safeError("privileges", password, err)
	if domain.KindOf(safe) == domain.KindPermission {
		return "not permitted. This account cannot read this section."
	}
	return safe.Error()
}
