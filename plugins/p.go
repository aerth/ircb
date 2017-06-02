package main

import ircb "github.com/aerth/ircb/lib/ircb"

func main() {
	println("plugin main")
}
func Init(c map[string]ircb.Command, m map[string]ircb.Command) error {
	c["test"] = CommandTest
	m["update-plugins"] = MasterCommandReloadPlugin
	return nil
}

func CommandTest(c *ircb.Connection, irc *ircb.IRC) {
	irc.Reply(c, "plugins work")
}

func MasterCommandReloadPlugin(c *ircb.Connection, irc *ircb.IRC) {
	err := ircb.LoadPlugin("plugin.so")
	if err != nil {
		c.SendMaster(err.Error())
	}
	irc.Reply(c, "plugins reloaded")
}
