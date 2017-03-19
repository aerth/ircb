package ircb

import (
	"io/ioutil"
	"os/exec"
)

func dotool(c *Connection, irc IRC) {

	n := len(irc.CommandArguments)

	if n < 2 {
		c.Write(irc.Channel, "need arg")
		return
	}

	var args []string

	if c.Config.Tools[irc.CommandArguments[1]] == irc.CommandArguments[1] {
		args = []string{"./tools/" + c.Config.Tools[irc.CommandArguments[1]]}
	}

	if args == nil {
		c.WriteMaster(red.Sprintf("tool bad: from %s, %q", irc.From, irc.Message))
		return
	}
	if n > 2 {
		for _, v := range irc.CommandArguments[2:] {
			args = append(args, v)
		}
	}
	cmd := exec.Command("sh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.Log(red.Sprint(err))
		c.WriteMaster(red.Sprint(err.Error()))
	}
	c.Log(string(out))
	c.WriteMaster(green.Sprint(string(out)))
	if irc.From != c.Config.Master {
		c.Write(irc.From, green.Sprint(string(out)))
	}

}

func domtool(c *Connection, irc IRC) {

	n := len(irc.CommandArguments)
	if n < 2 {
		c.Write(irc.Channel, "need arg")
		return
	}

	var args []string

	if c.Config.MasterTools[irc.CommandArguments[1]] != "" {
		args = []string{"./mtools/" + c.Config.MasterTools[irc.CommandArguments[1]]}
	}

	if args == nil {
		c.WriteMaster("bad tool")
		return
	}
	if n > 2 {
		for _, v := range irc.CommandArguments[2:] {
			args = append(args, v)
		}
	}
	cmd := exec.Command("sh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.Log(red.Sprint(err))
		c.WriteMaster(red.Sprint(err.Error()))
	}
	c.Log(string(out))
	c.WriteMaster(green.Sprint(string(out)))
}

func registerMasterTools() (map[string]string, error) {
	var mastertools = map[string]string{}
	dir, err := ioutil.ReadDir("./mtools")
	if err != nil {
		return mastertools, err
	}
	for _, file := range dir {
		mastertools[file.Name()] = file.Name()
	}
	return mastertools, nil
}

func registerTools() (map[string]string, error) {
	var tools = map[string]string{}
	dir, err := ioutil.ReadDir("./tools")
	if err != nil {
		return tools, err
	}
	for _, file := range dir {
		tools[file.Name()] = file.Name()
	}
	return tools, nil
}
