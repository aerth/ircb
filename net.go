package ircb

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	diamond "github.com/aerth/diamond/lib"
	"github.com/boltdb/bolt"
)

var version = "ircb v0.0.9"

type Connection struct {
	Log        *log.Logger
	HttpClient *http.Client       // customize user agent, proxy, tls, redirects, etc
	CommandMap map[string]Command // map of command names to Command functions
	MasterMap  map[string]Command // map of master command names to Command functions
	Diamond    *diamond.System    // can be nil
	config     *Config            // current config
	boltdb     *bolt.DB           // opened database
	conn       io.ReadWriteCloser
	since      time.Time // since connected to server
	masterauth time.Time // auth and auth timeout
	reader     *bufio.Reader
	maplock    sync.Mutex // guards (both) command map writes
	connected  bool
	joined     bool
	quiet      bool
}

func (config *Config) NewConnection() *Connection {
	if config.Database == "" {
		config.Database = "bolt.db"
	}
	c := new(Connection)
	c.config = config
	c.since = time.Now()

	// TODO: custom client for user agent, proxy support
	c.HttpClient = http.DefaultClient
	c.CommandMap = DefaultCommandMap()
	c.MasterMap = DefaultMasterMap()
	if config.Verbose {
		c.Log = log.New(os.Stderr, "", log.Lshortfile)
	} else {
		c.Log = log.New(os.Stderr, "", log.Ltime)
	}
	return c
}
func (c *Connection) Connect() (err error) {
	if !c.connected {
		c.connected = true
		defer func(c *Connection) {
			c.connected = false
			c.Close()
		}(c)
		if c.config.Diamond {
			d, err := diamond.New("diamond.socket")
			if err != nil {
				return err
			}
			d.Config.Kickable = true
			d.SetRunlevel(0, func() error {
				return c.Close()
			})
			d.SetRunlevel(1, func() error { return nil })
			d.Runlevel(1)
			c.Diamond = d
		}
		c.boltdb, err = loadDatabase(c.config.Database)
		if err != nil {
			return err
		}

		// dial direct
		c.Log.Println("connecting...")

		if c.config.UseSSL {
			c.conn, err = c.config.dialtls()
		} else {
			c.conn, err = net.Dial("tcp", c.config.Host)
		}
		if err != nil {
			return err
		}
		err = c.initialconnect()
		if err != nil {
			return err
		}

		c.Log.Println("connected.")
		return c.readerwriter()
	}
	return fmt.Errorf("already connected")
}
func (c *Connection) Close() error {
	if c == nil {

		return nil
	}
	if c.boltdb != nil {
		err1 := c.boltdb.Close()
		if err1 != nil {
			c.Log.Println(err1)
		}
	}
	os.Remove("diamond.socket")
	if c.conn != nil {
		_, err := c.conn.Write([]byte(fmt.Sprintf("QUIT :%s\r\n", version)))
		if err != nil {
			c.Log.Println(err)
		}
		if c != nil && c.conn != nil {
			return c.conn.Close()
		}
	}
	return nil
}

// Write to irc connection, adding '\r\n'
func (c *Connection) Write(b []byte) (n int, err error) {
	if strings.TrimSpace(string(b)) == "" || len(b) < 4 {
		return 0, fmt.Errorf("write too small")
	}
	if string(b[len(b)-2:]) != "\r\n" {
		b = append(b, "\r\n"...)
	}
	str := string(b)
	if c.quiet && strings.Contains(str, "PRIVMSG") {
		if c.config.Verbose {
			c.Log.Println("MUTED:", str)
		}
		return
	}

	if c.config.Verbose {
		c.Log.Println("SEND", str)
	}
	return c.conn.Write(b)
}

// MasterCheck sends a private message to NickServ to authenticate master user
//
// 	-1 no auth mode
//	0 default, freenode and oragono ACC style
//	1 STATUS style
//
func (c *Connection) MasterCheck() {
	switch c.config.AuthMode {
	case -1:
		// no auth mode
		c.Log.Println("WARNING:", "Authentication Disabled.")
		c.masterauth = time.Now()
	default:
		// freenode and oragono style
		_, err := c.conn.Write([]byte("" +
			"PRIVMSG NickServ :ACC " + strings.Split(c.config.Master, ":")[0] + "\r\n"))
		if err != nil {
			c.Log.Printf("auth error: %b", err)
		}

	case 1:
		// STATUS style
		_, err := c.conn.Write([]byte("" +
			"PRIVMSG NickServ :STATUS " + strings.Split(c.config.Master, ":")[0] + "\r\n"))
		if err != nil {
			c.Log.Printf("auth error: %b", err)
		}

	}

}

// SendMaster sends formatted text to master user
func (c *Connection) SendMaster(format string, i ...interface{}) {
	if strings.TrimSpace(format) == "" {
		return
	}
	reply := IRC{
		To:      strings.Split(c.config.Master, ":")[0],
		Message: fmt.Sprintf(format, i...),
	}
	c.Send(reply)
}

