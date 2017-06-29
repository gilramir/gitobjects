package gitobjects

type Object interface {
	// Returns the type of Object
	Type() string

	// Returns the sha1 of the object
	Sha1() string

	// Do any activities that require reading from disk
	// to populate internal information about the object
	Instantiate(repo *Repo) error
}
