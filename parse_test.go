package ircb

import (
	"bytes"
	"log"
	"os"
	"testing"
)

var testconfig = &Config{
	Nick:          "testing",
	Master:        "tester",
	CommandPrefix: "!",
}
var buf = new(bytes.Buffer)

type testconnection struct {
	buf *bytes.Buffer
	log *log.Logger
}

func (t *testconnection) Close() error {

	return nil
}

func (t *testconnection) Write(b []byte) (n int, err error) {
	return t.buf.Write(b)
}

func (t *testconnection) Read(b []byte) (n int, err error) {
	return t.buf.Read(b)
}

func NewTestConnection() *Connection {
	var tc = new(testconnection)
	tc.buf = new(bytes.Buffer)
	tc.log = log.New(os.Stderr, "testnet:", log.Lshortfile)
	return &Connection{
		Log:    log.New(os.Stderr, "conn:", log.Lshortfile),
		conn:   tc,
		config: testconfig,
	}

}

func TestTest(t *testing.T) {
	_, err := NewTestConnection().Write([]byte("PING"))
	if err != nil {
		t.Fail()
		t.Log(err)
	}
}

func TestParse(t *testing.T) {
	c := NewTestConnection()
	c.config.CommandPrefix = "!"
	irc := c.config.Parse("foo PRIVMSG :bar")
	if irc.Verb != "PRIVMSG" {
		t.Logf("expected verb: PRIVMSG, got %q", irc.Verb)
		t.Fail()
	}

	irc = c.config.Parse("FOO")
	if irc.Verb != "FOO" {
		t.Logf("expected verb: FOO, got %q", irc.Verb)
		t.Fail()
	}

	testcases := []struct {
		expected, input string
	}{
		{"433", ":host.test 433 * mustangsally :Nickname is already in use\r\n"},
		{"451", ":oragono.test 451 * :You need to register before you can use that command\r\n"},
		{"PING", "PING mustangsally\r\n"},
		{"PRIVMSG", "mustangsally!ok@ok PRIVMSG #ok :hello"},
	}

	for _, test := range testcases {
		if out := testconfig.Parse(test.input).Verb; out != test.expected {
			t.Logf("wanted: %q", test.expected)
			t.Logf("but got: %q", out)
		}
	}
	testcases = []struct {
		expected, input string
	}{
		{"Nickname is already in use", ":host.test 433 * mustangsally :Nickname is already in use\r\n"},
		{"You need to register before you can use that command", ":oragono.test 451 * :You need to register before you can use that command\r\n"},
		{"mustangsally", "PING mustangsally\r\n"},
		{"hello", "mustangsally!ok@ok PRIVMSG #ok :hello"},
	}

	for _, test := range testcases {
		if out := testconfig.Parse(test.input).Message; out != test.expected {
			t.Logf("wanted: %q", test.expected)
			t.Logf("but got: %q", out)
		}
	}

}
