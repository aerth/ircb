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

	"github.com/boltdb/bolt"
)

var version = "ircb v0.0.8"
var ErrNoPluginSupport = fmt.Errorf("no plugin support")
var ErrNoPlugin = fmt.Errorf("no plugin found")
var ErrPluginInv = fmt.Errorf("invalid plugin")

// PluginInitFunc for plugins
type PluginInitFunc (func(c *Connection) error)

// LoadPlugin loads the named plugin file
var LoadPlugin = func(c *Connection, s string) error {
	return ErrNoPluginSupport
}

func (c *Connection) LoadPlugin(name string) error {
	return LoadPlugin(c, name)
}

type Connection struct {
	Log        *log.Logger
	HttpClient *http.Client
	config     *Config
	boltdb     *bolt.DB
	conn       io.ReadWriteCloser
	since      time.Time // since connected to server
	masterauth time.Time // auth and auth timeout
	reader     *bufio.Reader
	CommandMap map[string]Command
	MasterMap  map[string]Command
	maplock    sync.Mutex
	connected  bool
	joined     bool
	quiet      bool
}

func (config *Config) NewConnection() (*Connection, error) {
	var err error
	if config.Database == "" {
		config.Database = "bolt.db"
	}
	c := new(Connection)
	c.config = config
	c.Log = log.New(os.Stderr, "", log.Lshortfile)
	c.boltdb, err = loadDatabase(c.config.Database)
	if err != nil {
		return nil, err
	}

	c.since = time.Now()
	// for now, using default client.
	c.HttpClient = http.DefaultClient
	c.CommandMap = DefaultCommandMap()
	c.MasterMap = DefaultMasterMap()
	return c, nil
}
func (c *Connection) Connect() (err error) {
	if c.config.Diamond {
		err = initializeDiamond(c)
		if err != nil {
			return err
		}
	}
	if !c.connected {
		c.connected = true

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
	err1 := c.boltdb.Close()
	if err1 != nil {
		c.Log.Println(err1)
	}
	if c.config.Diamond {
		os.Remove("diamond.socket")
	}
	_, err := c.conn.Write([]byte(fmt.Sprintf("QUIT :%s\r\n", version)))
	if err != nil {
		c.Log.Println(err)
	}
	return c.conn.Close()
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
	e := irc.Encode()
	c.Log.Printf(">%q", string(e))
	if len(e) > 512 {
		c.Log.Println("length: %v", len(e))
	}
	_, err := c.Write(e)
	if err != nil {
		c.Log.Println(err)
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
