package ircb

import (
	"encoding/base64"
	"fmt"
)

func init() {

}

func (c *Connection) AuthSASL1() {
	if c.Config.Verbose {
		c.Log(green.Sprint("[sending sasl requirement"))
	}
	if c.Config.Password != "" {
		c.Writer <- "CAP LS"
		c.Writer <- "CAP REQ :multi-prefix sasl"
	}
}

func (c *Connection) Mode(flags string) {
	if flags == "" || c.Config.Name == "" {
		c.Log("bad MODE flags")
		return
	}

	c.Writer <- fmt.Sprintf("MODE %s %s", c.Config.Name, flags)
}

// AuthSASL2 Stage 2: send credentials
func (c *Connection) AuthSASL2() {
	if c.Config.Verbose {
		c.Log("[sending sasl credentials]")
	}
	c.Writer <- "AUTHENTICATE PLAIN"

	if c.Config.Password != "" && c.Config.SASL == "" {
		c.Config.SASL = c.Base64()
		err := c.Config.Save()
		if err != nil {
			errorf("error saving base64 SASL credentials: %s", err)
		}
	}
	c.Writer <- "AUTHENTICATE " + c.Config.SASL
	c.authsent = true
}

// AuthServices sends credentials via PRIVMSG to NickServ
func (c *Connection) AuthServices() {
	if c.Config.Verbose {
		c.Log(green.Sprint("[identifying with services]"))
	}
	c.Writer <- "PRIVMSG NickServ :identify " + c.Config.Password
	c.authsent = true
}

// AuthRegister as a user/nick on a server
func (c *Connection) AuthRegister() {
	if c.Config.Verbose {
		c.Log(green.Sprintf("[registering user %s", c.Config.Name))
	}
	c.Writer <- fmt.Sprintf("NICK %s", c.Config.Name)
	if c.Config.Account == "" {
		c.Config.Account = c.Config.Name
	}
	c.Writer <- fmt.Sprintf("USER %s 0.0.0.0 0.0.0.0 :%s", c.Config.Account, c.Config.Account)
}

func (c Connection) Base64() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s", c.Config.Name, c.Config.Name, c.Config.Password)))
}
