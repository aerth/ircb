package ircb

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"sort"
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

// AddMasterCommand adds a new master command, named 'name' to the MasterMap
func (c *Connection) AddMasterCommand(name string, fn Command) {
	c.maplock.Lock()
	defer c.maplock.Unlock()
	c.MasterMap[name] = fn
	return
}

// AddCommand adds a new public command, named 'name' to the CommandMap
func (c *Connection) AddCommand(name string, fn Command) {
	c.maplock.Lock()
	defer c.maplock.Unlock()
	c.CommandMap[name] = fn
}

// AddCommand adds a new public command, named 'name' to the CommandMap
func (c *Connection) RemoveMasterCommand(name string) {
	c.maplock.Lock()
	defer c.maplock.Unlock()
	delete(c.MasterMap, name)
}

// RemoveCommand removes a public command, named 'name' to the CommandMap
func (c *Connection) RemoveCommand(name string) {
	c.maplock.Lock()
	defer c.maplock.Unlock()
	delete(c.CommandMap, name)
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
		go c.HandleLinks(irc)
	}

}

func DefaultCommandMap() map[string]Command {
	m := make(map[string]Command)
	m["quiet"] = commandQuiet
	m["up"] = commandUptime
	m["uptime"] = commandHostUptime
	m["help"] = commandHelp
	m["about"] = commandAbout
	m["echo"] = commandEcho
	m["karma"] = commandKarma
	m["define"] = commandDefine
	return m
}

func DefaultMasterMap() map[string]Command {
	m := make(map[string]Command)
	m["do"] = commandMasterDo
	m["upgrade"] = commandMasterUpgrade
	m["r"] = commandMasterReboot
	m["part"] = commandMasterPart
	m["quit"] = commandMasterQuit
	m["q"] = commandMasterQuit
	m["quit"] = commandMasterQuit
	m["help"] = commandMasterHelp
	m["set"] = commandMasterSet
	m["plugin"] = masterCommandLoadPlugin
	m["fetch"] = masterCommandFetchPlugin
	return m
}

