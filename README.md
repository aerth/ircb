# ircb

use flags once, saves in config

irc client

## help

visit ##ircb

### installation

```
#!/bin/sh
PATH=$PATH:/usr/local/go/bin # go root
NOWPWD=$PWD # ircb will land here
go get -v -u -d github.com/aerth/ircb
cd go/src/github.com/aerth/ircb && make && make install PREFIX=$NOWPWD
```
