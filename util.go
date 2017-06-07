package ircb

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aerth/spawn"
	"github.com/boltdb/bolt"
)

const (
	Green     = "\x033"
	Red       = "\x035"
	Purple    = "\x036"
	Yellow    = "\x038"
	GreenBold = "\x039"
)

// Respawn closes connections after executing self, can be called at any time.
func (c *Connection) Respawn() {
	spawn.Spawn()
	c.Close()
}

// ErrNoPluginSupport when compiled with no CGO or without 'plugins' tag
var ErrNoPluginSupport = fmt.Errorf("no plugin support")

// ErrNoPlugin when plugin is not found
var ErrNoPlugin = fmt.Errorf("plugin not found")

// ErrPluginInv when plugin does not have proper Init func
var ErrPluginInv = fmt.Errorf("invalid plugin")

// PluginInitFunc gets called when plugin is loaded. Init(c *Connection) error
type PluginInitFunc (func(c *Connection) error)

// LoadPlugin loads the named plugin file
// This is a stub, and should be replaced if ircb is built with plugin support
var LoadPlugin = func(c *Connection, s string) error {
	return ErrNoPluginSupport
}

func openlogfile() (f *os.File, err error) {
	return os.OpenFile(".log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
}

var dbkarma = []byte("karma")
var dbdef = []byte("dictionary")
var dbhistory = []byte("history")

// opendb, make buckets if not exist
func loadDatabase(filename string) (*bolt.DB, error) {
	db, err := bolt.Open(filename, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin(true)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	// make karma bucket
	_, err = tx.CreateBucketIfNotExists(dbkarma)
	if err != nil {
		return nil, err
	}
	// make dictionary bucket
	_, err = tx.CreateBucketIfNotExists(dbdef)
	if err != nil {
		return nil, err
	}
	// make history bucket
	_, err = tx.CreateBucketIfNotExists(dbhistory)
	if err != nil {
		return nil, err
	}

	// write db
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return db, nil
}
func (c *Connection) getDefinition(word string) (definition string) {
	tx, err := c.boltdb.Begin(false)
	if err != nil {
		c.Log.Println("database error:", err)
		return ""
	}

	defer tx.Rollback()
	bucket := tx.Bucket(dbdef)
	val := bucket.Get([]byte(word))
	return string(val)
}
func (c *Connection) databaseDefine(word, definition string) error {
	tx, err := c.boltdb.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	bucket := tx.Bucket(dbdef)
	if err != nil {
		return err
	}

	if err := bucket.Put([]byte(word), []byte(definition)); err != nil {
		return err
	}

	return tx.Commit()

}
func (c *Connection) karmaDown(name string) error {
	err := c.boltdb.Update(func(tx *bolt.Tx) error {
		defer tx.Rollback()
		bucket := tx.Bucket(dbkarma)
		current := bytes2int(bucket.Get([]byte(name)))
		if err := bucket.Put([]byte(name), int2bytes(current-1)); err != nil {
			return err
		}

		return nil
	})
	return err
}

func (c *Connection) karmaUp(name string) error {
	err := c.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(dbkarma)
		current := bytes2int(bucket.Get([]byte(name)))
		if err := bucket.Put([]byte(name), int2bytes(current+1)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		println(err)
	}
	//	return err
	return nil
}

func (c *Connection) karmaShow(name string) string {
	var current string
	err := c.boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(dbkarma)
		if bucket == nil {
			return fmt.Errorf("nil bucket")
		}
		current = strconv.Itoa(bytes2int(bucket.Get([]byte(name))))

		return nil
	})
	if err != nil {
		c.Log.Println("karma error:", err)
	}
	return current
}

// Converts bytes to an int
func bytes2int(b []byte) int {
	if b == nil || len(b) == 0 {
		return 0
	}
	return int(binary.BigEndian.Uint64(b))
}

// Converts int to bytes
func int2bytes(u int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(u))
	return buf
}
