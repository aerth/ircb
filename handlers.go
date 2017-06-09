package ircb

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func verbIntHandler(c *Connection, irc *IRC) bool {
	verb, err := strconv.Atoi(irc.Verb)
	if err != nil {
		return nothandled
	}
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

// handle anything from master, returning false if message has not been handled
func privmsgMasterHandler(c *Connection, irc *IRC) bool {
	if irc.ReplyTo != strings.Split(c.config.Master, ":")[0] {
		c.Log.Printf("not master: %s", irc.ReplyTo)
		return nothandled
	}

	if dur := time.Now().Sub(c.masterauth); dur > 5*time.Minute {
		c.Log.Println("need reauth after", dur)
		c.MasterCheck()
		defer c.SendMaster("you are now authenticated for 5 minutes")
		return privmsgMasterHandler(c, irc)
	}
	i := strings.Index(c.config.Master, ":")
	if i == -1 {
		c.Log.Println("*** bad config, not semicolon in Master field")
		return nothandled
	}
	if i >= len(c.config.Master) {
		c.Log.Println("*** bad config, bad semicolon in Master field")
		return nothandled
	}

	mp := c.config.Master[i+1:] // master prefix
	if !strings.HasPrefix(irc.Message, mp) {
		// switch prefix
		if irc.Command == c.config.CommandPrefix && len(irc.Arguments) == 1 {
			c.config.CommandPrefix = irc.Arguments[0]
			c.Log.Printf("**New command prefix: %q", c.config.CommandPrefix)
			c.SendMaster("**New command prefix: %q", c.config.CommandPrefix)
			return handled
		}

		// was not a prefix switch
		// just master sending messages or normal commands
		return nothandled
	}
	// re-parse for master command
	irc.Message = strings.TrimPrefix(irc.Message, mp)
	irc.Command = strings.TrimSpace(strings.Split(irc.Message, " ")[0])
	args := strings.Split(strings.TrimPrefix(irc.Message, irc.Command), " ")
	for _, v := range args {
		if strings.TrimSpace(v) != "" {
			irc.Arguments = append(irc.Arguments, v)
		}
	}
	if c.config.Verbose {
		c.Log.Printf("master command parsed: %s", irc)
	}
	if irc.Command != "" {
		if fn, ok := c.MasterMap[irc.Command]; ok {
			c.Log.Printf("master command found: %q", irc.Command)
			fn(c, irc)
			return handled
		}
		c.SendMaster("master command not found")
	}
	return nothandled

}

// handle any PRIVMSG, should go *after* privmsgMasterHandler and verbintHandler
func privmsgHandler(c *Connection, irc *IRC) bool {

	// is karma, sent to a channel (not /msg)
	if strings.HasPrefix(irc.To, "#") && c.parseKarma(irc.Message) {
		return handled

	}

	// is parsed as command
	if irc.Command != "" {
		if fn, ok := c.CommandMap[irc.Command]; ok {
			c.Log.Printf("command found: %q", irc.Command)
			fn(c, irc)
			return handled
		}
	}

	// handle channel defined definitions
	if irc.Command != "" {
		definition := c.getDefinition(irc.Command)
		if definition != "" {
			irc.Reply(c, definition)
			return handled
		}

		c.Log.Printf("command not found: %q", irc.Command)
		irc.ReplyUser(c, "command not found. try the 'help' command")
	}
	// try to parse http link title
	if c.config.ParseLinks && strings.Contains(irc.Message, "http") {
		if c.linkhandler(irc) {
			return handled
		}
	}

	return nothandled

}

// linkhandler replies to messages with http links
func (c *Connection) linkhandler(irc *IRC) bool {
	if !c.config.ParseLinks {
		return nothandled
	}
	// word starts with http and is url parsable
	i := strings.Index(irc.Message, "http")
	if i == -1 {
		// no links (already checked)
		return nothandled
	}
	s := irc.Message[i:]
	ss := strings.Split(s, " ")[0]
	_, err := url.Parse(ss)
	if err != nil {
		c.Log.Println("error parsing url:", err)
		c.SendMaster("error parsing url: %v", err)
		return nothandled
	}

	if strings.Contains(ss, "localhost") || strings.Contains(ss, "::") {
		c.Log.Println("bad url:", ss)
		c.SendMaster("bad url %q from %q", ss, irc.ReplyTo)
		return handled
	}
	req, err := http.NewRequest(http.MethodGet, ss, nil)
	if err != nil {
		c.Log.Println("error making request:", err)
		return handled
	}

	c.Log.Println("sending http request:", ss)
	defer c.Log.Println("done handling link %q in %q", ss, irc.ReplyTo)
	t1 := time.Now()
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Log.Println("error getting url:", ss, err)
		return handled
	}
	if resp.StatusCode != 200 {
		// reply error
		irc.Reply(c, fmt.Sprintf("%s %s", resp.Status, time.Now().Sub(t1)))
		return handled
	}
	defer resp.Body.Close()
	reader := io.LimitReader(resp.Body, 512)
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		c.Log.Println("error reading from reader:", err)
		// but still reply with response time
		irc.Reply(c, fmt.Sprintf("%s %s (%s)", resp.Status, time.Now().Sub(t1), "read error"))
		return handled
	}
	meta := getLinkTitleFromHTML(b)
	if meta.Title != "" {
		irc.Reply(c, fmt.Sprintf("%s %s %q (%s)", resp.Status, time.Now().Sub(t1), meta.Title, meta.ContentType))
		return handled
	}
	irc.Reply(c, fmt.Sprintf("%s %s (%s)", resp.Status, time.Now().Sub(t1), meta.ContentType))
	return handled
}
