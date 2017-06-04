export CGO_ENABLED=1
all:
	test -f plugin.so || CGO_ENABLED=1 make -C plugins 
	test -f plugin.so && echo plugin already exists || mv plugins/plugin.so plugin.so	
	@echo building irc client
	CGO_ENABLED=1 go build -o ircb github.com/aerth/ircb/cmd/ircb
