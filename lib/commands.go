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
	case 433:
		c.Close()
	}
}
func HandleMasterVerb(c *Connection, irc *IRC) bool {
	const handled = true
	const nothandled = false
	switch irc.Verb {
	default:
		c.Log.Printf("new verb: %q", irc.Verb)
	case "PRIVMSG":
		if irc.ReplyTo != strings.Split(c.config.Master, ":")[0] {
			c.Log.Printf("not master:", irc.ReplyTo)
			return nothandled
		}
		i := strings.Index(c.config.Master, ":")

		if i == -1 {
			c.Log.Println("bad config, not semicolon in Master field")
			return nothandled
		}
		if i >= len(c.config.Master) {
			c.Log.Println("bad config, bad semicolon in Master field")
			return nothandled
		}
		mp := c.config.Master[i+1:]
		if !strings.HasPrefix(irc.Message, mp) {
			c.Log.Println("bad master prefix:", irc.Message)
			return nothandled
		}
		irc.Message = strings.TrimPrefix(irc.Message, mp)
		irc.Command = strings.Split(irc.Message, " ")[0]
		irc.Arguments = strings.Split(strings.TrimPrefix(irc.Message, irc.Command+" "), " ")

		if irc.Command != "" {
			if fn, ok := c.mastermap[irc.Command]; ok {
				c.Log.Printf("master command found: %q", irc.Command)
				fn(c, irc)
				return handled
			}
		}
		c.Log.Printf("master command not found: %q", irc.Command)
		return nothandled

	}
	return nothandled
}
func HandleVerb(c *Connection, irc *IRC) {
	switch irc.Verb {
	default: // unknown string verb
		c.Log.Printf("new verb: %q", irc.Verb)
	case "MODE":
		c.Log.Printf("got mode: %q", irc.Message)
		if !c.joined {
			for _, ch := range strings.Split(c.config.Channels, ",") {
				if ch != "" {
					c.Log.Println("Joining channel:", ch)
					c.Write([]byte(fmt.Sprintf("JOIN %s", ch)))
				}
			}
			c.joined = true

		}
	case "PRIVMSG":
		if strings.Count(irc.Message, " ") == 0 && strings.HasPrefix(irc.To, "#") {
			err := c.ParseKarma(irc.Message)
			if err == nil {
				return
			}
			c.Log.Println(err) // continue maybe is command?
		}
		if irc.Command != "" {
			if fn, ok := c.commandmap[irc.Command]; ok {
				c.Log.Printf("command found: %q", irc.Command)
				fn(c, irc)
				return
			}
			c.Log.Printf("command not found: %q", irc.Command)
		}
		if strings.Contains(irc.Message, "http") {
			c.Log.Println("trying http!")
			c.HandleLinks(irc)
			return
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
	m["karma"] = KarmaShow
	return m
}

func DefaultMasterMap() map[string]Command {
	m := make(map[string]Command)
	m["do"] = CommandMasterDo
	m["upgrade"] = CommandMasterUpgrade
	//	m["voice"] = CommandMasterVoice
	//	m["reload"] = CommandMasterReload
	//	m["macro"] = CommandMasterMacro
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
func CommandMasterReload(c *Connection, irc *IRC) {}
func CommandMasterUpgrade(c *Connection, irc *IRC) {
	update := exec.Command("git", "pull", "origin", "lite")

	out, err := update.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		return
	}
	upgrade := exec.Command("go", "build")

	out, err = upgrade.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		return
	}
	c.Respawn()

}
func CommandMasterMacro(c *Connection, irc *IRC) {}
func CommandMasterQuit(c *Connection, irc *IRC) {
	c.Close()
}
