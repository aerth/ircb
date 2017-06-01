package ircb

import (
	"fmt"
	"strings"
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

// ReplyUser doesnt send to #channel
func (irc *IRC) ReplyUser(c *Connection, s string) {

	if strings.Contains(irc.ReplyTo, "#") || strings.TrimSpace(s) == "" {
		return
	}
	reply := IRC{
		To:      irc.ReplyTo,
		Message: s,
	}

	c.Send(reply)
}

// Reply replies to an irc message, preferring a channel
func (irc *IRC) Reply(c *Connection, s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	reply := IRC{
		To:      irc.ReplyTo,
		Message: s,
	}
	if strings.HasPrefix(irc.To, "#") {
		reply.To = irc.To
	}
	c.Send(reply)
}

// Parse input string into IRC struct. To parse fully we need commandprefix and nickname as well.
func Parse(commandprefix, nick, input string) *IRC {
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
	irc.IsWhisper = s[2] == nick

	// is a command?
	if strings.HasPrefix(irc.Message, commandprefix) {
		cmd := strings.TrimPrefix(irc.Message, commandprefix)
		args := strings.Split(cmd, " ")
		irc.Command = args[0]
		if len(args) > 1 {
			irc.Arguments = args[1:]
		}
	}

	return irc
}
