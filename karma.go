package ircb

import (
	"strings"
)

func commandKarma(c *Connection, irc *IRC) {
	if len(irc.Arguments) != 1 {
		irc.Reply(c, c.karmaShow(irc.ReplyTo))
		return
	}

	irc.Reply(c, c.karmaShow(irc.Arguments[0]))
}
func (c *Connection) parseKarma(input string) (handled bool) {
	handled = false
	split := strings.Split(input, " ")
	if len(split) < 1 {
		return false
	}

	if len(split) > 1 {
		if strings.Contains(input, "thank") {
			if i := strings.Index(input, ":"); i != -1 && i != 0 {
				c.Log.Println("Karma:", input[0:i])
				c.karmaUp(input[0:i])
				return true
			}
			return false
		}
		return false
	}

	if strings.HasSuffix(input, "+") {
		c.karmaUp(strings.Replace(input, "+", "", -1))
		return true
	}

	if strings.HasSuffix(input, "-") {
		c.karmaDown(strings.Replace(input, "-", "", -1))
		return true
	}
	return false
}
