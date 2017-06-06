// +build go1.8,cgo

package main

import "C"

import (
	"os"
	"plugin"
	"strings"

	"github.com/aerth/ircb"
)

func init() {
	ircb.LoadPlugin = loadPlugin
}

func loadPlugin(c *ircb.Connection, name string) error {
	_, err := os.Stat(name)
	if err != nil {
		if strings.Contains(err.Error(), "no such") {
			return ircb.ErrNoPlugin
		}

		return err
	}
	p, err := plugin.Open(name)
	if err != nil {
		return err
	}
	c.Log.Println("loading plugin:", name)
	initfn, err := p.Lookup("Init")
	fn, ok := initfn.(func(c *ircb.Connection) error)
	if !ok {
		return ircb.ErrPluginInv
	}
	return fn(c)
}
