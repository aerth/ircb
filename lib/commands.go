package ircb

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/kr/pretty"
)

type Command func(c *Connection, irc *IRC)

func HandleVerbINT(c *Connection, irc *IRC) {
	verb, _ := strconv.Atoi(irc.Verb)
	switch verb {
	default: // unknown numerical verb
		c.Log.Printf("new verb: %q", irc.Verb)

	}
}

func HandleVerb(c *Connection, irc *IRC) {
	switch irc.Verb {
	default: // unknown string verb
		c.Log.Printf("new verb: %q", irc.Verb)
	case "MODE":
		c.Log.Printf("got mode: %q", irc.Message)
	case "PRIVMSG":
		if irc.Command != "" {
			if fn, ok := c.commandmap[irc.Command]; ok {
				c.Log.Printf("command found: %q", irc.Command)
				fn(c, irc)
				return
			}
			c.Log.Printf("command not found: %q", irc.Command)
		}
	}
}

func (irc IRC) String() string {
	return pretty.Sprint(irc)
}

func DefaultCommandMap() map[string]Command {
	m := make(map[string]Command)
	m["up"] = CommandUptime
	m["uptime"] = CommandHostUptime
	m["help"] = CommandHelp
	m["about"] = CommandAbout
	m["lines"] = CommandLineCount
	m["line"] = CommandLine
	m["history"] = CommandHistorySearch
	m["do"] = CommandMasterDo
	m["voice"] = CommandMasterVoice
	m["reload"] = CommandMasterReload
	m["macro"] = CommandMasterMacro
	m["echo"] = CommandEcho
	return m
}

func CommandUptime(c *Connection, irc *IRC) {
	irc.Reply(c, time.Now().Sub(c.since).String())
}
func CommandHostUptime(c *Connection, irc *IRC) {
	uptime := exec.Command("uptime")

	out, err := uptime.CombinedOutput()
	if err != nil {
		c.Log.Println(irc, err)
	}
	output := string(out)
	if output != "" {
		irc.Reply(c, output)
	}
}
func CommandEcho(c *Connection, irc *IRC) {
	irc.Reply(c, fmt.Sprint(irc.Arguments))
}
func CommandHelp(c *Connection, irc *IRC)          {}
func CommandAbout(c *Connection, irc *IRC)         {}
func CommandLineCount(c *Connection, irc *IRC)     {}
func CommandLine(c *Connection, irc *IRC)          {}
func CommandHistorySearch(c *Connection, irc *IRC) {}
func CommandMasterDo(c *Connection, irc *IRC)      {}
func CommandMasterVoice(c *Connection, irc *IRC)   {}
func CommandMasterReload(c *Connection, irc *IRC)  {}
func CommandMasterMacro(c *Connection, irc *IRC)   {}
