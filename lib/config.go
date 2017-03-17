package ircb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

/*
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 * config.go
 */

// Config uration
type Config struct {
	Version        string       `json:"-"`
	ConfigLocation string       `json:"-"`
	Boottime       time.Time    `json:"-"`
	Dialer         proxy.Dialer `json:"-"`

	Master                            string // irc user
	Owners                            string // comma separated
	owners														map[string]int
	Hostname                          string
	Port                              int
	Name, AuthName, Account, Password string
	SASL                              string // base64 encoded sasl auth string
	Socks                             string // socks5:// address: "" for no proxy

	Channels []string
	//ChannelPassword                   map[string]string

	UseServices, NoTLS, InvalidTLS, Verbose, NoSecurity bool

	CommandPrefix string

	Commands map[string]func(c *Connection, irc IRC) `json:"-"`

}

// Save config to ('.config') by default
func (c *Config) Save() error {

	for key := range c.owners {
		c.Owners += key+","
	}
	c.Owners = strings.TrimSuffix(c.Owners, ",")


	// Create new config, writing over existing.
	configFile, err := os.OpenFile(c.ConfigLocation, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0640)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	var b []byte
	b, err = json.MarshalIndent(&c, "", " ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Config Write Error:", err)
		return err
	}
	configFile.Truncate(0)
	n, _ := configFile.Write(b)

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "Config file saved: %q (%v bytes)\n", c.ConfigLocation, n)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Can't write to config file:", err)
		return err

	}
	return nil
}

// Load config file
func InitConfig(configLocation string, c *Config) (*Config, error) {
	if c == nil {
		c = new(Config)
		print("[config] init\n")
	} else {
		print("[config] load\n")
	}

	defer print("\n")
	c.ConfigLocation = configLocation
	b, err := ioutil.ReadFile(c.ConfigLocation)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return c, nil
		}
		return nil, err
	}

	// no contents
	print(".")
	if len(b) == 0 {
		return nil, fmt.Errorf("[config] load error: config has 0 bytes")
	}

	// decode json
	print(".")
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, fmt.Errorf("[config] json error: %s", err)
	}

	print(".")
	if c.AuthName == "" && c.Name != "" {
		c.AuthName = c.Name
	}
	return c, nil
}

// Reload ('.config') in case it changed.
func (c *Config) Reload() error {
	dialer := c.Dialer // keep dialer

	b, err := ioutil.ReadFile(c.ConfigLocation)
	if err != nil {
		return err
	}

	if len(b) == 0 {
		return fmt.Errorf("config has 0 bytes")
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		return err
	}
	if c.AuthName == "" && c.Name != "" {
		c.AuthName = c.Name
	}
	c.Dialer = dialer
	return nil
}

var logo = `██╗██████╗  ██████╗██████╗
██║██╔══██╗██╔════╝██╔══██╗
██║██████╔╝██║     ██████╔╝
██║██╔══██╗██║     ██╔══██╗
██║██║  ██║╚██████╗██████╔╝
╚═╝╚═╝  ╚═╝ ╚═════╝╚═════╝`

// Display HUD
func (config *Config) Display() {

	fmt.Fprintln(os.Stderr, rnbo(logo))
	<-time.After(time.Second)
	// print if Verbose
	if config.Verbose {
		green.Fprintln(os.Stderr, config.String())

		<-time.After(time.Second)
		orange.Fprintln(os.Stderr, config.ListCommands())
	}

	if !config.NoTLS {
		fmt.Fprintln(os.Stderr, clrgood, "Using TLS")
	}
	if config.Password != "" {
		if config.UseServices {
			green.Fprintln(os.Stderr, clralert, "Using NickServ (no SASL)")
		} else {
			green.Fprintln(os.Stderr, clrgood, "Using SASL")
		}
	}

	if config.Socks != "" {
		fmt.Fprintln(os.Stderr, clrgood, "Using SOCKS")
	}

}
