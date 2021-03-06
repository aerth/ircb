package ircb

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
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

// RemoveMasterCommand adds a new public command, named 'name' to the CommandMap
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

// DefaultCommandMap returns default command map
func DefaultCommandMap() map[string]Command {
	m := make(map[string]Command)
	m["quiet"] = commandQuiet   // quiet
	m["up"] = commandUptime     // bot uptime
	m["help"] = commandHelp     // list commands
	m["about"] = commandAbout   // about link
	m["karma"] = commandKarma   // karma system
	m["define"] = commandDefine // define system
	return m
}

// DefaultMasterMap returns default master command map
func DefaultMasterMap() map[string]Command {
	m := make(map[string]Command)
	m["do"] = commandMasterDo             // do <raw irc>
	m["upgrade"] = commandMasterUpgrade   // upgrade and redeploy
	m["r"] = commandMasterReboot          // reboot
	m["part"] = commandMasterPart         // part <channel>
	m["echo"] = commandEcho               // echo back
	m["quit"] = commandMasterQuit         // cant quit
	m["q"] = commandMasterQuit            // cant quit
	m["help"] = commandMasterHelp         // list master commands
	m["set"] = commandMasterSet           // set (some) config options
	m["plugin"] = masterCommandLoadPlugin // load a local .so
	m["fetch"] = masterCommandFetchPlugin // fetch latest plugin from repo
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

	c.databaseDefine(action, definition)
	irc.Reply(c, fmt.Sprintf("defined: %q", action))

}
func commandMasterDo(c *Connection, irc *IRC) {
	c.Log.Println("GOT DO:", irc)
	c.Write([]byte(strings.Join(irc.Arguments, " ")))
}
func commandMasterReboot(c *Connection, irc *IRC) {
	b := c.MarshalConfig()
	err := ioutil.WriteFile("config.json", b, 0600)
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
func commandMasterSet(c *Connection, irc *IRC) {
	if len(irc.Arguments) != 2 {
		irc.Reply(c, `usage: set optionname on|off`)
		return
	}
	option := irc.Arguments[0]
	value := irc.Arguments[1]
	switch option {
	default:
		irc.Reply(c, `no option like that, 'links' 'define' or 'karma'`)
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

	case "karma":
		switch value {
		case "on":
			c.config.Karma = true
		case "off":
			c.config.Karma = false
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
	name := strings.TrimSpace(irc.Arguments[0])
	err := LoadPlugin(c, name)
	if err != nil {
		c.SendMaster("error loading plugin: %v", err)
		return
	}
	irc.Reply(c, "plugin loaded: "+irc.Arguments[0])
}

func masterCommandFetchPlugin(c *Connection, irc *IRC) {
	os.Setenv("CGO_ENABLED", "1")
	if irc.Arguments == nil || len(irc.Arguments) != 1 {
		irc.Reply(c, Red+"need plugin name")
		return
	}
	name := irc.Arguments[0]
	if strings.TrimSpace(name) == "" || strings.Contains(name, "..") {
		return
	}

	irc.Reply(c, "fetching plugin")
	fetch := exec.Command("go", "get", "-v", "-d", "github.com/aerth/ircb-plugins/...")
	out, err := fetch.CombinedOutput()
	if err != nil {
		c.Log.Printf("error while fetching plugin %q: %v", name, err)
		c.SendMaster(Red+"error: %s %v", string(out), err)
		return
	}
	irc.Reply(c, "compiling plugin")
	build := exec.Command("go", "build",
		"-o", name+".so", "-v", "-buildmode=plugin",
		"github.com/aerth/ircb-plugins/"+name)
	out, err = build.CombinedOutput()
	if err != nil {
		c.Log.Printf("error while fetching plugin %q: %v", name, err)
		c.SendMaster(Red+"error: %s %v", string(out), err)
		return
	}
	err = LoadPlugin(c, name)
	if err != nil {
		c.Log.Printf("error while loading plugin %q: %v", name, err)
		c.SendMaster(Red+"error loading: %v", err)
		return
	}

	irc.Reply(c, fmt.Sprintf(Green+"plugin loaded: %q", name))

}

func commandKarma(c *Connection, irc *IRC) {
	if !c.config.Karma {
		irc.ReplyUser(c, "karma is disabled")
		return
	}
	if len(irc.Arguments) != 1 {
		irc.Reply(c, c.karmaShow(irc.ReplyTo))
		return
	}

	irc.Reply(c, c.karmaShow(irc.Arguments[0]))
}
func (c *Connection) parseKarma(input string) bool {
	if !c.config.Karma {
		return nothandled
	}
	split := strings.Split(input, " ")
	if len(split) < 1 {
		return nothandled
	}

	// input is more than one word, try "thank" method
	if len(split) > 1 {
		if strings.Contains(input, "thank") {
			if i := strings.Index(input, ":"); i != -1 && i != 0 {
				c.Log.Println("Karma:", input[0:i])
				c.karmaUp(input[0:i])
				return handled
			}
			return nothandled
		}
		return nothandled
	}

	// user+
	if strings.HasSuffix(input, "+") {
		c.karmaUp(strings.Replace(input, "+", "", -1))
		return handled
	}

	// user-
	if strings.HasSuffix(input, "-") {
		c.karmaDown(strings.Replace(input, "-", "", -1))
		return handled
	}

	// no karma found
	return nothandled
}
