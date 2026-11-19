package domain

// MutationResult is the safe audit record of one structured write.
type MutationResult struct {
	Statement string
	Affected  int64
}

// RoleRef identifies a native role or user principal. Database is used by
// MongoDB; MySQL names may include @host. Builtin marks roles that cannot be
// edited through this tool (currently MongoDB built-in roles).
type RoleRef struct {
	Engine   Engine
	Database string
	Name     string
	Kind     ObjectKind
	Builtin  bool
}

// Key returns a stable matrix identity. Kind is part of the identity because
// MongoDB users and roles share a namespace, and MySQL accounts are typed.
func (r RoleRef) Key() string {
	return string(r.Engine) + "\x1f" + r.Database + "\x1f" + string(r.Kind) + "\x1f" + r.Name
}

// RoleMatrix is the Principal × Role membership grid.
type RoleMatrix struct {
	Principals []RoleRef
	Roles      []RoleRef
	Members    map[string]map[string]bool
}

// PrivilegeMatrix is the Principal/Role × Object privilege grid. Cell values
// are native privilege or action names.
type PrivilegeMatrix struct {
	Principals []RoleRef
	Objects    []ObjectRef
	Cells      map[string]map[string][]string
}

// RoleMembership grants Role to Member. Both sides are native principals.
type RoleMembership struct {
	Role   RoleRef
	Member RoleRef
}

// PrivilegeGrant grants native privilege names on a target to a role. The
// engine adapter validates the privilege names against its own allowlist.
type PrivilegeGrant struct {
	Role       RoleRef
	Target     ObjectRef
	Privileges []string
}

// RoleTargetKinds returns the native grant scopes offered for an engine.
func RoleTargetKinds(engine Engine) []ObjectKind {
	switch engine {
	case EnginePostgreSQL:
		return []ObjectKind{KindDatabase, KindSchema, KindTable}
	case EngineMySQL:
		return []ObjectKind{KindDatabase, KindTable}
	case EngineMongoDB:
		return []ObjectKind{KindDatabase, KindCollection}
	default:
		return nil
	}
}

// RolePrivilegeChoices returns native privilege/action names for one target.
func RolePrivilegeChoices(engine Engine, kind ObjectKind) []string {
	switch engine {
	case EnginePostgreSQL:
		switch kind {
		case KindDatabase:
			return []string{"CONNECT", "CREATE", "TEMPORARY"}
		case KindSchema:
			return []string{"USAGE", "CREATE"}
		case KindTable, KindView:
			return []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER", "MAINTAIN"}
		}
	case EngineMySQL:
		switch kind {
		case KindDatabase:
			return []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "INDEX", "REFERENCES", "CREATE TEMPORARY TABLES", "LOCK TABLES", "EXECUTE", "EVENT", "TRIGGER"}
		case KindTable, KindView:
			return []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "INDEX", "REFERENCES", "TRIGGER"}
		}
	case EngineMongoDB:
		switch kind {
		case KindDatabase:
			return []string{"find", "insert", "update", "remove", "createCollection", "dropCollection", "createIndex", "dropIndex", "listCollections", "listIndexes"}
		case KindCollection:
			return []string{"find", "insert", "update", "remove", "createIndex", "dropIndex"}
		}
	}
	return nil
}
