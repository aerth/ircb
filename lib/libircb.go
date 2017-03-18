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

			// Write to connection
			_, err := c.conn.Write([]byte(fmt.Sprintf("%s\r\n", out)))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}
	}()

}

// ircb only responds to pings, INT verbs, and PRIVMSG verbs for now
func (c *Connection) ircb() {
	c.Config.Commands = registerCommands()
	c.Config.MasterCommands = registerMasterCommands()
	if c.Config.Verbose {
		c.Log(green.Sprint("[ircb] on"))
		defer c.Log(green.Sprint("[ircb] off"))
	}
	for read := range c.Reader {
		c.Netlog = append(c.Netlog, read)

		c.Log(fmt.Sprintf("< %v %s", len(c.Netlog), read))

		if strings.HasPrefix(clean(read), "PING") {
			c.Writer <- strings.Replace(read, "PING", "PONG", -1)
			continue
		}

		if read == STOP {
			return
		}

		irc := ParseIRC(read, c.Config.CommandPrefix)
		c.HandleIRC(irc)
	}
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
			c.Reconnect()
			return
		}

		go func() { c.Reader <- msg }()
	}
}

// Start loops
func (c *Connection) initialConnect() {
	go c.netWriter()
	go c.netReader()
	go c.initializeConnection() // registers, waits for MODE set and launches c.ircb()
}

// joinChannels joins config channels and sends bot master the command prefix
func (c *Connection) joinChannels() {
	// msg master
	c.WriteMaster("Prefix commands with \"" + green.Sprint(c.Config.CommandPrefix) + "\"")
	c.WriteMaster(orange.Sprintf("Joining channels: %q", c.Config.Channels))

	// join channels
	for _, v := range c.Config.Channels {
		c.Writer <- fmt.Sprintf("JOIN %s", v)
		<-time.After(500 * time.Millisecond)
	}

}

func (c *Connection) Loop() {
	for c != nil && c.conn != nil {
		<-time.After(5 * time.Second)
	}

	fmt.Println("*** WOOT ***")
}
func quit(code ...int) {
	if code == nil {
		code = []int{0}
	}
	os.Exit(code[0])
}
