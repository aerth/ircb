package ircb

import (
	"github.com/aerth/spawn"
)

// Respawn closes connections after executing self, can be called at any time.
func (c *Connection) Respawn() {
	spawn.Spawn()
	c.Close()
}
