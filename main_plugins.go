// +build go1.8
package main

import (
	"plugin"
	"strings"

	"github.com/aerth/ircb/lib/ircb"
)

// // No C code needed, plugins need CGO_ENABLED=1
import "C"

func init() {
	ircb.LoadPlugin = loadPlugin
}

func loadPlugin(name string) (map[string]ircb.Command, map[string]ircb.Command, error) {
	p, err := plugin.Open(name)
	if err != nil {
		if strings.Contains(err.Error(), "no such") {
			return nil, nil, ircb.ErrNoPlugin
		}
		return nil, nil, err
	}
	println("loading plugin:", name)
	initfn, err := p.Lookup("Init")
	fn, ok := initfn.(func(map[string]ircb.Command, map[string]ircb.Command) (map[string]ircb.Command,
		map[string]ircb.Command, error))
	if !ok {
		return nil, nil, ircb.ErrPluginInv
	}
	return fn(ircb.CommandMap, ircb.MasterMap)
}
