package ircb

import (
	"encoding/json"
)

// Config holds configurable variables for ircb client
type Config struct {
	Host          string // in the form 'host:port'
	Nick          string
	Master        string // in the form 'master:prefix'
	CommandPrefix string
	Channels      string // comma separated channels to autojoin
	UseSSL        bool
	InvalidSSL    bool
	ParseLinks    bool
	Define        bool
	Verbose       bool
	Karma         bool
	History       bool
	Diamond       bool   // socket for console control (experimental)
	Database      string // path to boltdb (can be empty to use bolt.db)
	AuthMode      int    // 0 ACC (freenode), 1 STATUS, -1 none
}

// NewDefaultConfig returns the default config, minimal changes would be Host,Nick,Master for typical usage.
func NewDefaultConfig() *Config {
	config := new(Config)
	config.Host = "chat.freenode.net:6697"
	config.Nick = "mustangsally"
	config.Master = "aerth:$"
	config.CommandPrefix = "!"
	config.Channels = "##ircb,##ircb"
	config.UseSSL = true
	config.InvalidSSL = false
	config.Karma = true
	config.History = true
	config.ParseLinks = false
	config.Define = true
	return config
}

// MarshalConfig encodes the connection's config as JSON
func (c *Connection) MarshalConfig() []byte {
	b, _ := c.config.Marshal()
	return b
}

// Marshal into json encoded bytes from config values
func (c Config) Marshal() ([]byte, error) {
	return json.MarshalIndent(c, " ", " ")
}

// ConfigFromJSON loads a new config from json encoded bytes.
// It starts with a NewDefaultConfig, so not all fields must be present in json code.
func ConfigFromJSON(b []byte) (*Config, error) {
	config := NewDefaultConfig()
	err := json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
