package main

import (
	"flag"
	"log"
	"os"

	ircb "github.com/aerth/ircb/lib"
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
	config := buildconfig()
	conn, err := config.NewConnection()
	if err != nil {
		log.Fatal(err)
	}
	conn.Wait()
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
