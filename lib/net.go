package ircb

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 * net.go
 *
 * connects to irc Server
 *
 *
 */

// Connection type
type Connection struct {
	Host                string
	Config              *Config
	netlog              []string
	Reader, Writer      chan string
	connected, authsent bool
	logfile             *os.File
	conn                net.Conn
}

// Connect returns a connection
func (config *Config) Connect() *Connection {

	// some required fields
	for key, v := range map[string]string{
		"Master":        config.Master,
		"Hostname":      config.Hostname,
		"Name":          config.Name,
		"CommandPrefix": config.CommandPrefix,
	} {

		if v == "" {

			alertf("Need config field: %q\n", key)
			os.Exit(1)
		} else {
			infof("Config: %s is %s\n", key, v)
		}

	}

	// use tls by default
	if config.Port == 0 {
		config.Port = 6697
		if config.NoTLS {
			config.Port = 6667
		}
	}

	config.owners = map[string]int{}
	config.owners[config.Master] = 1
	for _, v := range strings.Split(config.Owners, ",") {
		config.owners[v] = 2
	}
	c := new(Connection)
	c.Writer = make(chan string)
	c.Reader = make(chan string)
	c.Config = config
	c.openlogfile()
	config.NewDialer()

	// dial
	var err error
	c.conn, err = config.Dialer.Dial("tcp", fmt.Sprintf("%s:%d", config.Hostname, config.Port))

	// connection error
	if err != nil {
		alert("ircb cant connect -", err)
		<-time.After(3 * time.Second)
		return config.Connect()
	}

	// tls handshake
	if !config.NoTLS {
		c.tlsHandshake()
	}

	// do once
	go c.initialConnect()
	<-time.After(time.Millisecond * 1000) // need 1 netlog to gather real hostname

	// find hostname
	c.Host = realHostname(c)
	return c
}

// Create a TLS Config for HandshakeTLS
func (config *Config) configTLS() *tls.Config {
	var tconf = new(tls.Config)
	if config.Hostname == "" {
		fmt.Println("no config hostname. cant do tls")
		os.Exit(1)
	}
	tconf.InsecureSkipVerify = config.InvalidTLS // FALSE by default
	tconf.ServerName = config.Hostname
	tconf.RootCAs = nil // use host RootCAs
	return tconf
}

// Start TLS handshake, swap c.conn for TLS connection
func (c *Connection) tlsHandshake() {
	if c.Config.Verbose {
		c.Log("TLS Handshake")
	}
	c.conn = tls.Client(c.conn, c.Config.configTLS())
}

// NewDialer adds a dialer to config. Either SOCKS or Direct.
func (config *Config) NewDialer() {
	// direct, no proxy
	if config.Socks == "" || config.Socks == "none" || config.Socks == "no" || config.Socks == "n" || config.Socks == "N" || config.Socks == "0" {
		config.Dialer = proxy.Direct
		fmt.Fprintln(os.Stderr, "Direct Connection")
		return
	}

	// proxy keywords
	if config.Socks == "1080" || config.Socks == "true" {
		config.Socks = "socks5://127.0.0.1:1080"
	} else if config.Socks == "tor" {
		config.Socks = "socks5://127.0.0.1:9050"
	}

	// add prefix if necessary
	if !strings.HasPrefix(config.Socks, "socks5://") && !strings.HasPrefix(config.Socks, "socks4://") {
		config.Socks = "socks5://" + config.Socks
	}

	// parse address
	proxyurl, err := url.Parse(config.Socks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't parse SOCKS address: %v\n", err)
		os.Exit(1)
	}

	// create dialer
	proxydialer, err := proxy.FromURL(proxyurl, proxy.Direct)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create SOCKS dialer: %v\n", err)
		os.Exit(1)
	}

	// use SOCKS dialer for future connections
	config.Dialer = proxydialer
}

func realHostname(c *Connection) string {

	for i := 0; len(c.netlog) < 1; i++ {
		if i >= 5 {
			fmt.Printf("Timeout.")
			c.Stop()
			os.Exit(1)
		}
		alertf("No netlog, waiting. Try %v\n", i)
		<-time.After(time.Millisecond * 1000) // need 1 netlog to gather real hostname
	}

	return strings.TrimPrefix(strings.Split(c.netlog[0], " ")[0], ":")
}

// Reconnect after disconnecting
func (c *Connection) Reconnect() {
	c.Stop("reconnect")
	c = c.Config.Connect()
}

