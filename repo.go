package gitobjects

import (
	"github.com/pkg/errors"
	//	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitRepo struct {
	gitDir string
}

func NewGitRepo(directory string) (*GitRepo, error) {
	var gitDir string

	// empty tring == CWD
	if directory == "" {
		directory = "."
	}

	// In newer versions of git
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = directory
	output, err := cmd.Output()
	if err == nil {
		gitDir = strings.TrimRight(string(output), "\n")
	} else {
		// In older versions of git
		cmd = exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = directory
		output, err := cmd.Output()
		if err != nil {
			return nil, errors.Wrapf(err, "Finding git dir for %s", directory)
		}
		gitDir = strings.TrimRight(string(output), "\n")
	}

	if !filepath.IsAbs(gitDir) {
		absDirectory, err := filepath.Abs(directory)
		if err != nil {
			return nil, errors.Wrapf(err, "Finding absolute path for %s", directory)
		}
		gitDir = filepath.Join(absDirectory, gitDir)
	}

	return &GitRepo{
		gitDir: gitDir,
	}, nil
}

func (self *GitRepo) GitDir() string {
	return self.gitDir
}

func (self *GitRepo) Command(cmdv []string) *exec.Cmd {
	if len(cmdv) == 0 {
		panic("Empty cmdv")
	}

	cmd := exec.Command("git", cmdv...)
	cmd.Dir = self.gitDir
	//	cmd.Stdout = os.Stdout
	//	cmd.Stderr = os.Stderr
	return cmd
}

func (self *GitRepo) Run(cmdv []string) error {
	cmd := self.Command(cmdv)
	return cmd.Run()
}

func (self *GitRepo) CmdOutput(cmdv []string) ([]byte, error) {
	cmd := self.Command(cmdv)
	cmd.Stdout = nil
	return cmd.Output()
}

func (self *GitRepo) CmdCombinedOutput(cmdv []string) ([]byte, error) {
	cmd := self.Command(cmdv)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.CombinedOutput()
}
