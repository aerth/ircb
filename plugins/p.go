package main

import (
	"time"

	"github.com/aerth/ircb"
)

var pluginversion = "ircb plugin 0.2"

func main() {}

func Init(c *ircb.Connection) error {
	c.CommandAdd("time-foo", CommandTime)
	c.CommandAdd("plugin", CommandVersion)
	return nil
}

func CommandTime(c *ircb.Connection, irc *ircb.IRC) {
	irc.Reply(c, time.Now().String())
}

func CommandVersion(c *ircb.Connection, irc *ircb.IRC) {
	irc.Reply(c, pluginversion)
}
