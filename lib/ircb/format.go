package ircb

import "github.com/kr/pretty"

func (c *Connection) String() string {
	return pretty.Sprint(c)
}
func (irc IRC) String() string {
	return pretty.Sprint(irc)
}
