package ircb

import (
	"fmt"
	"os"

	"github.com/aerth/spawn"
)

const (
	Green     = "\x033"
	Red       = "\x035"
	Purple    = "\x036"
	Yellow    = "\x038"
	GreenBold = "\x039"
)

// Respawn closes connections after executing self, can be called at any time.
func (c *Connection) Respawn() {
	spawn.Spawn()
	c.Close()
}

// ErrNoPluginSupport when compiled with no CGO or without 'plugins' tag
var ErrNoPluginSupport = fmt.Errorf("no plugin support")

// ErrNoPlugin when plugin is not found
var ErrNoPlugin = fmt.Errorf("plugin not found")

// ErrPluginInv when plugin does not have proper Init func
var ErrPluginInv = fmt.Errorf("invalid plugin")

// PluginInitFunc gets called when plugin is loaded. Init(c *Connection) error
type PluginInitFunc (func(c *Connection) error)

// LoadPlugin loads the named plugin file
// This is a stub, and should be replaced if ircb is built with plugin support
var LoadPlugin = func(c *Connection, s string) error {
	return ErrNoPluginSupport
}

func openlogfile() (f *os.File, err error) {
	return os.OpenFile(".log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
}
