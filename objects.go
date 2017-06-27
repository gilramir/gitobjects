package gitobjects

type Object interface {
	Type() string
	Sha1() string
}