func commandUptime(c *Connection, irc *IRC) {
	irc.Reply(c, time.Now().Sub(c.since).String())
}
func commandHostUptime(c *Connection, irc *IRC) {
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
func commandEcho(c *Connection, irc *IRC) {
	irc.Reply(c, fmt.Sprint(strings.Join(irc.Arguments, " ")))
}
func commandQuiet(c *Connection, irc *IRC) {
	c.quiet = !c.quiet
	if !c.quiet {
		c.Log.Println("no longer quiet")
		irc.Reply(c, "\x01ACTION gasps for air\x01")
	}
	c.Log.Println("muted")
}
func commandMasterHelp(c *Connection, irc *IRC) {
	if len(irc.Arguments) < 2 || irc.Arguments[0] == "" {
		var list []string
		for i := range c.MasterMap {
			list = append(list, i)
		}
		sort.Strings(list)
		irc.Reply(c, fmt.Sprintf("%v master commands: %s", len(list), list))
		return
	}

}
func commandHelp(c *Connection, irc *IRC) {
	if len(irc.Arguments) < 2 || irc.Arguments[0] == "" {
		var list []string
		for i := range c.CommandMap {
			list = append(list, i)
		}
		sort.Strings(list)
		irc.Reply(c, fmt.Sprintf("%v commands: %s", len(list), list))
		return
	}
}

func commandAbout(c *Connection, irc *IRC) {
	irc.Reply(c, "I'm a robot. You can learn more at https://aerth.github.io/ircb/")
}
func commandLineCount(c *Connection, irc *IRC) {}
func commandDefine(c *Connection, irc *IRC) {
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
func commandMasterDo(c *Connection, irc *IRC) {
	c.Log.Println("GOT DO:", irc)
	c.Write([]byte(strings.Join(irc.Arguments, " ")))
}
func commandMasterReboot(c *Connection, irc *IRC) {
	b, err := c.MarshalConfig()
	if err != nil {
		c.Log.Printf("error while trying to reboot: %v", err)
		irc.Reply(c, "cant reboot, check logs")
		return
	}
	err = ioutil.WriteFile("config.json", b, 0600)
	if err != nil {
		c.Log.Printf("error while trying to write config file for respawn: %v", err)
		irc.Reply(c, "cant reboot, check logs")
		return
	}
	irc.Reply(c, "brb")
	c.Respawn()
}

func commandMasterQuit(c *Connection, irc *IRC) {
	irc.Reply(c, "I am unstoppable. Did you mean... reboot ? upgrade ?")
}

func commandMasterPart(c *Connection, irc *IRC) {
	part := func(ch string) []byte {
		if strings.HasPrefix(ch, "#") {
			return []byte("PART :" + ch)
		}
		return nil
	}
	if strings.HasPrefix(irc.To, "#") {
		irc.Reply(c, ":(")
		c.Write(part(irc.To))
		c.SendMaster("Parted channel: %q", irc.To)
		return
	}
	if len(irc.Arguments) == 1 {
		c.Write(part(irc.Arguments[0]))
		c.SendMaster("Parted channel: %q", irc.Arguments[0])
	}

}

func commandMasterDebug(c *Connection, irc *IRC) {
	c.Log.Println(c, irc)
}
func commandMasterReload(c *Connection, irc *IRC) {
	err := LoadPlugin(c, "plugin.so")
	if err != nil {
		c.SendMaster("%v", err)
	}
	c.SendMaster("plugins reloaded")
}
func commandMasterSet(c *Connection, irc *IRC) {
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
func commandMasterUpgrade(c *Connection, irc *IRC) {
	checkout := exec.Command("git", "checkout", "master")
	out, err := checkout.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		c.SendMaster("Could not checkout 'master' branch: %v", err)
		c.SendMaster(string(out))
		return
	}
	update := exec.Command("git", "pull", "origin", "master")

	out, err = update.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		c.SendMaster("Could not pull 'master' branch: %v", err)
		c.SendMaster(string(out))
		return
	}
	upgrade := exec.Command("make", "all")

	out, err = upgrade.CombinedOutput()
	c.Log.Println(string(out))
	if err != nil {
		c.Log.Println(irc, err)
		c.SendMaster("Could not rebuild: %v", err)
		c.SendMaster(string(out))
		return
	}
	c.SendMaster("respawning now")
	c.Respawn()

}
func masterCommandLoadPlugin(c *Connection, irc *IRC) {
	if len(irc.Arguments) != 1 {
		irc.Reply(c, "need plugin name")
		return
	}
	err := LoadPlugin(c, irc.Arguments[0])
	if err != nil {
		c.SendMaster("error loading plugin: %v", err)
		return
	}
	irc.Reply(c, "plugin loaded: "+irc.Arguments[0])
}

func masterCommandFetchPlugin(c *Connection, irc *IRC) {
	if irc.Arguments == nil || len(irc.Arguments) != 1 {
		irc.Reply(c, red+"need plugin name")
		return
	}
	name := irc.Arguments[0]
	if strings.TrimSpace(name) == "" || strings.Contains(name, "..") {
		return
	}

	fetch := exec.Command("go", "build",
		"-o", name, "-v", "-buildmode=plugin",
		"github.com/aerth/ircb-plugins/"+name)

	out, err := fetch.CombinedOutput()
	if err != nil {
		c.Log.Println("error while fetching plugin %q: %v", name, err)
		c.SendMaster(red+"error: %s %v", string(out), err)
		return
	}
	err = LoadPlugin(c, name)
	if err != nil {
		c.Log.Println("error while loading plugin %q: %v", name, err)
		c.SendMaster(red+"error loading: %v", err)
		return
	}

	irc.Reply(c, fmt.Sprintf(green+"plugin loaded: %q", name))

}
