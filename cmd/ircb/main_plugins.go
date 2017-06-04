// +build go1.8

package main

import (
	"plugin"
	"strings"

	"github.com/aerth/ircb"
)

// // No C code needed, plugins need CGO_ENABLED=1
import "C"

func init() {
	ircb.LoadPlugin = loadPlugin
}

func loadPlugin(c *ircb.Connection, name string) error {
	p, err := plugin.Open(name)
	if err != nil {
		if strings.Contains(err.Error(), "no such") {
			return ircb.ErrNoPlugin
		}
		return err
	}
	c.Log.Println("loading plugin:", name)
	initfn, err := p.Lookup("Init")
	fn, ok := initfn.(ircb.PluginInitFunc)
	if !ok {
		return ircb.ErrPluginInv
	}
	return fn(c)
}
