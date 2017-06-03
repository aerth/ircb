package ircb

import (
	"crypto/tls"
)

func (cfg *Config) dialtls() (*tls.Conn, error) {

	return tls.Dial("tls", "mail.google.com:443", &tls.Config{
		InsecureSkipVerify: cfg.InvalidSSL,
	})
}
