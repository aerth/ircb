package ircb

import (
	"fmt"
	"strconv"
	"strings"
)

// IRC message
type IRC struct {
	Host             string
	Channel          string
	From             string
	FromFull         string
	Verb             string
	Message          string
	Command          string // ircb internal commands
	Raw              string
	CommandArguments []string
}

// ParseIRC parses an IRC message
func ParseIRC(s string, cmdprefix string) (irc IRC) {
	irc.Raw = s
	words := strings.Split(s, " ")
	switch len(words) {
	// never happens
	case 1:
		irc.Message = s
		irc.Verb = s
		irc.From = s
		irc.Channel = s

	// pings
	case 2:
		irc.Verb = words[0]
		irc.Message = words[1]
		irc.From = words[1]
		irc.Channel = words[1]

	// never happens
	case 3:
		irc.From = words[0]
		irc.Verb = words[1]
		irc.Channel = words[2]
		irc.Message = words[2]

	// every message not a ping
	default:
		irc.From = strings.TrimPrefix(strings.Split(words[0], "!")[0], ":") // usually host , channel or user.
		irc.Verb = words[1]                                                 // PRIVMSG, NOTICE
		irc.Channel = words[2]                                              // can be c.Config.Name
		irc.Message = strings.TrimPrefix(strings.Join(words[3:], " "), ":") // rest of message
	}

	irc.Channel = clean(irc.Channel)
	irc.From = clean(irc.From)
	irc.FromFull = clean(irc.FromFull)
	irc.Host = clean(irc.Host)
	irc.Message = clean(irc.Message)

	if strings.HasPrefix(irc.Message, cmdprefix) {
		irc.CommandArguments = strings.Fields(strings.TrimPrefix(irc.Message, cmdprefix))
		irc.Command = irc.CommandArguments[0]
	}

	return
}

func clean(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(s, "\r\n"))
}

// Handle an IRC message, only INT verbs, and PRIVMSG verbs for now
func (c *Connection) HandleIRC(irc IRC) {
	// probably a number
	verbint, err := strconv.Atoi(irc.Verb)
	if err == nil {
		if c.HandleVerbINT(verbint, irc) {
			return
		}
	}

	// not a number verb
	switch irc.Verb {
	default:
		c.Logf("New Verb %q - Message %q", irc.Verb, irc.Message)
	case PRIVMSG:
		if irc.Command != "" {
			c.HandlePRIVMSG(irc)
		} else if strings.Contains(irc.Message, c.Config.Name){
			c.WriteMaster(green.Sprintf("%s [%s] %q", irc.Channel, irc.From, irc.Message))
		}
	case ":Closing":
			quit()
	case "QUIT", "PART": //
	case "JOIN":
		if _, ok := c.Config.owners[getuser(irc.From)]; ok {
				c.WriteMaster(fmt.Sprintf("OP %q in %q", irc.From, irc.Channel))
				c.Writer <- fmt.Sprintf("MODE %s +o %s", irc.Channel, getuser(irc.From))
		}
	case "353": // channel users
		var channel string
		names := strings.Split(irc.Message, ":")
		if len(names) > 1 { // not 1 name, but if contains : and has stuff after it
			channel = names[0]
			names = names[1:]
			c.WriteMaster(blue.Sprint(strings.TrimPrefix(channel, "= "))+"// "+green.Sprint(names))
		}
	case "366": //
	case "MODE":
		mode := irc.Message
		c.WriteMaster(blue.Sprint(strings.TrimPrefix(irc.Channel, "= "))+" GOT MODE "+green.Sprint(mode))

	}

}
