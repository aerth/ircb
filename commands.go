package ircb

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const handled = true
const nothandled = false

// Command is what executes using a parsed IRC message
// irc message '!echo arg1 arg2' gets parsed as:
// 	'irc.Command = echo', 'irc.CommandArguments = []string{"arg1","arg2"}
// Command will be executed if it is in CommandMap or MasterMap
// Map Commands before connecting:
//	ircb.CommandMap
// 	ircb.DefaultCommandMaps() // load defaults, optional.
// 	ircb.CommandMap["echo"] = CommandEcho
//	// Add a new command called hello, executed with !hello
// !hello responds in channel, using name of user commander
//	ircb.CommandMap["hello"] = func(c *Connection, irc *IRC){
//		irc.Reply(c, fmt.Sprintf("hello, %s!", irc.ReplyTo))
//		}
//	// Command parser will deal with authentication.
//	// This makes adding new master commands easy:
//	ircb.MasterMap["stat"] = func(c *Connection, irc.*IRC){
//		irc.ReplyUser(c, fmt.Sprintf("lines received: %v", c.lines))
//	}
// Reply with irc.ReplyUser (for /msg reply) or irc.Reply (for channel)
type Command func(c *Connection, irc *IRC)

func nilcommand(c *Connection, irc *IRC) {
	c.Log.Println("nil command ran successfully")
}

func verbIntHandler(c *Connection, irc *IRC) bool {
	verb, _ := strconv.Atoi(irc.Verb)
	switch verb {
	default: // unknown numerical verb
		c.Log.Printf("new verb: %q", irc.Verb)
	case 372, 1, 2, 3, 4, 5, 6, 7, 0:
		return handled
	case 221:
		c.Log.Printf("new mode: %q", irc.Message)
	case 433:

		c.Close()
	}
	return nothandled
}

func privmsgMasterHandler(c *Connection, irc *IRC) bool {
	if irc.ReplyTo != strings.Split(c.config.Master, ":")[0] {
		c.Log.Printf("not master: %s", irc.ReplyTo)
		return nothandled
	}

	if time.Now().Sub(c.masterauth) > 5*time.Minute {
		c.Log.Println("bad master, need reauth")
		c.MasterCheck()
		return nothandled
	}
	c.Log.Println("got master message, parsing...")
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

		// switch prefix
		if irc.Command == c.config.CommandPrefix && len(irc.Arguments) == 1 {
			c.config.CommandPrefix = irc.Arguments[0]
			c.Log.Printf("**New command prefix: %q", c.config.CommandPrefix)
			c.SendMaster("**New command prefix: %q", c.config.CommandPrefix)
			return handled
		}
		if c.config.Verbose {
			c.Log.Println("not master command prefixed")
		}
		return nothandled
	}
	irc.Message = strings.TrimPrefix(irc.Message, mp)

	irc.Command = strings.Split(irc.Message, " ")[0]
	irc.Arguments = strings.Split(strings.TrimPrefix(irc.Message, irc.Command+" "), " ")
	c.Log.Printf("master command request: %s %s", irc.Command, irc.Arguments)
	if irc.Command != "" {
		c.Log.Println("trying master command:", irc.Command)
		if fn, ok := c.MasterMap[irc.Command]; ok {
			c.Log.Printf("master command found: %q", irc.Command)
			fn(c, irc)
			return handled
		}
	}
	c.Log.Printf("master command not found: %q", irc.Command)
	return nothandled

}

func (c *Connection) GetPublicCommand(word string) Command {
	fn, ok := c.CommandMap[word]
	if !ok {
		return nilcommand
	}
	return fn
}

func (c *Connection) IsPublicCommand(word string) bool {
	_, ok := c.CommandMap[word]
	return ok
}
func privmsgHandler(c *Connection, irc *IRC) {

	// is karma
	if strings.HasPrefix(irc.To, "#") && c.ParseKarma(irc.Message) {
		return

	}

	if irc.Command != "" {
		if fn, ok := c.CommandMap[irc.Command]; ok {
			c.Log.Printf("command found: %q", irc.Command)
			fn(c, irc)
			return
		}
	}

	// handle channel defined definitions
	if irc.Command != "" && len(irc.Arguments) == 0 {
		definition := c.getDefinition(irc.Command)
		if definition != "" {
			irc.Reply(c, definition)
			return
		}

	}

	if irc.Command != "" {
		c.Log.Printf("command not found: %q", irc.Command)
	}
	// try to parse http link title
	if c.config.ParseLinks && strings.Contains(irc.Message, "http") {
		go c.HandleLinks(irc)
	}

}

