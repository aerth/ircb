package ircb

import (
	"strings"

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
func (c *Connection) HandleMasterCommand(irc IRC) {
	if irc.Command == "" {
		return
	}
	c.Log("[MasterCommand] ", irc)

	// Master commands

	/*
	 * must be identified by services if possible
	 * disabled for now
	 */

	// if !c.Config.NoSecurity {
	// 	if !c.isIdentified(c.Config.Master) {
	// 		c.Write(c.Config.Master, "You must identify with services to use this command.")
	// 		return
	// 	}
	// }

	switch irc.Command {
	default:
	case "leave", "part", "bye":
		c.Writer <- "PART " + irc.Channel
	case "q", "quit": // quit
		c.Stop()

	case "save-config":
		err := c.Config.Save()
		if err != nil {
			c.Write(c.Config.Master, err.Error())
			return
		}
		c.Write(irc.Channel, "config saved")
	case "reload-config":
		err := c.Config.Reload()
		if err != nil {
			c.Write(c.Config.Master, err.Error())
			return
		}
		c.Write(irc.Channel, "config reloaded. restart for changes to take effect.")
		return
	case "r", "reboot", "restart": // reboot
		for _, v := range c.Config.Channels {
			c.Write(v, "brb")
		}
		c.Write(c.Config.Master, "brb")
		spawn.Spawn()
		c.Stop()
	case "u", "upgrade", "rebuild": // upgrade
		out, ok := spawn.Rebuild(".", "upgrade.sh")
		if !ok {
			out, ok = spawn.Rebuild(".", "make")
		}
		if ok {
			for _, v := range c.Config.Channels {
				c.Write(v, "brb")
			}
			c.Write(c.Config.Master, "brb")
			c.Log(out)
			spawn.Spawn()
			c.Stop()

		} else {
			c.Log(out)
			c.Write(irc.Channel, "no")
			c.Write(c.Config.Master, strings.TrimSpace(out))
		}
	case "say":
		if len(irc.CommandArguments) > 1 {
			c.Write(irc.Channel, strings.Join(irc.CommandArguments, " "))
		} else {
			c.Write(irc.Channel, "say what?")
		}
	case "owner":
		c.Log("new master:", c.Config.Master)
		if len(irc.CommandArguments) > 1 {
			for i := 1; i < len(irc.CommandArguments); i ++ {
				c.Config.Owners = append(c.Config.Owners, irc.CommandArguments[i])
			}
		}
	case "do":
				c.Writer <- strings.TrimPrefix(irc.Message, "do ")


	}

	switch {
	default:
		c.Write(c.Config.Master, red.Sprintf("i dont know the command %q in %q", irc.Command, irc.Message))
		//

	case strings.HasPrefix(irc.Command, c.Config.CommandPrefix): // change prefix
		c.Config.CommandPrefix = strings.TrimPrefix(irc.Command, c.Config.CommandPrefix)
		c.Write(c.Config.Master, c.Config.CommandPrefix)
	case strings.HasPrefix(irc.Command, "sayin "):
		cmd := strings.TrimPrefix(irc.Command, "sayin ")
		irc.Channel = strings.Split(cmd, " ")[0]
		c.Write(irc.Channel, strings.TrimPrefix(cmd, irc.Channel))
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
