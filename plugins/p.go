package main

import (
	"time"

	ircb "github.com/aerth/ircb/lib/ircb"
)

func main() {
	println("plugin main")
}
func Init(c map[string]ircb.Command, m map[string]ircb.Command) (map[string]ircb.Command, map[string]ircb.Command, error) {
	if c == nil {
		c = make(map[string]ircb.Command)

	}

	if m == nil {
		m = make(map[string]ircb.Command)
	}
	c["time"] = CommandTime
	m["update-plugins"] = MasterCommandReloadPlugin
	return c, m, nil
}

func CommandTime(c *ircb.Connection, irc *ircb.IRC) {
	irc.Reply(c, time.Now().String())
}

func MasterCommandReloadPlugin(c *ircb.Connection, irc *ircb.IRC) {
	err := ircb.LoadPlugin("plugin.so")
	if err != nil {
		c.SendMaster(err.Error())
	}
	irc.Reply(c, "plugins reloaded")
}
