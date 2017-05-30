package ircb

import (
	"github.com/aerth/spawn"
)

func (c *Connection) Respawn() {
	spawn.Spawn()
	c.Close()
}
