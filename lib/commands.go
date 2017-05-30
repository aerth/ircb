package ircb

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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
func HandleMasterVerb(c *Connection, irc *IRC) {
	defer c.Log.Println("Got master command:", irc)
	switch irc.Verb {
	default:
		c.Log.Printf("new verb: %q", irc.Verb)
	case "PRIVMSG":
		if irc.ReplyTo != strings.Split(c.config.Master, ":")[0] {
			c.Log.Printf("not master:", irc.ReplyTo)
			return
		}
		i := strings.Index(c.config.Master, ":")

		if i == -1 {
			c.Log.Println("bad config, not semicolon in Master field")
			return
		}
		if i >= len(c.config.Master) {
			c.Log.Println("bad config, bad semicolon in Master field")
			return
		}
		mp := c.config.Master[i+1:]
		if !strings.HasPrefix(irc.Message, mp) {
			c.Log.Println("bad master prefix:", irc.Message)
			return
		}
		irc.Message = strings.TrimPrefix(irc.Message, mp)
		irc.Command = strings.Split(irc.Message, " ")[0]
		irc.Arguments = strings.Split(strings.TrimPrefix(irc.Message, irc.Command+" "), " ")

		if irc.Command != "" {
			if fn, ok := c.mastermap[irc.Command]; ok {
				c.Log.Printf("master command found: %q", irc.Command)
				fn(c, irc)
				return
			}
		}
		c.Log.Printf("master command not found: %q", irc.Command)

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
	m["echo"] = CommandEcho
	return m
}

func DefaultMasterMap() map[string]Command {
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
	// output of uptime command
	uptime := exec.Command("/usr/bin/uptime")

	out, err := uptime.CombinedOutput()
	if err != nil {
		c.Log.Println(irc, err)
	}
	output := strings.Split(string(out), "\n")[0]
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
func CommandMasterDo(c *Connection, irc *IRC) {
	c.Log.Println("GOT DO:", irc)
	c.Write([]byte(strings.Join(irc.Arguments, " ")))
}
func CommandMasterVoice(c *Connection, irc *IRC) {}
func CommandMasterDebug(c *Connection, irc *IRC) {
	c.Log.Println(c, irc)
}
func CommandMasterMacro(c *Connection, irc *IRC)  {}
func CommandMasterReload(c *Connection, irc *IRC) {}
