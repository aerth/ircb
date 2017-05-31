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
)

var version = "ircb v0.0.7"

type Connection struct {
	Log         *log.Logger
	HttpClient  *http.Client
	conn        io.ReadWriteCloser
	config      *Config
	since       time.Time
	lines       int
	historyfile *os.File
	karmafile   *os.File
	reader      *bufio.Reader
	writer      *bufio.Writer
	done        chan int
	definitions map[string]string
	commandmap  map[string]Command
	mastermap   map[string]Command
	karma       map[string]int // map[nick]level
	karmalock   sync.Mutex
	joined      bool
	quiet       bool
}

func (config *Config) NewConnection() (*Connection, error) {
	var err error
	if config.HistoryFile == "" {
		config.HistoryFile = "history.db"
	}
	if config.KarmaFile == "" {
		config.KarmaFile = "karma.db"
	}
	if config.DictionaryFile == "" {
		config.DictionaryFile = "dictionary.db"
	}
	c := new(Connection)
	c.Log = log.New(os.Stderr, "", log.Lshortfile)
	c.config = config
	c.since = time.Now()
	c.lines = 0
	c.done = make(chan int)
	c.historyfile, err = os.OpenFile(config.HistoryFile, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return c, err
	}
	c.karmafile, err = os.OpenFile(config.KarmaFile, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return c, err
	}
	c.definitions, err = LoadDefinitions(config.DictionaryFile)
	if err != nil {
		return c, err
	}
	c.karma, err = LoadKarmaMap(c.karmafile)
	if err != nil {
		return c, err
	}
	c.commandmap = DefaultCommandMap()
	c.mastermap = DefaultMasterMap()
	c.HttpClient = http.DefaultClient
	// dial direct
	c.conn, err = net.Dial("tcp", c.config.Host)
	if err != nil {
		return c, err
	}
	err = c.initialconnect()
	if err != nil {
		return c, err
	}
	c.Log.Println("connected.")
	go c.readerwriter()
	return c, err
}

func (c *Connection) Close() error {
	defer c.historyfile.Close()
	defer c.karmafile.Close()
	_, err := c.conn.Write([]byte(fmt.Sprintf("QUIT :%s\r\n", version)))
	if err != nil {
		return err
	}
	c.done <- 0
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
	c.Log.Println("SEND", string(b))
	return c.conn.Write(b)
}

// Send IRC
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
func (c *Connection) Wait() {
	<-c.done
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

func (c *Connection) readerwriter() {
	c.Log.Println("reading from net")
	defer c.Log.Println("reader stopping")
	c.reader = bufio.NewReaderSize(c.conn, 512)
	for {
		msg, err := c.reader.ReadString('\n')
		if err != nil {
			c.Log.Println("read error:", err)
			if err == io.EOF || strings.Contains(err.Error(), "use of closed") {
				return
			}
		}
		c.Log.Printf("read: %q", msg)
		if strings.HasPrefix(msg, "PING") {
			pong := []byte(strings.Replace(msg, "PING", "PONG", -1))
			_, err = c.Write(pong)
			if err != nil {
				c.Log.Println(err)
			}
			continue
		}
		msg = strings.TrimPrefix(msg, ":")
		irc := c.Parse(msg)
		if _, err := strconv.Atoi(irc.Verb); err == nil {
			HandleVerbINT(c, irc)
			continue
		}

		if irc.ReplyTo == strings.Split(c.config.Master, ":")[0] {
			if HandleMasterVerb(c, irc) {
				continue
			}
		}

		HandleVerb(c, irc)
	}
}
