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
 *
 */

// Commands map
var Commands = map[string]func(c *Connection, irc IRC){}

// Verbs can be map
var Verbs = map[string]func(c *Connection, read string){}

var boottime = time.Now()

func init() {

	// -=about
	Commands["about"] = func(c *Connection, irc IRC) {
		c.Write(irc.Channel, "ircb v2 (https://github.com/aerth/ircb)")
	}

	// -=hello
	Commands["hello"] = func(c *Connection, irc IRC) {
		c.Write(irc.Channel, "hello "+irc.Channel)
	}
	// -=up
	Commands["up"] = func(c *Connection, irc IRC) {
		c.Write(irc.Channel, "uptime: "+time.Since(boottime).String())
	}

	// -=ping
	Commands["ping"] = CommandSay("pong")

	// -=fortune
	Commands["fortune"] = dofortune

}

func dofortune(c *Connection, irc IRC) {
	cmd := exec.Command("fortune", "-s")
	b, _ := cmd.Output()
	split := strings.Split(string(b), "\n")
	for i, line := range split {
		line = strings.TrimSpace(line)
		if line == "" && i == len(split)-1 {
			return
		}
		if line == "" {
			line = " "
		}
		c.Write(irc.Channel, randomcolor().Sprint(line))
		<-time.After(1000 * time.Millisecond)
	}
}

// CommandSay returns command function that says s
func CommandSay(s string) func(c *Connection, irc IRC) {
	return func(c *Connection, irc IRC) {
		c.Write(irc.Channel, s)
	}
}

// CommandDoreturns command function that sends raw s
func CommandDo(s string) func(c *Connection, irc IRC) {
	return func(c *Connection, irc IRC) {
		c.Writer <- s
	}
}

// HandleVerbINT handles only INT verbs suchs as 443 (nick in use)
func (c *Connection) HandleVerbINT(verb int, irc IRC) {
	// if c.Config.Verbose {
	// 	t1 := time.Now()
	// 	defer func() { c.Logf("int handle took %s", time.Now().Sub(t1)) }()
	// }
	switch verb {
	default:
		if c.Config.Verbose {
			c.Log("** Got new INT verb:", verb)
		}
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
		c.Write(c.Config.Master, irc.Message)
	case 433: // nick in use
		go func() {
			c.connected = false
			c.Config.Password = "" // chances are...
			alert("Nick in use. Adding _ and removing password.")
			c.Config.Name = c.Config.Name + "_"
			<-time.After(1 * time.Second)
			c.Reconnect()
		}()
	}
}


func (c *Connection) HandlePRIVMSG(irc IRC) {
	t1 := time.Now()
	defer fmt.Printf("PRIVMSG handle took %q\n", time.Now().Sub(t1))

	if irc.Channel == c.Config.Name {
		irc.Channel = irc.From
	}
	if irc.Command == "" {
		fmt.Println("Not a command:", irc.Message)
		if strings.Contains(irc.Message, c.Config.Name){
			c.Write(c.Config.Master, irc.Message)
			return
		}
		fmt.Println("not good")
		return
	}
	// cmd := strings.Split(irc.Message, " ")[0]
	// cmd = strings.TrimSpace(cmd)
	cmd := irc.Command
	if Commands[cmd] != nil {
		c.Log(orange.Sprint("Command map command"))
		Commands[cmd](c, irc)
		return
	}

	c.Log(blue.Sprintf("switch cmd: %q ", cmd))
	switch cmd { // first word

	// case "v":
	// 	c.Write(irc.Channel, "ircb "+c.Config.Version)
	case "h":
		c.Write(irc.From, "q r u v h beer about")
	case "beer":
		c.Write(irc.Channel, fmt.Sprintf("Have a beer, %s!", irc.From))
		//	case "about":
		//		c.Write(irc.Channel, fmt.Sprintf("%s: ircb v%s source code at https://github.com/aerth/ircb", irc.From, c.Config.Version))

	default:
		// started with command prefix, but not found in Command map or above cases.
		if irc.From == c.Config.Master {
			c.HandleMasterCommand(irc)
		}
	}
}
