package gitobjects

import (
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

type Blob struct {
	sha1 string
}

func (self *Blob) Type() string {
	return "blob"
}

func (self *Blob) Sha1() string {
	return self.sha1
}

func (self *Blob) Instantiate(repo *Repo) error {
	return nil
}

func (self *Blob) DecompressedSizeBytes(repo *Repo) (int, error) {
	output, err := repo.CmdOutput([]string{"cat-file", "-s", self.sha1})
	if err != nil {
		return 0, errors.Wrapf(err, "Getting decompressed size for blob %s", self.sha1)
	}
	sizeString := strings.TrimRight(string(output), "\n")
	size, err := strconv.Atoi(sizeString)
	if err != nil {
		return 0, errors.Wrapf(err, "Converting decompressed size '%s' for blob %s", sizeString, self.sha1)
	}
	return size, nil
}
