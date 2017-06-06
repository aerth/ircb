package main

import (
	"bufio"
	"os"
	"strconv"
	"time"

	"github.com/aerth/ircb"
)

func main() {}

func Init(c *ircb.Connection) error {
	c.CommandAdd("time", CommandTime)
	c.CommandAdd("time-baz", CommandTime)
	c.CommandAdd("time-boo", CommandTime)
	c.CommandMasterAdd("update-plugins", MasterCommandReloadPlugin)
	return nil
}

func CommandTime(c *ircb.Connection, irc *ircb.IRC) {
	irc.Reply(c, time.Now().String())
}
func CommandSeen(c *ircb.Connection, irc *ircb.IRC) {
	if len(irc.Arguments) == 0 {
		stat, err := os.Stat(".log.txt")
		if err == nil {
			irc.Reply(c, strconv.FormatUint(uint64(stat.Size()), 10))
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
	err := ircb.LoadPlugin(c, "plugin.so")
	if err != nil {
		c.SendMaster("error reloading plugins: %v", err)
	}
	irc.Reply(c, "plugins reloaded")
}
