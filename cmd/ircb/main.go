package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aerth/ircb"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var (
	flaghost          = flag.String("h", "localhost:6667", "host (in the format 'host:port')")
	flagnick          = flag.String("n", "mustangsally", "nick")
	flagmaster        = flag.String("m", "root:@", "master:commandprefix")
	flagcommandprefix = flag.String("c", "!", "public command prefix")
	flagssl           = flag.Bool("ssl", false, "use ssl to connect")
	flaginvalidssl    = flag.Bool("x", false, "accept invalid tls certificates")
	flagdisablemacros = flag.Bool("nodefine", false, "dont use definition system")
	flagdisablekarma  = flag.Bool("nokarma", false, "dont use karma system")
	verbose           = flag.Bool("v", false, "lots of extra printing")
)

func main() {
	flag.Parse()

LoadConfig:
	config := buildconfig()
	b, err := ioutil.ReadFile("config.json")
	if err == nil {
		if len(b) != 0 {
			err = json.Unmarshal(b, &config)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else if strings.Contains(err.Error(), "no such") {
		_, err = os.Create("config.json")
		if err != nil {
			log.Fatalln("cant create config.json:", err)
		}
		goto LoadConfig

	}
	if *verbose {
		config.Verbose = *verbose
	}
	conn := config.NewConnection()
	err = ircb.LoadPlugin(conn, "plugin.so")
	if err != nil && err != ircb.ErrNoPluginSupport && err != ircb.ErrNoPlugin {
		log.Fatal(err)
	}

	go catchSignals(conn)

	err = conn.Connect()
	if err != nil {
		if strings.Contains(err.Error(), "delete if you want") {
			os.Remove("diamond.socket")
			err = conn.Connect()
		}
	}
	conn.Log.Println(err)
	os.Exit(111)
}
func buildconfig() *ircb.Config {
	config := ircb.NewDefaultConfig()
	config.Host = *flaghost
	config.Nick = *flagnick
	config.Master = *flagmaster
	config.UseSSL = *flagssl
	config.InvalidSSL = *flaginvalidssl
	config.CommandPrefix = "!"
	config.Karma = (*flagdisablekarma == false)
	config.Define = (*flagdisablemacros == false)

	if master := os.Getenv("MASTER"); master != "" {
		config.Master = master
	}
	if addr := os.Getenv("ADDR"); addr != "" {
		config.Host = addr
	}
	if nick := os.Getenv("NICK"); nick != "" {
		config.Nick = nick
	}

	return config
}

func catchSignals(c *ircb.Connection) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGQUIT, syscall.SIGHUP)
	// Block until a signal is received.
	s := <-ch
	c.Log.Println("Got signal:", s)
	if d := c.Diamond(); d != nil {
		d.Runlevel(0)
	}
	os.Exit(111)
}
