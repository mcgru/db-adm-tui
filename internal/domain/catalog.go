package domain

import (
	"strings"
)

// ObjectKind identifies a tree/catalog node.
type ObjectKind string

const (
	KindDatabase   ObjectKind = "database"
	KindSchema     ObjectKind = "schema"
	KindTable      ObjectKind = "table"
	KindCollection ObjectKind = "collection"
	KindView       ObjectKind = "view"
	KindSequence   ObjectKind = "sequence"
	KindSecurity   ObjectKind = "security"
	KindUsers      ObjectKind = "users"
	KindRoles      ObjectKind = "roles"
	KindUser       ObjectKind = "user"
	KindRole       ObjectKind = "role"
)

// Access is the native access marker for a table, collection, or view.
type Access string

const (
	AccessUnknown Access = ""
	AccessRead    Access = "ro"
	AccessWrite   Access = "rw"
	AccessNone    Access = "—"
)

// ObjectRef is the stable identity of a catalog node.
type ObjectRef struct {
	Engine   Engine     `json:"engine,omitempty"`
	Kind     ObjectKind `json:"kind"`
	Database string     `json:"database,omitempty"`
	Schema   string     `json:"schema,omitempty"`
	Name     string     `json:"name,omitempty"`
}

// Key returns a collision-free identity string for a node.
func (r ObjectRef) Key() string {
	return strings.Join([]string{
		string(r.Engine), string(r.Kind), r.Database, r.Schema, r.Name,
	}, "\x1f")
}

// AccessFrom maps a native read/write check to a marker.
func AccessFrom(read, write bool) Access {
	switch {
	case write:
		return AccessWrite
	case read:
		return AccessRead
	default:
		return AccessNone
	}
}

// AccessMarker reports whether the kind carries an access marker.
func (k ObjectKind) AccessMarker() bool {
	switch k {
	case KindTable, KindCollection, KindView:
		return true
	}
	return false
}

// Node is one tree node returned by an engine catalog.
type Node struct {
	Ref        ObjectRef  `json:"ref"`
	Name       string     `json:"name"`
	Kind       ObjectKind `json:"kind"`
	Access     Access     `json:"access,omitempty"`
	Expandable bool       `json:"expandable,omitempty"`
	Scope      string     `json:"scope,omitempty"`
	Attributes []string   `json:"attributes,omitempty"`
}

// HasAccessMarker reports whether the kind carries an access marker.
func (n Node) HasAccessMarker() bool { return n.Kind.AccessMarker() }

// IsObject reports whether the node is a browsable object.
func (n Node) IsObject() bool {
	switch n.Kind {
	case KindTable, KindCollection, KindView, KindSequence:
		return true
	}
	return false
}

// IsPrincipal reports whether the node is a user or role.
func (n Node) IsPrincipal() bool {
	return n.Kind == KindUser || n.Kind == KindRole
}

// Principal is one native user or role.
type Principal struct {
	Name       string     `json:"name"`
	Kind       ObjectKind `json:"kind"`
	Database   string     `json:"database,omitempty"`
	Scope      string     `json:"scope,omitempty"`
	Attributes []string   `json:"attributes,omitempty"`
}
