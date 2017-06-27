package gitobjects

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	//	"log"
	"os/exec"
	"path/filepath"
)

func (s *MySuite) TestRepoBareNoSuffix(c *C) {
	// Create a temp dir
	dir, err := ioutil.TempDir(s.tmpDir, "")
	c.Assert(err, IsNil)

	// Create a bare git repo with no ".git" suffix
	repoDir := filepath.Join(dir, "bare-repo-no-suffix")
	argv := []string{"init", "--bare", repoDir}

	cmd := exec.Command("git", argv...)
	cmd.Dir = dir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the GitRepo object
	repo, err := NewGitRepo(repoDir)
	c.Assert(err, IsNil)

	// The repoDir *is* the git dir
	c.Check(repo.GitDir(), Equals, repoDir)
}

func (s *MySuite) TestRepoBareWithSuffix(c *C) {
	// Create a temp dir
	dir, err := ioutil.TempDir(s.tmpDir, "")
	c.Assert(err, IsNil)

	// Create a bare git repo with a ".git" suffix
	repoDir := filepath.Join(dir, "bare-repo-with-suffix.git")
	argv := []string{"init", "--bare", repoDir}

	cmd := exec.Command("git", argv...)
	cmd.Dir = dir
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Create the GitRepo object
	repo, err := NewGitRepo(repoDir)
	c.Assert(err, IsNil)

	// The repoDir *is* the git dir
	c.Check(repo.GitDir(), Equals, repoDir)
}

func (s *MySuite) TestRepoWorkspace(c *C) {
	// Create a temp dir
	dir, err := ioutil.TempDir(s.tmpDir, "")
	c.Assert(err, IsNil)

	// Create a workspace git repo
	repoDir := filepath.Join(dir, "workspace")
	argv := []string{"init", repoDir}

	cmd := exec.Command("git", argv...)
	cmd.Dir = dir
	err = cmd.Run()
	c.Assert(err, IsNil)

	repoGitDir := filepath.Join(repoDir, ".git")

	// Create the GitRepo object
	repo, err := NewGitRepo(repoDir)
	c.Assert(err, IsNil)

	// The gitDir is ${workspaceDir}/.git
	c.Check(repo.GitDir(), Equals, repoGitDir)

	// Create another GitRepo object, pointing to the git dir
	repo2, err := NewGitRepo(repoGitDir)
	c.Assert(err, IsNil)

	// The gitDir is ${workspaceDir}/.git
	c.Check(repo2.GitDir(), Equals, repoGitDir)
}
