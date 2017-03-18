package ircb

import (
	"os"
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

// HandleMasterCommand handles master commands
func (c *Connection) HandleMasterCommand(irc IRC) (handled bool) {
	if irc.Command == "" {
		return false
	}

	if c.Config.MasterCommands == nil {
		c.Write(irc.From, "no")
		return false
	}

	c.Logf("[MasterCommand] %q %q\n", irc.Command, irc.Message)

	if c.Config.MasterCommands[irc.Command] != nil {
		c.Config.MasterCommands[irc.Command](c, irc)
		return true
	}

	c.WriteMaster(red.Sprintf("i dont know the command %q in %q", irc.Command, irc.Message))
	return false

}

// listMasterCommands for botmaster
func listMasterCommands(c *Connection, irc IRC) {
	c.Write(irc.Channel, c.Config.ListMasterCommands())
}

// list of built-in masterCommands
func registerMasterCommands() map[string]func(c *Connection, irc IRC) {
	var masterCommands = map[string]func(c *Connection, irc IRC){}

	masterCommands["part"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Write(irc.Channel, "PART "+strings.Join(irc.CommandArguments[1:], " "))
			c.Writer <- "PART " + strings.Join(irc.CommandArguments[1:], " ")
		}
	}
	masterCommands["join"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Write(irc.Channel, "JOIN "+strings.Join(irc.CommandArguments[1:], " "))
			c.Writer <- "JOIN " + strings.Join(irc.CommandArguments[1:], " ")
		}
	}

	masterCommands["tell"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Write(irc.CommandArguments[1], strings.Join(irc.CommandArguments[2:], " "))
		}
	}

	masterCommands["getenv"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) == 2 {
			c.Write(c.Config.Master, os.Getenv(irc.CommandArguments[1]))
		}
	}
	masterCommands["setenv"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 2 {
			os.Setenv(irc.CommandArguments[1], (irc.CommandArguments[2]))
			c.WriteMaster("set")
		}
	}


	masterCommands["list"] = func(c *Connection, irc IRC) {

	}



	masterCommands["q"] = func(c *Connection, irc IRC) {
		c.Stop()
	}
	masterCommands["autojoin"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
		} else {
			c.WriteMaster(red.Sprintf("Not enough args for %q", irc.Command))
		}
	}
	masterCommands["save-config"] = func(c *Connection, irc IRC) {
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
			return
		}
		c.Write(irc.Channel, "config saved")
		return
	}
	masterCommands["reload-config"] = func(c *Connection, irc IRC) {
		err := c.Config.Reload()
		if err != nil {
			c.WriteMaster(err.Error())
			return
		}
		c.Write(irc.Channel, "config reloaded. restart for changes to take effect.")
	}
	masterCommands["r"] = func(c *Connection, irc IRC) {
		for _, v := range c.Config.Channels {
			c.Write(v, "brb")
		}
		c.WriteMaster("brb")
		spawn.Spawn()
		c.Stop("brb")

	}
	masterCommands["u"] = func(c *Connection, irc IRC) {
		t1 := time.Now()
		c.Write(irc.Channel, "rebuilding @ "+t1.Format(time.Kitchen))
		out, ok := spawn.Rebuild("", "upgrade.sh")
		if !ok {
			out, ok = spawn.Rebuild("", "make")
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
			} else {
				c.WriteMaster(red.Sprint(lines[len(lines)-5:]))
			}
		}
		return
	}
	masterCommands["say"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Write(irc.Channel, strings.Join(irc.CommandArguments[1:], " "))
		} else {
			c.Write(irc.Channel, "say what?")
		}
		return
	}
	masterCommands["sayin"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 2 {
			c.Write(irc.CommandArguments[1], strings.Join(irc.CommandArguments[2:], " "))
		} else {
			c.Write(irc.Channel, "say what?")
		}
	}
	masterCommands["owner"] = func(c *Connection, irc IRC) {

		if len(irc.CommandArguments) > 1 {
			for i := 1; i < len(irc.CommandArguments)-1; i++ {
				c.Config.owners[irc.CommandArguments[i]] = i
				c.WriteMaster(orange.Sprintf("new owner: %q\n", irc.CommandArguments[i]))
			}
		} else {
			c.WriteMaster(green.Sprint(len(irc.CommandArguments)))
			c.WriteMaster(orange.Sprintf("all owners: %q\n", c.Config.Owners))

		}

		c.Config.Owners = ""
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
		}

		c.WriteMaster(orange.Sprintf("all owners: %q\n", c.Config.Owners))

	}
	masterCommands["ownerdel"] = func(c *Connection, irc IRC) {
		delete(c.Config.owners, getuser(irc.From))

		c.Config.Owners = ""
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
		}

		c.WriteMaster(orange.Sprintf("all owners: %q\n", c.Config.Owners))
	}
	masterCommands["do"] = func(c *Connection, irc IRC) {
		c.Writer <- strings.TrimPrefix(irc.Message, c.Config.CommandPrefix+"do ")
	}

	return masterCommands

}
