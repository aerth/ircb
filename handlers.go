package ircb

import (
	"strconv"
	"strings"
	"time"
)

func verbIntHandler(c *Connection, irc *IRC) bool {
	verb, _ := strconv.Atoi(irc.Verb)
	switch verb {
	default: // unknown numerical verb
		c.Log.Printf("%s %q", irc.Verb, irc.Message)
		return handled
	case 331:
		c.Log.Printf("%s TOPIC: %q", irc.Raw, "")
		return handled
	case 353:
		c.Log.Printf("%s USER LIST: %q", irc.Raw, "")
		return handled
	case 372, 1, 2, 3, 4, 5, 6, 7, 0, 366:
		return handled
	case 221:
		c.Log.Printf("UMODE: %q", irc.Message)
		return handled
	case 433:

		c.Close()
		return handled
	}

}

func privmsgMasterHandler(c *Connection, irc *IRC) bool {
	if irc.ReplyTo != strings.Split(c.config.Master, ":")[0] {
		c.Log.Printf("not master: %s", irc.ReplyTo)
		return nothandled
	}

	if dur := time.Now().Sub(c.masterauth); dur > 5*time.Minute {
		c.Log.Println("need reauth after", dur)
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

	irc.Command = strings.TrimSpace(strings.Split(irc.Message, " ")[0])
	args := strings.Split(strings.TrimPrefix(irc.Message, irc.Command), " ")
	for _, v := range args {
		if strings.TrimSpace(v) == "" {
			continue
		}
		irc.Arguments = append(irc.Arguments, v)
	}
	c.Log.Printf("master command request: %s, %v args)", irc.Command, len(irc.Arguments))
	if c.config.Verbose {
		c.Log.Printf("master command request: %s", irc)
	}
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

func privmsgHandler(c *Connection, irc *IRC) {

	// is karma
	if strings.HasPrefix(irc.To, "#") && c.parseKarma(irc.Message) {
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
		go c.linkhandler(irc)
	}

}
