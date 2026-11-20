package domain

// UserSpec describes one native user creation. Password is never persisted in
// profiles, history, or mutation results.
type UserSpec struct {
	Engine   Engine
	Database string
	Name     string
	Host     string
	// MongoHosts is the comma-separated host:port list written to a MongoDB
	// credential URI. The live user creation still uses the current session.
	MongoHosts string
	Password   string
}

// UserResult is the safe outcome of a user creation. Source points at the
// credential file written after the native account was created.
type UserResult struct {
	Mutation MutationResult
	Source   *Source
}
