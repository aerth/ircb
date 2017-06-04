package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aerth/ircb"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var (
	flaghost           = flag.String("h", "localhost:6667", "host (in the format 'host:port')")
	flagnick           = flag.String("n", "mustangsally", "nick")
	flagmaster         = flag.String("m", "root:@", "master:commandprefix")
	flagcommandprefix  = flag.String("c", "!", "public command prefix")
	flagssl            = flag.Bool("ssl", false, "use ssl to connect")
	flaginvalidssl     = flag.Bool("x", false, "accept invalid tls certificates")
	flagdisabletools   = flag.Bool("notools", false, "dont use toolkit system")
	flagdisablemacros  = flag.Bool("nomacros", false, "dont use macros system")
	flagdisablehistory = flag.Bool("nohistory", false, "dont use history system")
	flagdisablekarma   = flag.Bool("nokarma", false, "dont use karma system")
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
	conn, err := config.NewConnection()
	if err != nil {
		log.Fatal(err)
	}
	ircb.DefaultCommandMaps()
	publicmap, mastermap, err := ircb.LoadPlugin("plugin.so")

	if err != nil && err != ircb.ErrNoPluginSupport {
		log.Fatal(err)
	}

	if err != nil {
		ircb.CommandMap, ircb.MasterMap = publicmap, mastermap
	}
	err = conn.Connect()

	if err != nil {
		log.Fatal(err)
	}

	if b, err := conn.MarshalConfig(); err == nil {
		err := ioutil.WriteFile("config.json", b, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func buildconfig() *ircb.Config {
	config := ircb.NewDefaultConfig()
	config.Host = *flaghost
	config.Nick = *flagnick
	config.Master = *flagmaster
	config.UseSSL = *flagssl
	config.InvalidSSL = *flaginvalidssl
	config.CommandPrefix = "!"
	config.EnableTools = (*flagdisabletools == false)
	config.EnableKarma = (*flagdisablekarma == false)
	config.EnableHistory = (*flagdisablehistory == false)
	config.EnableMacros = (*flagdisablemacros == false)

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