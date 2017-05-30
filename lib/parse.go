package ircb

import (
	"fmt"
	"strings"

	"github.com/kr/pretty"
)

type IRC struct {
	Raw       string // As received
	Verb      string
	ReplyTo   string   // From user or channel
	To        string   // can be c.config.Nick
	Channel   string   // From channel (can be user)
	IsWhisper bool     // Is not from channel
	Message   string   // Parsed message (can include command prefix)
	Command   string   // Parsed command (stripped of command prefix)
	Arguments []string // Parsed arguments (can be nil)

}

func (irc IRC) Encode() []byte {
	return []byte(fmt.Sprintf("PRIVMSG %s :%s\r\n", irc.To, irc.Message))
}

func (irc *IRC) Reply(c *Connection, s string) {
	reply := IRC{
		To:      irc.ReplyTo,
		Message: s,
	}
	c.Send(reply)
}

func (c *Connection) Parse(input string) *IRC {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	var irc = new(IRC)
	irc.Raw = input
	input = strings.TrimPrefix(input, ":")
	s := strings.Split(input, " ")
	irc.Verb = s[0]

	// never happens
	if len(s) == 1 {
		c.Log.Println("unreachable")
		return irc
	}

	irc.ReplyTo = strings.Split(s[0], "!")[0]
	irc.Channel = s[0]
	irc.Verb = s[1]
	if len(s) < 3 {
		return irc

	}

	irc.To = s[2]
	if len(s) < 4 {
		return irc
	}
	irc.Message = s[3]
	// extract message
	for i, v := range s[3:] {
		if strings.HasPrefix(v, ":") {
			// message has colon prefix which marks the end of tokens
			irc.Message = strings.Join(s[i+3:], " ")
			break
		}
	}
	irc.Message = strings.TrimPrefix(irc.Message, ":")
	irc.IsWhisper = s[2] == c.config.Nick

	// is a command?
	if strings.HasPrefix(irc.Message, c.config.CommandPrefix) {
		cmd := strings.TrimPrefix(irc.Message, c.config.CommandPrefix)
		args := strings.Split(cmd, " ")
		irc.Command = args[0]
		if len(args) > 1 {
			irc.Arguments = args[1:]
		}
	}

	// parsed
	c.Log.Println("parsed:", pretty.Sprint(irc))
	return irc
}