func DefaultCommandMap() map[string]Command {
	m := make(map[string]Command)
	m["quiet"] = CommandQuiet
	m["up"] = CommandUptime
	m["uptime"] = CommandHostUptime
	m["help"] = CommandHelp
	m["about"] = CommandAbout
	m["echo"] = CommandEcho
	m["karma"] = KarmaShow
	m["define"] = CommandDefine
	return m
}

func DefaultMasterMap() map[string]Command {
	m := make(map[string]Command)
	m["do"] = CommandMasterDo
	m["upgrade"] = CommandMasterUpgrade
	m["q"] = CommandMasterQuit
	m["quit"] = CommandMasterQuit
	m["set"] = CommandMasterSet
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
		c.SendMaster("%s", err)
	}

	output := strings.Split(string(out), "\n")[0]
	if strings.TrimSpace(output) != "" {
		irc.Reply(c, output)
	}
}
func CommandEcho(c *Connection, irc *IRC) {
	irc.Reply(c, fmt.Sprint(strings.Join(irc.Arguments, " ")))
}
func CommandQuiet(c *Connection, irc *IRC) {
	c.quiet = !c.quiet
	if !c.quiet {
		c.Log.Println("no longer quiet")
		irc.Reply(c, "\x01ACTION gasps for air\x01")
	}
	c.Log.Println("muted")
}
func CommandHelp(c *Connection, irc *IRC) {
	if len(irc.Arguments) < 2 || irc.Arguments[0] == "" {
		var commandlist string
		for i := range c.CommandMap {
			commandlist += i + " "
		}
		irc.Reply(c, "commands: "+commandlist)
		return
	}
}

func CommandAbout(c *Connection, irc *IRC) {
	irc.Reply(c, "I'm a robot. You can learn more at https://aerth.github.io/ircb/")
}
func CommandLineCount(c *Connection, irc *IRC) {}
func CommandDefine(c *Connection, irc *IRC) {
	if !c.config.Define {
		return
	}
	if len(irc.Arguments) < 2 || irc.Arguments[0] == "" {
		irc.Reply(c, "usage: define [word] [text]")
		return
	}
	action := irc.Arguments[0]
	if _, ok := c.CommandMap[action]; ok {
		irc.Reply(c, fmt.Sprintf("already defined as command: %q", action))
		return
	}
	definition := strings.Join(irc.Arguments[1:], " ")

	c.DatabaseDefine(action, definition)
	irc.Reply(c, fmt.Sprintf("defined: %q", action))

}
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
func CommandMasterSet(c *Connection, irc *IRC) {
	if len(irc.Arguments) != 2 {
		irc.Reply(c, `usage: set optionname on|off`)
		return
	}
	option := irc.Arguments[0]
	value := irc.Arguments[1]
	switch option {
	default:
		irc.Reply(c, `no option like that`)
		return
	case "links":
		switch value {
		case "on":
			c.config.ParseLinks = true
		case "off":
			c.config.ParseLinks = false
		default:
			irc.Reply(c, `usage: set optionname on|off`)
			return
		}
	case "define":
		switch value {
		case "on":
			c.config.Define = true
		case "off":
			c.config.Define = false
		default:
			irc.Reply(c, `usage: set optionname on|off`)
			return
		}

	case "history":
		switch value {
		case "on":
			c.config.History = true
		case "off":
			c.config.History = false
		default:
			irc.Reply(c, `usage: set optionname on|off`)
			return
		}

	}

}

func CommandMasterUpgrade(c *Connection, irc *IRC) {
	checkout := exec.Command("git", "checkout", "master")
	out, err := checkout.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		return
	}
	update := exec.Command("git", "pull", "origin", "master")

	out, err = update.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		return
	}
	upgrade := exec.Command("make")

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
