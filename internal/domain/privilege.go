package domain

// PrivilegeSection is one native privilege section. Lines are engine-native
// strings; Err is a per-section failure and never hides the other sections.
type PrivilegeSection struct {
	Title string
	Lines []string
	Err   string
}

// PrivilegeReport is one principal's native grants, memberships, and effective
// checks. The TUI renders the native terms without a cross-engine vocabulary.
type PrivilegeReport struct {
	Principal string
	Scope     string
	Note      string
	Sections  []PrivilegeSection
}
