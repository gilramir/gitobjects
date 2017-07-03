package gitobjects

import (
	"fmt"
)

type Entry struct {
	sha1        string
	permissions string // XXX - int?
	name        string

	tree *Tree
	blob *Blob
}

func (self *Entry) Type() string {
	if self.tree != nil {
		return "tree"
	} else if self.blob != nil {
		return "blob"
	} else {
		panic(fmt.Sprintf("Entry sha1=%s has neither tree nor blob", self.sha1))
	}
}

func (self *Entry) Name() string {
	return self.name
}

func (self *Entry) Sha1() string {
	return self.sha1
}

func (self *Entry) Tree() *Tree {
	if self.tree == nil {
		panic(fmt.Sprintf("Entry %s has no tree", self.sha1))
	}

	return self.tree
}
