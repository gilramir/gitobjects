package gitobjects

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	//	"log"
	"strings"
)

type Commit struct {
	sha1     string
	tree     *Tree
	treeSha1 string

	parentSha1s   []string
	authorLine    string
	committer     string
	committerLine string
	msg           string
}

func (self *Commit) Type() string {
	return "commit"
}

func (self *Commit) Sha1() string {
	return self.sha1
}

func (self *Commit) Instantiate(repo *Repo) error {
	if self.sha1 == "" {
		panic("Instantiate called on Commit that has no sha1")
	}
	output, err := repo.CmdOutput([]string{"cat-file", "-p", self.sha1})
	if err != nil {
		return errors.Wrapf(err, "Calling cat-file commit on %s", self.sha1)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	inHeader := true
	readFirstMessageLine := false
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if inHeader {
			switch fields[0] {
			case "tree":
				self.treeSha1 = fields[1]
			case "parent":
				self.parentSha1s = append(self.parentSha1s, fields[1])
			case "author":
				self.authorLine = scanner.Text()
			case "committer":
				self.committerLine = scanner.Text()
			case "":
				inHeader = false
			}
		} else {
			// The commit message is stored w/o a trailing \n, so we
			// have to carefully reconstruct the message from the split lines,
			// knowing how we split on "\n"'s
			if readFirstMessageLine {
				self.msg += "\n" + scanner.Text()
			} else {
				readFirstMessageLine = true
				self.msg = scanner.Text()
			}
		}
	}

	// Scanner error?
	err = scanner.Err()
	if err != nil {
		return errors.Wrapf(err, "Scanning cat-file commit %s output", self.sha1)
	}

	return nil
}

func (self *Commit) Message() string {
	return self.msg
}

func (self *Commit) Tree() *Tree {
	if self.tree != nil {
		return self.tree
	} else if self.treeSha1 == "" {
		panic(fmt.Sprintf("Commit %s has no tree sha1", self.sha1))
	} else {
		panic(fmt.Sprintf("Commit %s has not had its tree instantiated", self.sha1))
	}
}

func (self *Commit) InstantiateTree(repo *Repo) (*Tree, error) {
	if self.tree != nil {
		panic(fmt.Sprintf("Commit %s tree has already been intantiated", self.sha1))
	} else if self.treeSha1 == "" {
		panic(fmt.Sprintf("Commit %s has no tree sha1", self.sha1))
	}

	tree, has := repo.treeCache.Get(self.treeSha1)
	if has {
		self.tree = tree
		return tree, nil
	}

	self.tree = repo.treeCache.CreateIfNotPresent(self.treeSha1)
	//	log.Printf("Commit %s is instantiating root tree", self.sha1)
	err := self.tree.Instantiate(repo)
	if err != nil {
		return nil, errors.Wrapf(err, "Instantiating Commit %s Tree=%s", self.sha1, self.treeSha1)
	}
	return self.tree, nil
}
