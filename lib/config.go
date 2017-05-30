package ircb

import (
	"encoding/json"
)

// Config holds configurable variables for ircb client
type Config struct {
	Host          string // in the form 'host:port'
	Nick          string
	Master        string
	CommandPrefix string
	UseSSL        bool
	InvalidSSL    bool
	EnableTools   bool
	EnableKarma   bool
	EnableHistory bool
	EnableMacros  bool
	HistoryFile   string
	KarmaFile     string
}

// NewDefaultConfig returns the default config, minimal changes would be Host,Nick,Master for typical usage.
func NewDefaultConfig() *Config {
	config := new(Config)
	config.Host = "localhost:6667"
	config.Nick = "mustangsally"
	config.Master = "aerth"
	config.CommandPrefix = "!"
	config.UseSSL = false
	config.InvalidSSL = false
	config.EnableTools = true
	config.EnableKarma = true
	config.EnableHistory = true
	config.EnableMacros = true
	return config
}

// Marshal into json encoded bytes from config values
func (c Config) Marshal() ([]byte, error) {
	return json.Marshal(c)
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
