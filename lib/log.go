package ircb

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/kr/pretty"
)

/*
 *
 * ircb Copyright 2017  aerth <aerth@riseup.net>
 * log.go
 *
 * logs all input/output to stderr and a file.
 * filters and colorizes stderr output
 *
 */

var green = color.New(color.FgGreen)
var red = color.New(color.FgRed)
var orange = color.New(color.FgRed)
var cyan = color.New(color.FgCyan)
var blue = color.New(color.FgBlue)
var clr0 = color.New(color.Reset)
var clrgood = clr0.Sprint("[") + green.Sprint("*") + "]"
var clrbad = clr0.Sprint("[") + red.Sprint("*") + "]"
var clralert = clr0.Sprint("[") + orange.Sprint("*") + "]"

func alert(i ...interface{}) {
	orange.Fprintln(os.Stderr, i...)
}
func alertf(f string, i ...interface{}) {
	orange.Fprintf(os.Stderr, f, i...)
}
func info(i ...interface{}) {
	green.Fprintln(os.Stderr, i...)
}
func infof(f string, i ...interface{}) {
	green.Fprintf(os.Stderr, f, i...)
}

func doreport(netlog []string) {
	var report []byte
	for _, v := range netlog {
		if strings.HasPrefix(v, "\n") {
			v = strings.TrimPrefix(v, "\n")
			v = strings.TrimPrefix(v, "\n")
		}
		report = append(report, []byte(v+"\n")...)
	}
	if report != nil {
		file, _ := ioutil.TempFile("./logs/", "ircb_")
		filename := file.Name()
		if filename != "" {
			ioutil.WriteFile(filename, report, 644)
		}
		fmt.Fprintln(os.Stderr, "ircb: log to", filename)
	}
}

// Log to stderr and logfile
func (c *Connection) Log(i ...interface{}) {

	if i == nil { // skip nil
		return
	}

	// add new line if needed
	str := fmt.Sprint(i...)
	if !strings.HasSuffix(str, "\n") {
		str += "\n"
	}
	// print to stderr
	if c.Config.Verbose {
		fmt.Fprint(os.Stderr, str)
	}

	// write logfile
	c.logfile.WriteString(str)
}

// Logf to stderr and logfile
func (c *Connection) Logf(f string, i ...interface{}) {
	// add new line if needed
	if !strings.HasSuffix(f, "\n") {
		f += "\n"
	}
	if c.Config.Verbose {
		fmt.Fprintf(os.Stderr, f, i...)
	}

	// write logfile
	c.logfile.WriteString(fmt.Sprintf(f, i...))
}

func (c *Connection) openlogfile() {
	var err error
	os.Mkdir("logs", 0750)
	boottime := strconv.FormatInt(c.Config.Boottime.Unix(), 10)
	logfilename := "logs/" + boottime + ".log"
	c.logfile, err = os.OpenFile(logfilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		fmt.Println(err)
		fmt.Println("using stderr")
		c.logfile = os.Stderr
	}
}

func errorf(f string, i ...interface{}) {
	fmt.Fprintf(os.Stderr, f, i...)
}

func randomcolor() *color.Color {
	colorlist := []color.Attribute{color.FgBlue, color.FgCyan, color.FgGreen,
		color.FgMagenta, color.FgRed, color.FgHiBlue, color.FgHiCyan,
		color.FgHiGreen, color.FgHiMagenta, color.FgHiRed}
	clr := color.New(colorlist[rand.Intn(len(colorlist)-1)])
	return clr
}

func rnbo(s string) string {
	runes := []rune(s)
	var out string
	for _, v := range runes {
		ran := randomcolor().Sprint(string(v))
		out += ran
	}
	return out
}

// String config
func (config *Config) String() string {
	return fmt.Sprintf("%# v\n", pretty.Formatter(config))
}

// ListCommands string
func (config *Config) ListCommands() string {
	var commands string
	commands += fmt.Sprintf("Total: %v\n", len(config.Commands))
	for key := range config.Commands {
		commands += fmt.Sprintf("Command: %s\n", key)
	}
	return commands
}

// ListMasterCommands returns a string
func (config *Config) ListMasterCommands() string {
	var commands string
	commands += fmt.Sprintf("*Total: %v\n", len(config.MasterCommands))
	for key := range config.MasterCommands {
		commands += fmt.Sprintf("*Command: %s\n", key)
	}
	return commands
}

// ListMasterTools returns a list of tools in the tool dir
func (config *Config) ListMasterTools() string {
	var tools string
	tools += fmt.Sprintf("*Tools: %v\n", len(config.MasterTools))
	for tool := range config.MasterTools {
		tools += fmt.Sprintf("*Tool: %s\n", tool)
	}
	return tools
}

// ListTools returns a list of tools in the tool dir
func (config *Config) ListTools() string {
	var tools string
	tools += fmt.Sprintf("*Tools: %v\n", len(config.Tools))
	for tool := range config.Tools {
		tools += fmt.Sprintf("*Tool: %s\n", tool)
	}
	return tools
}

func randint() int {
	return rand.Intn(9)
}
func randstr() string {
	return strconv.Itoa(randint())
}
