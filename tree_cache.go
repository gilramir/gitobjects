package gitobjects

import (
	"sync"
)

type treeCacheConcurrentSafe struct {
	sync.RWMutex
	// Key = sha1, Value = *Tree
	treeCache map[string]*Tree
}

func NewTreeCache() *treeCacheConcurrentSafe {
	return &treeCacheConcurrentSafe{
		treeCache: make(map[string]*Tree),
	}
}

func (self *treeCacheConcurrentSafe) Has(sha1 string) bool {
	self.RLock()
	defer self.RUnlock()
	_, has := self.treeCache[sha1]
	return has
}

func (self *treeCacheConcurrentSafe) Get(sha1 string) (*Tree, bool) {
	self.RLock()
	defer self.RUnlock()
	tree, has := self.treeCache[sha1]
	return tree, has
}

func (self *treeCacheConcurrentSafe) Set(sha1 string, tree *Tree) {
	self.Lock()
	defer self.Unlock()
	self.treeCache[sha1] = tree
}

func (self *treeCacheConcurrentSafe) CreateIfNotPresent(sha1 string) *Tree {
	self.Lock()
	defer self.Unlock()
	existing, has := self.treeCache[sha1]
	if has {
		return existing
	} else {
		newTree := &Tree{
			sha1: sha1,
		}
		self.treeCache[sha1] = newTree
		return newTree
	}
}
