package domain

import "strings"

// Engine identifies a supported database engine.
type Engine string

const (
	EnginePostgreSQL Engine = "postgresql"
	EngineMySQL      Engine = "mysql"
	EngineMongoDB    Engine = "mongodb"
)

// Engines lists the engines supported by v1 in display order.
var Engines = []Engine{EnginePostgreSQL, EngineMySQL, EngineMongoDB}

// Valid reports whether e is a supported engine.
func (e Engine) Valid() bool {
	switch e {
	case EnginePostgreSQL, EngineMySQL, EngineMongoDB:
		return true
	}
	return false
}

// DisplayName returns the engine name shown in the interface.
func (e Engine) DisplayName() string {
	switch e {
	case EnginePostgreSQL:
		return "PostgreSQL"
	case EngineMySQL:
		return "MySQL"
	case EngineMongoDB:
		return "MongoDB"
	default:
		return string(e)
	}
}

// DefaultPort returns the engine's conventional port.
func (e Engine) DefaultPort() int {
	switch e {
	case EnginePostgreSQL:
		return 5432
	case EngineMySQL:
		return 3306
	case EngineMongoDB:
		return 27017
	default:
		return 0
	}
}

// Scheme returns the URI scheme used by the engine.
func (e Engine) Scheme() string {
	switch e {
	case EnginePostgreSQL:
		return "postgres"
	case EngineMySQL:
		return "mysql"
	case EngineMongoDB:
		return "mongodb"
	default:
		return string(e)
	}
}

// ParseEngine resolves a stored engine value.
func ParseEngine(s string) (Engine, bool) {
	e := Engine(strings.ToLower(strings.TrimSpace(s)))
	return e, e.Valid()
}

// ParseScheme resolves a URI scheme to an engine. PostgreSQL accepts both
// postgres:// and postgresql://.
func ParseScheme(s string) (Engine, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "postgres", "postgresql":
		return EnginePostgreSQL, true
	case "mysql":
		return EngineMySQL, true
	case "mongodb":
		return EngineMongoDB, true
	}
	return "", false
}
