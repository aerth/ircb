package main

import (
	"bufio"
	"os"
	"strconv"
	"time"

	ircb "github.com/aerth/ircb/lib/ircb"
)

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
func CommandSeen(c *ircb.Connection, irc *ircb.IRC) {
	if len(irc.Arguments) == 0 {
		stat, err := os.Stat(".log.txt")
		if err == nil {
			irc.Reply(c, strconv.FormatUint(uint64(stat.Size), 10))
		}
		return
	}

	if len(irc.Arguments) != 1 {
		return
	}

	f, err := os.Open(".log.txt")
	if err != nil {
		c.SendMaster("%q", err)
		return
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parsed := ircb.Parse(line)
		if parsed.ReplyTo == irc.Arguments[0] {
			irc.ReplyUser(c, parsed.Message)
		}
	}
}

func MasterCommandReloadPlugin(c *ircb.Connection, irc *ircb.IRC) {
	var err error
	ircb.CommandMap, ircb.MasterMap, err = ircb.LoadPlugin("plugin.so")

	if err != nil {
		c.SendMaster(err.Error())
	}
	irc.Reply(c, "plugins reloaded")
}
