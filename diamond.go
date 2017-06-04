package ircb

import (
	"io"

	diamond "github.com/aerth/diamond/lib"
)

func initializeDiamond(closer io.Closer) error {
	d, err := diamond.New("diamond.socket")
	if err != nil {
		return err
	}
	d.SetRunlevel(0, closer.Close)
	d.Runlevel(1)
	return nil
}
