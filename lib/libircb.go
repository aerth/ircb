package ircb

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 */

const (
	// PRIVMSG const
	PRIVMSG = "PRIVMSG"
	// STOP is used to stop a channel
	STOP = "STOP"
)

var boottime = time.Now()
var commit string
var version = "ircb v0.0.4 https://github.com/aerth/ircb "
var versionstring = version + "-" + commit

func (c *Connection) registercommands() {
	c.Config.Commands = registerCommands()
	c.Config.MasterCommands = registerMasterCommands()
	var err error
	c.Config.MasterTools, err = registerMasterTools()
	if err != nil {
		c.Log("Can't register master tools:", err)
	}
	c.Config.Tools, err = registerTools()
	if err != nil {
		c.Log("Can't register public tools:", err)
	}

}

func (c *Connection) Wait(dur time.Duration) bool {
	select {
	case <-c.wait:
		return true
	case <-time.After(dur):
		return false
	}
}

// ircb logs net receives. responds to PING or handle parsed irc message
func (c *Connection) ircb() {

	if c.Config.Verbose {
		c.Log(green.Sprint("[ircb] on"))
		defer c.Log(green.Sprint("[ircb] off"))
	}
	go func() {
		c.wait <- 1
	}()
	for {
		select {
		case <-time.After(30 * time.Second):
			c.Log(green.Sprint("ircb") + ": no net reads (30s)")
		case read := <-c.Reader:
			//c.Log(green.Sprint("ircb")+":", read)
			if strings.HasPrefix(clean(read), "PING") {
				fmt.Println(red.Sprint("WRONG PONG"))
				c.Writer <- strings.Replace(read, "PING", "PONG", -1)
				continue
			}

			// stop
			if read == STOP {
				return
			}

			// parse IRC message and handle in a goroutine
			go c.HandleIRC(ParseIRC(read, c.Config.CommandPrefix))
		}
	}
}

// Writes when needed
func (c *Connection) netWriter() {
	if c.Config.Verbose {
		c.Log(green.Sprint("[net writer] on"))
	}
	go func() {
		defer c.Log(red.Sprint("[net writer] off"))
		for out := range c.Writer {
			// log output
			c.Log("> " + out)

			// stop writer loop
			if out == STOP {
				return
			}
			if len([]byte(out)) > 512 {
				c.Log(red.Sprint("Write will be truncated"))
			}
			// Write to connection
			_, err := c.conn.Write([]byte(fmt.Sprintf("%s\r\n", out)))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			<-time.After(500 * time.Millisecond)
		}
	}()

}

// netInput stays on. Reads from irc connection, sends to c.Reader channel.
func (c *Connection) netReader() {
	if c.Config.Verbose {
		fmt.Println("[Reader] on")
	}
	buf := bufio.NewReaderSize(c.conn, 512)
	for {
		msg, err := buf.ReadString('\n')
		if msg == STOP {
			alert("[Reader] off")
			return
		}
		if err != nil {

			// fatal error
			if err.Error() == "EOF" || strings.Contains(err.Error(), "use of closed") {
				return
			}

			// tls error
			if strings.HasPrefix(err.Error(), "tls: ") {
				errorf("%s\n", err)
				c.Stop()
				return
			}

			// non-fatal error, reconnect
			fmt.Fprintf(os.Stderr, "Error [%s] while reading message, reconnecting...\n", err)
			go c.Reconnect()
			return
		}

		go func() { c.Reader <- msg }()
	}
}

// Start loops
func (c *Connection) initialConnect() {
	go c.netWriter()
	go c.netReader()
	go c.startup() // registers, waits for MODE set and launches c.ircb()
}

// joinChannels joins config channels and sends bot master the command prefix
func (c *Connection) joinChannels() {
	// msg master
	c.WriteMaster(logo)
	c.WriteMaster(rnbo(version))
	c.WriteMaster("Prefix commands with \"" + green.Sprint(c.Config.CommandPrefix) + "\"")
	c.WriteMaster(orange.Sprintf("Joining channels: %q", c.Config.Channels))

	// join channels
	for _, v := range c.Config.Channels {
		c.Writer <- fmt.Sprintf("JOIN %s", v)
		<-time.After(500 * time.Millisecond)
	}

}

// Loop will continue until connection is nil
func (c *Connection) Loop() {
	for c != nil {
		<-time.After(5 * time.Second)
	}
}

func quit(code ...int) {
	if code == nil {
		code = []int{0}
	}
	os.Exit(code[0])
}
