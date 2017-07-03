package gitobjects

import (
	"context"
	. "gopkg.in/check.v1"
	"time"
)

func (s *MySuite) TestTreeInstantiate(c *C) {
	repo, _ := s.setupRepoWithReadme(c)

	// Stream the commit objects, with a time out in case the goroutine goes crazy.
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	objectChan, errorChan := repo.StreamObjectsOfType(ctx, "tree", 1)

	var treeObj *Tree

	for keepGoing := true; keepGoing; {
		select {
		case <-ctx.Done():
			c.Error("Timed out")
			c.FailNow()
		case obj, ok := <-objectChan:
			if !ok {
				keepGoing = false
				break
			}
			if treeObj == nil {
				treeObj = obj.(*Tree)
				keepGoing = false
				break
			} else {
				c.Errorf("Received sha1=%s but already have sha1=%s",
					obj.Sha1(), treeObj.Sha1())
				c.FailNow()
			}
		case recvErr, ok := <-errorChan:
			if !ok {
				keepGoing = false
				break
			}
			c.Errorf("Received error: %s", recvErr)
		}
	}

	err := treeObj.Instantiate(repo)
	c.Assert(err, IsNil)

	c.Assert(len(treeObj.entries), Equals, 1)
	entry := treeObj.entries[0]
	c.Check(entry.Type(), Equals, "blob")
	c.Check(entry.Name(), Equals, "README")
}

func (s *MySuite) TestStreamBlobPathsUnique(c *C) {
	repo, _ := s.setupRepoWithReadme(c)

	// Get each commit
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	objectChan, errorChan := repo.StreamObjectsOfType(ctx, "commit", 1)

	sha1sSeen := make(map[string]bool)
	blobPaths := make([]*BlobPath, 0, 1)

	for keepGoing := true; keepGoing; {
		select {
		case <-ctx.Done():
			c.Error("Timed out")
			c.FailNow()
		case obj, ok := <-objectChan:
			if !ok {
				keepGoing = false
				break
			}
			var commit *Commit
			commit = obj.(*Commit)

			// Stream each blobPath
			//tree, err := commit.InstantiateTree(repo)
			//c.Assert(err, IsNil)
			blobPathChan, error2Chan := commit.tree.StreamBlobPathsUnique(repo, sha1sSeen)
			for blobPath := range blobPathChan {
				blobPaths = append(blobPaths, blobPath)
			}
			err2 := <-error2Chan
			c.Assert(err2, IsNil)

		case recvErr, ok := <-errorChan:
			if !ok {
				keepGoing = false
				break
			}
			c.Errorf("Received error: %s", recvErr)
		}
	}

	c.Assert(len(blobPaths), Equals, 1)
	c.Check(blobPaths[0].Path, Equals, "README")
}