// Stop will quit clean, Stop("reconnect") will attempt to reconnect
func (c *Connection) Stop(args ...interface{}) {
	quitmsg := ""
	reconnecting := "reconnecting"
	if c.Config.Verbose {
		fmt.Println("Attempting [STOP]")
	}

	for _, arg := range args {
		switch arg.(type) {
		case error:
			fmt.Println(red.Sprint(arg))
			quitmsg = "got error"
		case string:
			fmt.Println(green.Sprint(arg))
			if arg == "reconnect" {
				quitmsg = reconnecting
			} else {
				quitmsg = arg.(string)
			}
		default:
			quitmsg = "ircb client: https://github.com/aerth/ircb"
		}
	}

	if c != nil && quitmsg != reconnecting {
		doreport(c.netlog)
		if quitmsg != "" {
			go func() { c.Writer <- "QUIT :" + quitmsg }()
		}
		<-time.After(500 * time.Millisecond)
		go func() { c.Writer <- STOP }()
		<-time.After(500 * time.Millisecond)
		go func() { c.Reader <- STOP }()
		<-time.After(500 * time.Millisecond)
	}

	if c != nil && c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, red.Sprint(err))
		}
	}

	fmt.Fprintln(os.Stderr, "ircb: gone", time.Now().String())
	c = nil
	if quitmsg != reconnecting {
		go func() {
			<-time.After(3 * time.Second)
			os.Exit(0)
		}()
	}
}

// startup
func (c *Connection) startup() {
	green.Println("[connecting]")
	defer func() {
		green.Println("[connected]")
	}()

	for read := range c.Reader {
		c.netlog = append(c.netlog, read)

		// ircb STOP
		if read == STOP {
			return
		}

		// "Closing Link"
		if strings.HasPrefix(read, "ERROR") {
			c.Stop()
			os.Exit(1)
			return
		}

		// Parse IRC
		irc := ParseIRC(read, c.Config.CommandPrefix)

		verbint, err := strconv.Atoi(irc.Verb)
		if err == nil { // is number verb
			if c.HandleVerbINT(verbint, irc) {
				continue // need MODE to exit startup to exit startup
			}
		}

		if irc.Verb != "" {
			c.Log(fmt.Sprintf("< %v %s", len(c.netlog), read))
		}

		// Three non-int verbs matter during inital connection
		switch irc.Verb {
		case "PING", ":" + c.Host:
			fmt.Println("PONGMF")
			c.Writer <- strings.Replace(read, "PING", "PONG", -1)
		case "NOTICE":
			if len(c.netlog) == 1 {
				// SASL (no SERVICES)
				if c.Config.Password != "" && !c.Config.UseServices {
					c.AuthSASL1()    // require SASL before registering
					c.AuthRegister() // NICK/USER
					continue
				}

				// Register NICK/USER
				c.AuthRegister() // NICK/USER

				// USE SERVICES
				if c.Config.UseServices {
					c.AuthServices()
				}

				continue // need MODE to exit startup
			}
		case "NICK":
			if strings.Contains(irc.Message, c.Config.Master) {
				c.WriteMaster(c.Config.CommandPrefix)
			}
		case "MODE": // got first MODE change. join channels and start ircb
			go c.ircb()
			c.joinChannels()
			return
		case "CAP":
			if !c.Config.UseServices && irc.Message == "ACK :multi-prefix sasl" {
				fmt.Println("AuthSasl2")
				c.AuthSASL2()
				continue // need MODE to exit startup
			}
		}
	}
}

// WaitFor a string or return most recent occurance
func (c *Connection) WaitFor(grep []string, timelimit time.Duration) int {
	t1 := time.Now()
	defer green.Printf("WaitFor %q took %s\n", grep, time.Now().Sub(t1))
	filter := make(chan int)

	go func() {
		var all []string
		copy(all, c.netlog)
		for i := len(all) - 1; i >= 0; i-- {
			// Got new message
			if len(c.netlog) != len(all) {
				filter <- (-1)
			}
			for _, grepfor := range grep {
				// Visit each backwards
				if strings.Contains(all[i], grepfor) {
					filter <- i
				}
			}
		}

	}()

	// time limit
	select {
	case <-time.After(timelimit):
		return -1
	case line := <-filter:
		if line == -1 {
			return c.WaitFor(grep, timelimit)
		}
		return line
	}

}

// Write a PRIVMSG to user or channel
func (c *Connection) Write(channel, message string) {
	if !strings.Contains(message, "\n") {
		c.Writer <- fmt.Sprintf(`PRIVMSG %s :%s`, channel, message)
	} else {
		c.SlowSend(channel, message)
	}
}

// WriteMaster a PRIVMSG to c.Config.Master
func (c *Connection) WriteMaster(message string) {
	go func() { c.Write(c.Config.Master, message) }()
}
