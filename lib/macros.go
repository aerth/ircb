package ircb

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

func LoadBackupDefinitions(filename string) (map[string]string, error) {
	m := make(map[string]string)
	if _, err := os.Stat(filename); err != nil {
		if strings.Contains(err.Error(), "no such") {

			return m, ioutil.WriteFile(filename, []byte("{}"), 0600)

		}
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)

	return m, err
}

func (c *Connection) BackupDefinitions() error {
	b, err := json.Marshal(c.definitions)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("dictionary.backup", b, 0600)
}
