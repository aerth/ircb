package ircb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func KarmaShow(c *Connection, irc *IRC) {
	if len(irc.Arguments) != 1 {
		return
	}

	irc.Reply(c, c.karmaShow(irc.Arguments[0]))
}
func (c *Connection) ParseKarma(input string) error {
	split := strings.Split(input, " ")
	if len(split) < 1 {
		return fmt.Errorf("too short: %s", split)
	}

	if len(split) > 1 {
		return fmt.Errorf("too long")
	}

	if strings.HasSuffix(input, "+") {
		c.KarmaUp(strings.Replace(input, "+", "", -1))
		return nil
	}

	if strings.HasSuffix(input, "-") {
		c.KarmaDown(strings.Replace(input, "-", "", -1))
		return nil
	}

	return fmt.Errorf("no karma to parse")
}

func LoadKarmaMap(f *os.File) (map[string]int, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var m = make(map[string]int)
	if stat.Size() == 0 {
		return m, nil
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Connection) SaveKarmaMap() error {
	b, err := json.Marshal(c.karma)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.karmafile.Name(), b, 0700)
}
func (c *Connection) KarmaUp(name string) {
	c.karmalock.Lock()

	defer c.karmalock.Unlock()

	c.Log.Println("karma up:", name)
	if _, ok := c.karma[name]; !ok {
		c.karma[name] = 1
		return
	}
	c.karma[name]++
	if err := c.SaveKarmaMap(); err != nil {
		c.Log.Println("cant save karma map:", err)
	}

}

func (c *Connection) KarmaDown(name string) {
	c.karmalock.Lock()
	defer c.karmalock.Unlock()
	c.Log.Println("karma down:", name)
	if _, ok := c.karma[name]; !ok {
		c.karma[name] = 0
		return
	}
	c.karma[name]--
	if err := c.SaveKarmaMap(); err != nil {
		c.Log.Println("cant save karma map:", err)
	}
}

func (c *Connection) karmaShow(name string) string {
	c.karmalock.Lock()
	defer c.karmalock.Unlock()
	if _, ok := c.karma[name]; !ok {
		return ""
	}
	return strconv.Itoa(c.karma[name])
}
