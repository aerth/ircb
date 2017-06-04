package ircb

import (
	"fmt"
	"strings"
)

// IRC is a parsed message received from IRC server
type IRC struct {
	Raw       string   // As received
	Verb      string   // Using 'Verb' because we took 'Command' :)
	ReplyTo   string   // From user or channel
	To        string   // can be c.config.Nick
	Channel   string   // From channel (can be user)
	IsCommand bool     // Is a public command
	IsWhisper bool     // Is not from channel
	Message   string   // Parsed message (would still include command prefix)
	Command   string   // Parsed command (stripped of command prefix, first word)
	Arguments []string // Parsed arguments (can be nil)

}

// Encode prepares an IRC message to be sent to server
func (irc IRC) Encode() []byte {
	return []byte(fmt.Sprintf("PRIVMSG %s :%s\r\n", irc.To, irc.Message))
}

// ReplyUser doesnt send to #channel, only sends
func (irc *IRC) ReplyUser(c *Connection, s string) {
	if strings.Contains(irc.ReplyTo, "#") || strings.TrimSpace(s) == "" {
		c.Log.Println("should not use ReplyUser for channel")
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

// Parse input string into IRC struct. To parse fully, use config method cfg.Parse(input string)
// 	:Name COMMAND parameter list
// Where list could begin with ':', which states the rest of list is just one item
// Sending, we use this format:
//	:COMMAND argument :string\r\n
//	:PRIVMSG ##ircb :hello world\r\n
func Parse(input string) *IRC {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	var irc = new(IRC)
	irc.Raw = input
	input = strings.TrimPrefix(input, ":")
	// split input by spaces
	s := strings.Split(input, " ")
	switch len(s) {

	case 1:
		// for receiving
		irc.Verb = s[0]
		// for sending
		irc.Message = s[0]
		return irc
	case 2:
		irc.Message = s[1]
		// for receiving
		irc.Verb = s[0]
		// for sending
		irc.To = s[0]
		return irc
	case 3:
		irc.ReplyTo = strings.Split(s[0], "!")[0]
		irc.Channel = s[0]
		irc.Verb = s[1]
		irc.To = s[2]
		return irc
	default:
		irc.ReplyTo = strings.Split(s[0], "!")[0]
		irc.Channel = s[0]
		irc.Verb = s[1]
		irc.To = s[2]

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
	}
	return irc
}

func (cfg Config) Parse(input string) *IRC {
	irc := Parse(input)
	fmt.Println("definitely parsing:", irc)
	// Add IsWhisper
	irc.IsWhisper = irc.To == cfg.Nick

	// What is a command?
	// > anything with commandprefix gets split into irc.Command and irc.Arguments
	if strings.HasPrefix(irc.Message, cfg.CommandPrefix) && len(irc.Message) > 1 {
		cmd := strings.TrimPrefix(irc.Message, cfg.CommandPrefix) // trim prefix
		args := strings.Split(cmd, " ")                           // split words
		irc.Command = args[0]                                     // first word
		if len(args) > 1 {
			irc.Arguments = args[1:] // rest of words if they exist
		}

		// Add "IsCommand"
		irc.IsCommand = irc.Command != ""
	}

	return irc
}

const formatauth = "NickServ!NickServ@services. NOTICE %s :%s ACC 3" // botname mastername
const formatauth2 = "STATUS %s 1 "                                   // mastername
