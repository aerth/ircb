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
	d.SetRunlevel(1, func() error { return nil })
	d.Runlevel(1)
	return nil
}
