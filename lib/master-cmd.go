package ircb

import (
	"strings"
	"time"

	"github.com/aerth/spawn"
)

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 * commands.go
 *
 * respond to privileged user commands
 *
 */

/*


 *
 *
 *
 *
 *
 *
 *
 */

// HandleMasterCommand handles master commands
func (c *Connection) HandleMasterCommand(irc IRC) (handled bool) {
	if irc.Command == "" {
		return false
	}
	c.Logf("[MasterCommand] %q %q\n", irc.Command, irc.Message)

	// Master commands

	/*
	 * must be identified by services if possible
	 * disabled for now
	 */

	// if !c.Config.NoSecurity {
	// 	if !c.isIdentified(c.Config.Master) {
	// 		c.WriteMaster("You must identify with services to use this command.")
	// 		return
	// 	}
	// }

	switch irc.Command {
	default:
	case "leave", "part", "bye":
		c.Writer <- "PART " + irc.Channel
		return true
	case "q", "quit": // quit
		c.Stop()
		return true
	case "autojoin":
		if len(irc.CommandArguments) > 1 {
		} else {
			c.WriteMaster(red.Sprintf("Not enough args for %q", irc.Command))
		}
	case "save-config":
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
			return true
		}
		c.Write(irc.Channel, "config saved")
		return true
	case "reload-config":
		err := c.Config.Reload()
		if err != nil {
			c.WriteMaster(err.Error())
			return true
		}
		c.Write(irc.Channel, "config reloaded. restart for changes to take effect.")
		return true
	case "r", "reboot", "restart": // reboot
		for _, v := range c.Config.Channels {
			c.Write(v, "brb")
		}
		c.WriteMaster("brb")
		spawn.Spawn()
		c.Stop("brb")
		return true
	case "u", "upgrade", "rebuild": // upgrade
		t1 := time.Now()
		c.Write(irc.Channel, "rebuilding @ " + t1.Format(time.Kitchen))
		out, ok := spawn.Rebuild(".", "upgrade.sh")
		if !ok {
			out, ok = spawn.Rebuild(".", "make")
		}
		if ok {
			c.WriteMaster(green.Sprintf("brb (build took %s)", time.Now().Sub(t1).String()))
			c.Log(out)
			spawn.Spawn()
			c.Stop("brb")

		} else {
			c.Log(out)
			c.Write(irc.Channel, "no")
			// short output | tail
			lines := strings.Split(out, "\n")
			if len(lines) < 5 {
			c.WriteMaster(red.Sprint(out))
		}	 else {
			c.WriteMaster(red.Sprint(lines[len(lines)-5:]))
		}
		}
		return true
	case "say":
		if len(irc.CommandArguments) > 1 {
			c.Write(irc.Channel, strings.Join(irc.CommandArguments[1:], " "))
		} else {
			c.Write(irc.Channel, "say what?")
		}
		return true
	case "sayin":
		if len(irc.CommandArguments) > 2 {
			c.Write(irc.CommandArguments[1], strings.Join(irc.CommandArguments[2:], " "))
		} else {
			c.Write(irc.Channel, "say what?")
		}
		return true
	case "owner":

		if len(irc.CommandArguments) > 1 {
			for i := 1; i < len(irc.CommandArguments)-1; i++ {
				c.Config.owners[irc.CommandArguments[i]] = i
				c.WriteMaster(orange.Sprintf("new owner: %q\n", irc.CommandArguments[i]))
			}
		} else {
			c.WriteMaster(green.Sprint(len(irc.CommandArguments)))
			c.WriteMaster(orange.Sprintf("all owners: %q\n", c.Config.Owners))
			return true
		}

		c.Config.Owners = ""
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
		}

		c.WriteMaster(orange.Sprintf("all owners: %q\n", c.Config.Owners))
		return true

	case "ownerdel":
		delete(c.Config.owners,getuser(irc.From))

		c.Config.Owners = ""
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
		}

		c.WriteMaster(orange.Sprintf("all owners: %q\n", c.Config.Owners))
		return true
	case "do":
		c.Writer <- strings.TrimPrefix(irc.Message, c.Config.CommandPrefix+"do ")
		return true
	}

	switch {
	default:
		c.WriteMaster(red.Sprintf("i dont know the command %q in %q", irc.Command, irc.Message))
		return false

	case strings.HasPrefix(irc.Command, c.Config.CommandPrefix): // change prefix
		c.Config.CommandPrefix = strings.TrimPrefix(irc.Command, c.Config.CommandPrefix)
		c.WriteMaster(c.Config.CommandPrefix)
		return true

	}
}

// func (c *Connection) isIdentified(name string) bool {
// 	c.Writer <- "WHOIS " + c.Config.Master
// 	line := c.WaitFor([]string{"is logged in as", "account"}, time.Second)
// 	if line == -1 {
// 		return false
// 	}
// 	irc := ParseIRC(c.Netlog[line], c.Config.CommandPrefix)
// 	return strings.Contains(irc.Message, c.Config.Master)
// }
