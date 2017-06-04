package ircb

import (
	"crypto/tls"
)

func (cfg *Config) dialtls() (*tls.Conn, error) {

	return tls.Dial("tcp", cfg.Host, &tls.Config{
		InsecureSkipVerify: cfg.InvalidSSL,
	})
}
