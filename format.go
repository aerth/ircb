package ircb

import "github.com/kr/pretty"

const (
	green = "\x033"
	red   = "\x035"
)

func (c *Connection) String() string {
	return pretty.Sprint(c)
}
func (irc IRC) String() string {
	return pretty.Sprint(irc)
}
