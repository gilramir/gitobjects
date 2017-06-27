package gitobjects

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (s *MySuite) TestFindLooseObjectFiles(c *C) {
	// Create a temp dir
	dir, err := ioutil.TempDir(s.tmpDir, "")
	c.Assert(err, IsNil)
	repoDir := filepath.Join(dir, "repo")

	// Create a git repo
	argv := []string{"init", repoDir}
	cmd := exec.Command("git", argv...)
	cmd.Dir = dir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the GitRepo object
	repo, err := NewGitRepo(repoDir)
	c.Assert(err, IsNil)

	// Touch a file
	readmeFile := filepath.Join(repoDir, "README")
	err = ioutil.WriteFile(readmeFile, []byte{'t', 'e', 's', 't', '\n'}, 0666)
	c.Assert(err, IsNil)

	// Commit it
	cmd = repo.Command([]string{"add", "README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)
	cmd = repo.Command([]string{"commit", "-m", "Add README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Find all loose object files, with a time out in case
	// the goroutine goes crazy

	sha1Chan := make(chan string)
	errorChan := make(chan error)
	go _findLooseObjectFiles(repo.GitDir(), sha1Chan, errorChan)

	timeout := time.NewTimer(time.Duration(3) * time.Second)
	for keepGoing := true; keepGoing; {
		select {
		case <-timeout.C:
			c.Error("Timed out")
			c.FailNow()
		case sha1, ok := <-sha1Chan:
			if !ok {
				keepGoing = false
				break
			}
			output, err := repo.CmdOutput([]string{"cat-file", "-t", sha1})
			c.Assert(err, IsNil)
			if strings.TrimRight(string(output), "\n") == "blob" {
				output, err := repo.CmdOutput([]string{"cat-file", "blob", sha1})
				c.Assert(err, IsNil)
				c.Check(output, DeepEquals, []byte{'t', 'e', 's', 't', '\n'})
			}
		case err = <-errorChan:
			c.Errorf("Received error: %s", err)
		}
	}
	timeout.Stop()
	close(errorChan)
}

func (s *MySuite) TestPackFiles(c *C) {
	// Create a temp dir
	dir, err := ioutil.TempDir(s.tmpDir, "")
	c.Assert(err, IsNil)
	repoDir := filepath.Join(dir, "repo")

	// Create a git repo
	argv := []string{"init", repoDir}
	cmd := exec.Command("git", argv...)
	cmd.Dir = dir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the GitRepo object
	repo, err := NewGitRepo(repoDir)
	c.Assert(err, IsNil)

	// Touch a file
	readmeFile := filepath.Join(repoDir, "README")
	err = ioutil.WriteFile(readmeFile, []byte{'t', 'e', 's', 't', '\n'}, 0666)
	c.Assert(err, IsNil)

	// Commit it
	cmd = repo.Command([]string{"add", "README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)
	cmd = repo.Command([]string{"commit", "-m", "Add README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Modify the same file
	err = ioutil.WriteFile(readmeFile, []byte{'t', 'e', 's', 't', '\n', 'l', 'i', 'n', 'e', '2', '\n'}, 0666)
	c.Assert(err, IsNil)

	// Commit it
	cmd = repo.Command([]string{"add", "README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)
	cmd = repo.Command([]string{"commit", "-m", "Modify README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the pack files
	cmd = repo.Command([]string{"gc"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Find all the pack files, with a time out in case
	// the goroutine goes crazy, and also get the sha1s embedded in them.

	packFileChan := make(chan string)
	sha1Chan := make(chan string)
	errorChan := make(chan error)
	go _findPackFiles(repo.GitDir(), packFileChan)
	go _parsePackFile(repo, packFileChan, sha1Chan, errorChan)

	timeout := time.NewTimer(time.Duration(3) * time.Second)
	foundFirst := false
	foundSecond := false
	for keepGoing := true; keepGoing; {
		select {
		case <-timeout.C:
			c.Error("Timed out")
			c.FailNow()
		case sha1, ok := <-sha1Chan:
			if !ok {
				keepGoing = false
				break
			}
			output, err := repo.CmdOutput([]string{"cat-file", "-t", sha1})
			c.Assert(err, IsNil)
			if strings.TrimRight(string(output), "\n") == "blob" {
				output, err := repo.CmdOutput([]string{"cat-file", "blob", sha1})
				c.Assert(err, IsNil)
				switch string(output) {
				case "test\n":
					foundFirst = true
				case "test\nline2\n":
					foundSecond = true
				default:
					c.Errorf("Wrong content in blob %s: %s", sha1, string(output))
					c.FailNow()
				}
			}
		case err = <-errorChan:
			c.Errorf("Received error: %s", err)
		}
	}
	timeout.Stop()
	close(errorChan)

	c.Check(foundFirst, Equals, true)
	c.Check(foundSecond, Equals, true)
}

func (s *MySuite) TestStreamCommitObjects(c *C) {
	// Create a temp dir
	dir, err := ioutil.TempDir(s.tmpDir, "")
	c.Assert(err, IsNil)
	repoDir := filepath.Join(dir, "repo")

	// Create a git repo
	argv := []string{"init", repoDir}
	cmd := exec.Command("git", argv...)
	cmd.Dir = dir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the GitRepo object
	repo, err := NewGitRepo(repoDir)
	c.Assert(err, IsNil)

	// Touch a file
	readmeFile := filepath.Join(repoDir, "README")
	err = ioutil.WriteFile(readmeFile, []byte{'t', 'e', 's', 't', '\n'}, 0666)
	c.Assert(err, IsNil)

	// Commit it
	cmd = repo.Command([]string{"add", "README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)
	cmd = repo.Command([]string{"commit", "-m", "Add README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Modify the same file
	err = ioutil.WriteFile(readmeFile, []byte{'t', 'e', 's', 't', '\n', 'l', 'i', 'n', 'e', '2', '\n'}, 0666)
	c.Assert(err, IsNil)

	// Commit it
	cmd = repo.Command([]string{"add", "README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)
	cmd = repo.Command([]string{"commit", "-m", "Modify README"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the pack files
	cmd = repo.Command([]string{"gc"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Commit a new file, to create a new loose object
	// Touch a file
	readmeFile = filepath.Join(repoDir, "NOTES")
	err = ioutil.WriteFile(readmeFile, []byte{'t', 'e', 's', 't', '\n'}, 0666)
	c.Assert(err, IsNil)

	// Commit it
	cmd = repo.Command([]string{"add", "NOTES"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)
	cmd = repo.Command([]string{"commit", "-m", "Add NOTES"})
	cmd.Dir = repoDir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Stream the commit objects, with a time out in case
	// the goroutine goes crazy.
	objectChan, errorChan := repo.StreamObjectsOfType("commit")

	timeout := time.NewTimer(time.Duration(3) * time.Second)
	numFound := 0
	for keepGoing := true; keepGoing; {
		select {
		case <-timeout.C:
			c.Error("Timed out")
			c.FailNow()
		case obj, ok := <-objectChan:
			if !ok {
				keepGoing = false
				break
			}
			numFound++
			c.Check(obj.Type(), Equals, "commit")
		case recvErr, ok := <-errorChan:
			if !ok {
				keepGoing = false
				break
			}
			c.Errorf("Received error: %s", recvErr)
		}
	}
	timeout.Stop()

	c.Check(numFound, Equals, 3)
}
