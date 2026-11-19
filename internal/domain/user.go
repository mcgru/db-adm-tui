package domain

// UserSpec describes one native user creation. Password is never persisted in
// profiles, history, or mutation results.
type UserSpec struct {
	Engine   Engine
	Database string
	Name     string
	Host     string
	Password string
}

// UserResult is the safe outcome of a user creation. Source points at the
// credential file written after the native account was created.
type UserResult struct {
	Mutation MutationResult
	Source   *Source
}
