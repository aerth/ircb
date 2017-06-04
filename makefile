export CGO_ENABLED=1
GOPATH:=${shell go env GOPATH}
IRCB=${GOPATH}/src/github.com/aerth/ircb
rebuild:
	test -f plugin.so || CGO_ENABLED=1 make -C ${IRCB}/plugins/
	test -f plugin.so && echo plugin already exists || \
		mv ${IRCB}/plugins/plugin.so plugin.so	
	@echo building irc client
	CGO_ENABLED=1 go build -o ircb github.com/aerth/ircb/cmd/ircb
	test -f config.json || cp ${IRCB}/default.json config.json && echo "new default config"

run:
	test -x ./ircb || ${MAKE} rebuild
	test -x ./ircb || exit 111
	test -f config.json || cp ${IRCB}/default.json config.json && echo "new default config"
	./ircb

