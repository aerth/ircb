# ircb

[![Build Status](https://travis-ci.org/aerth/ircb.svg?branch=master)](https://travis-ci.org/aerth/ircb)

friendly channel bot

  * karma system
  * http title parser
  * bot master must be identified with services (uses NickServ ACC)

## help

See [docs](https://aerth.github.io/ircb/) and [GoDoc](https://godoc.org/github.com/aerth/ircb/lib/ircb) and visit `##ircb` on Freenode for help

### usage

quick setup (requires [Go](https://golang.org) to compile):

1. log in to the user you would like to use ircb with,
2. change directory to the path you would like ircb to live (can be empty)

```

curl https://raw.githubusercontent.com/aerth/ircb/master/makefile > makefile
make
vim config.json
./ircb

```

#### authentication

send two private messages to bot: `/msg bot !help` and `/msg bot $upgrade`

the first message from master, ircb will check if user is identified with services
if so, you will be 'authenticated' for 5 minutes

the second command, it will try to fetch newest source code and rebuild and redeploy itself
if fails, should private message master reason

#### commands

See p.go in ./plugins/ directory for an example plugin

Build the plugin.so, move next to ircb executable, and send `$update-plugins` command to ircb

