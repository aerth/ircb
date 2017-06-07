package ircb

import (
	"fmt"
)

var ErrNoPluginSupport = fmt.Errorf("no plugin support")
var ErrNoPlugin = fmt.Errorf("plugin not found")
var ErrPluginInv = fmt.Errorf("invalid plugin")

// PluginInitFunc for plugins
type PluginInitFunc (func(c *Connection) error)

// LoadPlugin loads the named plugin file
// This is a stub, and should be replaced if ircb is built with plugin support
var LoadPlugin = func(c *Connection, s string) error {
	return ErrNoPluginSupport
}