// Send IRC message (uses To and Message fields)
func (c *Connection) Send(irc IRC) {
	irc.Message = strings.TrimSuffix(irc.Message, "\n")
	if strings.Contains(irc.Message, "\n") {
		messages := strings.Split(irc.Message, "\n")
		for _, v := range messages {
			if strings.TrimSpace(v) == "" {
				continue
			}

			line := IRC{
				To:      irc.To,
				Message: v,
			}
			c.Send(line)
			<-time.After(time.Second)

		}

		return
	}
	e := irc.Encode()
	c.Log.Printf(">%q", string(e))
	if len(e) < 512 {
		_, err := c.Write(e)
		if err != nil {
			c.Log.Println(err)
		}
		return
	}
	var line string

	for i, r := range []rune(irc.Message) {
		line = line + string(r)
		if i > 0 && (i+1)%500 == 0 {
			msg := IRC{
				To:      irc.To,
				Message: line,
			}
			c.Write(msg.Encode())
			line = ""
		}
	}
}

func (c *Connection) initialconnect() error {
	b := make([]byte, 512)
	_, err := c.conn.Read(b)
	if err != nil {
		return err
	}
	c.Log.Println(string(b))
	_, err = c.conn.Write([]byte(fmt.Sprintf("NICK %s\r\n", c.config.Nick)))
	if err != nil {
		return err
	}
	_, err = c.conn.Write([]byte(fmt.Sprintf("USER %s 0.0.0.0 0.0.0.0 :%s\r\n", c.config.Nick, c.config.Nick)))
	if err != nil {
		return err
	}

	_, err = c.conn.Write([]byte(fmt.Sprintf("MODE %s :%s", c.config.Nick, "+i\r\n")))
	if err != nil {
		return err
	}

	return nil
}

// read until read error
func (c *Connection) readerwriter() error {
	logfile, err := openlogfile()
	if err != nil {

		return err
	}
	defer logfile.Close()
	defer c.Log.SetOutput(os.Stderr)
	mw := io.MultiWriter(os.Stderr, logfile)
	c.Log.SetOutput(mw)
	logfile.Write([]byte(fmt.Sprintf("log started: %s\n", time.Now().String())))
	logfile.Sync()
	c.Log.Println("reading from net")
	defer c.Log.Println("reader stopping")
	c.reader = bufio.NewReaderSize(c.conn, 512)
	for {
		msg, err := c.reader.ReadString('\n')
		if err != nil {
			return err
		}
		if c.config.Verbose {
			c.Log.Printf("read: %q", msg)
			logfile.Sync()
		}

		// handle PING
		if strings.HasPrefix(msg, "PING") {
			pong := []byte(strings.Replace(msg, "PING", "PONG", -1))
			_, err = c.Write(pong)
			if err != nil {
				c.Log.Println(err)
			}
			continue
		}
		msg = strings.TrimPrefix(msg, ":")

		// parse
		cfg := *c.config
		irc := cfg.Parse(msg)
		// numeric 'verb'
		if _, err := strconv.Atoi(irc.Verb); err == nil {
			if verbIntHandler(c, irc) {
				continue
			}
		}

		switch irc.Verb {
		default:
			c.Log.Println("new verb", irc.Verb, irc.Message)
			if c.config.Verbose {
				c.Log.Println(irc)
			}
			continue
		case "QUIT", "PART", "NICK", "JOIN":
			continue
		case "NOTICE":
			// :NickServ!NickServ@services. NOTICE mastername :mustangsally ACC 3
			switch irc.ReplyTo {
			case "NickServ":
				switch c.config.AuthMode {
				default:
					if irc.Raw == fmt.Sprintf(formatauth, c.config.Nick, strings.Split(c.config.Master, ":")[0]) {
						c.masterauth = time.Now()
					}
				case -1:
					c.masterauth = time.Now()
				case 1:
					if irc.Message == fmt.Sprintf(formatauth2, strings.Split(c.config.Master, ":")[0]) {
						c.masterauth = time.Now()
					}

				}

			default:
				c.Log.Println("NOTICE", irc.ReplyTo, irc.Message)
			}

		case "MODE":
			c.Log.Printf("NEW MODE: %q", irc.Message)
			if !c.joined {
				for _, ch := range strings.Split(c.config.Channels, ",") {
					if ch != "" {
						c.Log.Println("Joining channel:", ch)
						c.Write([]byte(fmt.Sprintf("JOIN %s", ch)))
					}
				}
				c.joined = true

			}

		case "PRIVMSG":

			// maybe master command
			if irc.ReplyTo == strings.Split(c.config.Master, ":")[0] {
				if privmsgMasterHandler(c, irc) {
					continue
				}
			}

			privmsgHandler(c, irc)
		}
	}
}
