package ircb

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 * commands.go
 *
 * respond to commands
 * new command functions look like this:
 *
 * func (c *Connection, irc IRC){}
 */

func dofortune(c *Connection, irc IRC) {
	cmd := exec.Command("fortune", "-s")
	b, err := cmd.CombinedOutput()
	if err != nil {
		c.WriteMaster(string(err.Error()))
	}
	c.SlowSend(irc, string(b))
}

// SlowSend sends every 800 millisecond
func (c *Connection) SlowSend(irc IRC, message string) {
	if message == "" {
		return
	}
	split := strings.Split(message, "\n")
	for i, line := range split {
		line = strings.TrimSpace(line)
		if line == "" && i == len(split)-1 {
			return
		}
		if clean(line) == "" {
			continue
		}
		if !strings.HasPrefix(irc.From, "#") {
			c.Write(irc, randomcolor().Sprint(line))
		} else {
			c.Write(irc, line)
		}
		<-time.After(500 * time.Millisecond)
	}
}

// CommandSay returns command function that says s
func CommandSay(str string) func(c *Connection, irc IRC) {
	return func(c *Connection, irc IRC) {
		c.Write(irc, str)
	}
}

// CommandSayf returns command function that says s
func CommandSayf(s string, si ...string) func(c *Connection, irc IRC) {
	return func(c *Connection, irc IRC) {
		var i []interface{}
		for _, v := range si {
			switch v {
			case "from":
				i = append(i, irc.From)
			case "channel":
				i = append(i, irc.Channel)
			case "command":
				i = append(i, irc.Command)
			case "args":
				i = append(i, irc.CommandArguments)
			default:
				i = append(i, v)
			}
		}
		c.Write(irc, fmt.Sprintf(s, i...))
	}
}

// CommandDo returns command function that sends raw s
func CommandDo(s string) func(c *Connection, irc IRC) {
	return func(c *Connection, irc IRC) {
		c.Writer <- s
	}
}

// HandleVerbINT handles only INT verbs suchs as 443 (nick in use)
func (c *Connection) HandleVerbINT(verb int, irc IRC) (handled bool) {
	// if c.Config.Verbose {
	// 	t1 := time.Now()
	// 	defer func() { c.Logf("int handle took %s", time.Now().Sub(t1)) }()
	// }

	switch verb {
	default:
		if c.Config.Verbose {
			c.Log("** Got new INT verb:", verb)
		}
		return false
	case 1, 2, 3, 4, 5, 6, 7, 8, 9,
		250, 251, 252, 253, 254, 255, 256, 257, 258, 259, 260,
		261, 262, 263, 264, 265, 266, 267, 268, 269:
	case 372: //   MOTD
	case 375: // START MOTD
	case 376: // END MOTD
	case 332: // channel topic
	case 333: // channel topic info
	case 353: // chanel user list
	case 366: // end of NAMES list
	case 903: // SASL success
	case 904: // SASL fail
		c.Stop()
	case 900: // now logged in
		if c.Config.Password != "" && !c.Config.UseServices {
			c.Writer <- "CAP END"
		}
		return
	case 401: // no such nick/channel
	case 474: // banned
		c.WriteMaster(irc.Message)
	case 433: // nick in use
		go func() {
			c.connected = false
			c.Config.Password = "" // chances are, new user name is not registered
			alert("Nick in use. Adding int and removing password.")
			c.Config.Name = c.Config.Name + randstr()
			<-time.After(1 * time.Second)
			c.Reconnect()
		}()
	}
	handled = true
	return handled
}

func (c *Connection) HandlePRIVMSG(irc IRC) bool {
	var handled = false
	var t1 = time.Now()
	defer fmt.Printf("PRIVMSG handle took %q\n", time.Now().Sub(t1))

	if irc.Channel == c.Config.Name {
		irc.Channel = irc.From
	}

	// cmd := strings.Split(irc.Message, " ")[0]
	// cmd = strings.TrimSpace(cmd)

	if irc.Command == "" {
		c.Log(red.Sprint("blank command\n"))
		return handled
	}
	if c.Config.Commands[irc.Command] != nil {
		c.Log(orange.Sprint("Command map command"))
		c.Config.Commands[irc.Command](c, irc)
		return true

	}
	if irc.From == c.Config.Master && c.Config.MasterCommands[irc.Command] != nil {
		c.Config.MasterCommands[irc.Command](c, irc)
		return true
	}
	c.Log(blue.Sprintf("switch irc.Command: %q ", irc.Command))
	switch irc.Command { // first word
	case "h", "help", "commands", "list":
		if c.Config.Commands["help"] != nil {
			c.Config.Commands["help"](c, irc)
		}
		return handled
	case "about":
		c.Write(irc, fmt.Sprintf("%s: ircb v%s source code at https://github.com/aerth/ircb", irc.From, c.Config.Version))
		return handled
	default:
		// started with command prefix, but not found in Command map or above cases.
		if irc.From == c.Config.Master {
			c.HandleMasterCommand(irc)
			return handled
		}

		c.Write(irc, red.Sprintf("i dont know the command %q in %q", irc.Command, irc.Message))

	}
	return false
}

// return username extracted from between : and !
func getuser(s string) string {
	return strings.TrimPrefix(strings.Split(s, "!")[0], ":")
}

func registerCommands() map[string]func(c *Connection, irc IRC) {
	var commands = map[string]func(c *Connection, irc IRC){}

	// -=about
	commands["about"] = func(c *Connection, irc IRC) {
		c.Write(irc, "ircb v0.0.3 (https://github.com/aerth/ircb)")
	}
	// -=hello
	commands["hello"] = CommandSayf("Hello, %s", "channel")

	// -=up
	commands["up"] = douptime

	// -=ping
	commands["ping"] = CommandSay("pong")
	commands["v"] = CommandSay(version)

	// -=help
	commands["help"] = ListCommands

	// ircb logo
	commands["ircb"] = CommandSay(logo + "\n" + version)

	// beer me
	commands["beer"] = CommandSayf("Have a beer, %s", "from")

	// -=fortune
	commands["fortune"] = dofortune
	commands["tools"] = listTools
	commands["tool"] = dotool
	return commands
}

// ListCommands for public
func ListCommands(c *Connection, irc IRC) {
	c.Write(irc, c.Config.ListCommands())
}

func douptime(c *Connection, irc IRC) {
	c.Write(irc, "Uptime: "+time.Now().Sub(boottime).String())
}
