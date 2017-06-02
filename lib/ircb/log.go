package ircb

import "os"

func openlogfile() (f *os.File, err error) {
	return os.OpenFile(".log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
}
