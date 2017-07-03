package gitobjects

import (
	. "gopkg.in/check.v1"
)

func (s *MySuite) TestTreeCache(c *C) {

	cache := NewTreeCache()

	c.Check(cache.Has("foo"), Equals, false)

	xTree := &Tree{
		sha1: "123",
	}
	cache.Set("x", xTree)
	c.Check(cache.Has("x"), Equals, true)

	retrievedTree, has := cache.Get("x")
	c.Check(retrievedTree, Equals, xTree)
	c.Check(has, Equals, true)

	retrievedTree, has = cache.Get("y")
	c.Check(has, Equals, false)
	c.Check(retrievedTree, IsNil)
}
