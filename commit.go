package gitobjects

type Commit struct {
	sha1 string
}

func (self *Commit) Type() string {
	return "commit"
}

func (self *Commit) Sha1() string {
	return self.sha1
}
