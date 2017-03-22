package ircb

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aerth/spawn"
)

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 * master-cmd.go
 *
 * respond to privileged user commands
 *
 * new command functions look like this:
 *
 * func (c *Connection, irc IRC){}
 *
 */

// HandleMasterCommand handles master commands
func (c *Connection) HandleMasterCommand(irc IRC) (handled bool) {
	if irc.Command == "" {
		return false
	}

	if c.Config.MasterCommands == nil {
		c.WriteMaster("no")
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
	c.WriteMaster(c.Config.ListMasterCommands())
}

func listMasterTools(c *Connection, irc IRC) {
	c.WriteMaster(c.Config.ListMasterTools())
}

func listTools(c *Connection, irc IRC) {
	c.Write(irc, c.Config.ListTools())
}

// list of built-in masterCommands
func registerMasterCommands() map[string]func(c *Connection, irc IRC) {
	var masterCommands = map[string]func(c *Connection, irc IRC){}

	masterCommands["part"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Writer <- "PART " + strings.Join(irc.CommandArguments[1:], " ")
		}
	}
	masterCommands["join"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Writer <- "JOIN " + strings.Join(irc.CommandArguments[1:], " ")
		}
	}

	masterCommands["tell"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Write(IRC{From: irc.CommandArguments[1]}, strings.Join(irc.CommandArguments[2:], " "))
		}
	}

	masterCommands["getenv"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) == 2 {
			c.WriteMaster(os.Getenv(irc.CommandArguments[1]))
		}
	}
	masterCommands["env"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) == 1 {
			c.WriteMaster(fmt.Sprint(os.Environ()))
		}
	}

	masterCommands["logcat"] = func(c *Connection, irc IRC) {
		lines := strings.Split(c.logcat(), "\n")
		c.WriteMaster(green.Sprintf("Lines: %v", len(lines)))
		if len(lines) < 10 {
			c.WriteMaster(strings.Join(lines, "\n"))
			return
		}

		c.WriteMaster(strings.Join(lines[len(lines)-9:], "\n"))
	}
	masterCommands["logname"] = func(c *Connection, irc IRC) {
		c.WriteMaster(c.logfile.Name())
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
			return
		}
		c.Config.Channels = append(c.Config.Channels, irc.CommandArguments[1])
		c.WriteMaster(green.Sprintf("Autojoin channel: %q", irc.CommandArguments[1]))
	}
	masterCommands["save-config"] = func(c *Connection, irc IRC) {
		err := c.Config.Save()
		if err != nil {
			c.WriteMaster(err.Error())
			return
		}
		c.WriteMaster("config saved")
		return
	}
	masterCommands["reload-config"] = func(c *Connection, irc IRC) {
		err := c.Config.Reload()
		if err != nil {
			c.WriteMaster(err.Error())
			return
		}
		c.WriteMaster("config reloaded. restart for changes to take effect.")
	}
	masterCommands["r"] = func(c *Connection, irc IRC) {
		c.WriteMaster("brb")
		spawn.Spawn()
		c.Stop("brb")

	}
	masterCommands["u"] = func(c *Connection, irc IRC) {
		t1 := time.Now()
		c.WriteMaster("rebuilding @ " + t1.Format(time.Kitchen))

		cmd := exec.Command("sh", "./upgrade.sh")
		out, err := cmd.CombinedOutput()
		if err == nil {
			c.WriteMaster(green.Sprintf("brb (build took %s)", time.Now().Sub(t1).String()))
			c.Log(out)
			spawn.Spawn()
			c.Stop("brb")

		} else {
			c.Log(string(out))
			c.WriteMaster("no")
			c.WriteMaster(string(out))
			}
		return
	}
	masterCommands["say"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 1 {
			c.Write(irc, strings.Join(irc.CommandArguments[1:], " "))
		} else {
			c.Write(irc, "say what?")
		}
		return
	}
	masterCommands["sayin"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) > 2 {
			c.Write(IRC{Channel: irc.CommandArguments[1]}, strings.Join(irc.CommandArguments[2:], " "))
		} else {
			c.WriteMaster("say what?")
		}
	}
	masterCommands["owner"] = func(c *Connection, irc IRC) {
		if len(irc.CommandArguments) < 2 {
			c.WriteMaster(orange.Sprintf("all owners: %q\n",
				c.Config.Owners))
			return
		}

		for i := 1; i < len(irc.CommandArguments)-1; i++ {
			c.Config.owners[irc.CommandArguments[i]] = i
			c.WriteMaster(orange.Sprintf("new owner: %q\n", irc.CommandArguments[i]))
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

	masterCommands["list*"] = listMasterCommands
	masterCommands["help*"] = listMasterCommands
	masterCommands["mtools"] = listMasterTools
	masterCommands["mtool"] = domtool
	masterCommands["reload-tools"] = func(c *Connection, irc IRC) {
		var err error
		c.Config.Tools, err = registerTools()
		if err != nil {
			c.WriteMaster("reload failed: " + err.Error())
			return
		}
		c.WriteMaster("Tools reloaded.")
	}

	return masterCommands

}
