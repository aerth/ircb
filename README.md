# ircb

friendly channel bot

  * karma system
  * http title parser
  * bot master must be identified with services (uses NickServ ACC)

## help

See [docs](https://aerth.github.io/ircb/) and [GoDoc](https://godoc.org/github.com/aerth/ircb/lib/ircb) and visit `##ircb` on Freenode for help

### usage

for upgrade command to function as expected:

```
#!/bin/sh
PATH=$PATH:/usr/local/go/bin # go root
NOWPWD=$PWD # ircb will land here
go get -v -u -d github.com/aerth/ircb
cd go/src/github.com/aerth/ircb 
go build
./ircb
```
