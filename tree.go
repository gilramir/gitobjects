package gitobjects

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	"path/filepath"
	"strings"
	"sync"
)

type Tree struct {
	sync.RWMutex
	sha1         string
	entries      []*Entry
	instantiated bool
}

type BlobPath struct {
	Blob *Blob
	Path string
}

func (self *Tree) Type() string {
	return "tree"
}

func (self *Tree) Sha1() string {
	return self.sha1
}

func (self *Tree) Instantiate(repo *Repo) error {
	self.Lock()
	defer self.Unlock()

	if self.instantiated {
		return nil
	}
	if self.sha1 == "" {
		panic("Instantiate called on Tree that has no sha1")
	}
	output, err := repo.CmdOutput([]string{"cat-file", "-p", self.sha1})
	if err != nil {
		return errors.Wrapf(err, "Calling cat-file commit on %s", self.sha1)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line[52] != '\t' {
			return errors.Errorf("No tab at pos=52 in line for cat-file of tree %s: %s",
				self.sha1, line)
		}
		// permissions, entryType, sha1
		fields := strings.Split(line[:52], " ")
		name := line[53:]
		permissions := fields[0]
		type_ := fields[1]
		entrySha1 := fields[2]
		entry := &Entry{
			sha1:        entrySha1,
			permissions: permissions,
			name:        name,
		}

		switch type_ {
		case "tree":
			// XXX - add switch to use or not use cache?
			entryTree, has := repo.treeCache.Get(entrySha1)
			if !has {
				entryTree = &Tree{
					sha1: entrySha1,
				}
				repo.treeCache.Set(entrySha1, entryTree)
				err := entryTree.Instantiate(repo)
				if err != nil {
					return errors.Wrapf(err, "Instanting tree %s", entrySha1)
				}
			}
			entry.tree = entryTree
		case "blob":
			entry.blob = &Blob{
				sha1: entrySha1,
			}
		default:
			panic("cannot reach")
		}
		self.entries = append(self.entries, entry)
	}

	// Scanner error?
	err = scanner.Err()
	if err != nil {
		return errors.Wrapf(err, "Scanning cat-file tree %s output", self.sha1)
	}
	self.instantiated = true
	return nil
}

func (self *Tree) StreamBlobPathsUnique(repo *Repo, sha1sSeen map[string]bool) (<-chan *BlobPath, <-chan error) {
	self.RLock()
	defer self.RUnlock()

	blobPathChan := make(chan *BlobPath)
	errorChan := make(chan error)

	go self._streamBlobPathsUnique("", repo, sha1sSeen, blobPathChan, errorChan)
	return blobPathChan, errorChan
}

func (self *Tree) _streamBlobPathsUnique(parentPath string, repo *Repo, sha1sSeen map[string]bool,
	blobPathChan chan<- *BlobPath, errorChan chan<- error) {
	// Only the top-most function in the call-stack can close these channels
	if parentPath == "" {
		defer close(blobPathChan)
		defer close(errorChan)
	}

	for _, entry := range self.entries {
		// Already seen it?
		if _, ok := sha1sSeen[entry.Sha1()]; ok {
			continue
		}
		sha1sSeen[entry.Sha1()] = true
		if entry.Type() == "blob" {
			blobPath := &BlobPath{
				Blob: entry.blob,
				Path: filepath.Join(parentPath, entry.name),
			}
			blobPathChan <- blobPath
		} else if entry.Type() == "tree" {
			nextPath := filepath.Join(parentPath, entry.name)
			if !entry.tree.instantiated {
				panic("not reached")
			}
			entry.tree._streamBlobPathsUnique(nextPath, repo, sha1sSeen, blobPathChan, errorChan)
		} else {
			panic("cannot reach")
		}
	}
}
