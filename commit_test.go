package gitobjects

import (
	"context"
	. "gopkg.in/check.v1"
	"time"
)

func (s *MySuite) TestCommitInstantiate(c *C) {
	repo, _ := s.setupRepoWithReadme(c)

	// Stream the commit objects, with a time out in case the goroutine goes crazy.
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	objectChan, errorChan := repo.StreamObjectsOfType(ctx, "commit")

	var commitObj *Commit

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
			if commitObj == nil {
				commitObj = obj.(*Commit)
				keepGoing = false
				break
			} else {
				c.Errorf("Received sha1=%s but already have sha1=%s",
					obj.Sha1(), commitObj.Sha1())
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

	err := commitObj.Instantiate(repo)
	c.Assert(err, IsNil)

	c.Check(commitObj.Message(), Equals, "Add README")
}
