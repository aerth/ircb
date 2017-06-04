package ircb

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

func KarmaShow(c *Connection, irc *IRC) {
	if len(irc.Arguments) != 1 {
		irc.Reply(c, c.KarmaShow(irc.ReplyTo))
		return
	}

	irc.Reply(c, c.KarmaShow(irc.Arguments[0]))
}
func (c *Connection) ParseKarma(input string) (handled bool) {
	handled = false
	split := strings.Split(input, " ")
	if len(split) < 1 {
		return false
	}

	if len(split) > 1 {
		if strings.Contains(input, "thank") {
			if i := strings.Index(input, ":"); i != -1 && i != 0 {
				c.Log.Println("Karma:", input[0:i])
				c.KarmaUp(input[0:i])
				return true
			}
			return false
		}
		return false
	}

	if strings.HasSuffix(input, "+") {
		c.KarmaUp(strings.Replace(input, "+", "", -1))
		return true
	}

	if strings.HasSuffix(input, "-") {
		c.KarmaDown(strings.Replace(input, "-", "", -1))
		return true
	}
	return false
}

func LoadBackupKarmaMap(filename string) (map[string]int, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	var m = make(map[string]int)
	if stat.Size() == 0 {
		return m, nil
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Connection) SaveBackupKarmaMap() error {
	b, err := json.Marshal(c.karma)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("karma.backup", b, 0600)
}
