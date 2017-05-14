// 'ircb' (irc bot)
package main

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 */

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	ircb "github.com/aerth/ircb/lib"
	"github.com/fatih/color"
)

func init() {
	rand.Seed(time.Now().Add(time.Hour).UnixNano())
}

var (

	/*
	 * flags override config, overwriting .config
	 * example:
	 * 			ircb -host localhost -port 6667 -nick mustangsally_ -notls
	 * 			# open .config and edit "password" field
	 * 			# run 'ircb' with no flags
	 *
	 *
	 */
	version     string
	unsafe      = flag.Bool("unsafe", false, "trust invalid TLS certificates")
	verbose     = flag.Bool("v", false, "verbose output")
	notls       = flag.Bool("notls", false, "no SSL/TLS")
	useservices = flag.Bool("nicksrv", false, "use Services instead of SASL")
	socks       = flag.String("socks", "true", "SOCKS proxy to use")
	configloc   = flag.String("conf", ".config", "config file location")
	botname     = flag.String("nick", "mustangsally", "Bot Nickname")
	botacct     = flag.String("acct", "bot", "nick acct@hostname")
	botmaster   = flag.String("master", "aerth", "nickname allowed to run master commands")
	nosec       = flag.Bool("nosec", false, "set this flag to allow master commands from master who hasn't identified with services.")
	bothost     = flag.String("host", "chat.freenode.net", "IRC Server (hostname)")
	botport     = flag.Int("port", 6697, "IRC Server (port)")
	channels    = flag.String("channels", "##ircb", "comma separated channels to autojoin. eg: -channels='#test,#test3'")
	cmdprefix   = flag.String("prefix", "-=", "respond to messages with this prefix")
)

var built = "go get -v github.com/aerth/ircb/cmd/ircb"

// colors are only for logs
var green = color.New(color.FgGreen)
var red = color.New(color.FgRed)
var orange = color.New(color.FgRed)
var clr0 = color.New(color.Reset)
var good = clr0.Sprint("[") + green.Sprint("*") + "]"
var bad = clr0.Sprint("[") + red.Sprint("*") + "]"
var alert = clr0.Sprint("[") + orange.Sprint("*") + "]"

func randomcolor() *color.Color {
	colorlist := []color.Attribute{color.FgBlue, color.FgCyan, color.FgGreen,
		color.FgMagenta, color.FgRed, color.FgHiBlue, color.FgHiCyan,
		color.FgHiGreen, color.FgHiMagenta, color.FgHiRed}
	clr := color.New(colorlist[rand.Intn(len(colorlist)-1)])
	return clr
}

func main() {
	config := DoConfig()
	config.Display()
	green.Fprintf(os.Stderr, "connecting to %s in 3", config.Hostname)
	<-time.After(1 * time.Second)
	green.Fprintf(os.Stderr, "..2")
	<-time.After(1 * time.Second)
	green.Fprintf(os.Stderr, "..1\n")
	<-time.After(1 * time.Second)
	DoConnect(config)
}

// DoConnect really connects
func DoConnect(config *ircb.Config) {

	c := config.Connect()

	// catch signals
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch) // all signals

	go func() {
		for sign := range sigch {
			fmt.Fprintln(os.Stderr, "ircb got", sign.String())
			// quit clean
			c.Stop()
			os.Exit(0)
		}
	}()

	c.Loop()
}

// DoConfig with flag overrider
func DoConfig() *ircb.Config {
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "too many arguments")
		os.Exit(1)
	}

	config := configFromFlags()

	// load json from ConfigLocation
	err := config.Reload()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[config] read error:", err)
	}
	visitFlags(config)
	// create dialer
	config.NewDialer()

	// save (updated) config to file
	err = config.Save()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[config] write error:", err)
		os.Exit(1)
	}

	return config
}

func configFromFlags() *ircb.Config {

	// load defaults from flagset
	config := new(ircb.Config)
	config.Boottime = time.Now()
	config.Channels = strings.Split(*channels, ",")
	config.CommandPrefix = *cmdprefix
	config.ConfigLocation = *configloc
	config.Hostname = *bothost
	config.Master = *botmaster
	config.Name = *botname
	config.Account = *botacct
	config.Port = *botport
	config.Socks = *socks
	config.InvalidTLS = *unsafe
	config.Version = green.Sprint(version, built)
	config.Version = version
	config.NoTLS = *notls
	config.Verbose = *verbose
	config.UseServices = *useservices
	config.ConfigLocation = *configloc
	config.Version = built
	return config

}

func visitFlags(config *ircb.Config) {
	// visit user flags, over-riding config file
	flag.Visit(func(a *flag.Flag) {
		switch a.Name {
		case "host":
			config.Hostname = a.Value.String()
		case "port":
			port, err := strconv.Atoi(a.Value.String())
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			config.Port = port
		case "master":
			config.Master = a.Value.String()
		case "nick":
			config.Name = a.Value.String()
		case "acct":
			config.Account = a.Value.String()
		case "channels":
			config.Channels = strings.Split(a.Value.String(), ",")
		case "conf":
			config.ConfigLocation = a.Value.String()
		case "prefix":
			config.CommandPrefix = a.Value.String()
		case "socks":
			config.Socks = a.Value.String()
		case "notls":
			config.NoTLS, _ = strconv.ParseBool(a.Value.String())
		case "unsafe":
			config.InvalidTLS, _ = strconv.ParseBool(a.Value.String())
		case "v":
			config.Verbose, _ = strconv.ParseBool(a.Value.String())
			//		case "nosec":
			//			config.NoSecurity, _ = strconv.ParseBool(a.Value.String())
		}
		fmt.Fprintln(os.Stderr, alert, "config override: ", a.Name, "=", a.Value)

	}) // flag.Visit

}
