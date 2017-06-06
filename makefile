export CGO_ENABLED=1
GOPATH:=${shell go env GOPATH}
IRCB=${GOPATH}/src/github.com/aerth/ircb
rebuild:
	test -f plugin.so || CGO_ENABLED=1 make -C ${IRCB}/plugins/
	test -f plugin.so && echo plugin already exists || \
		mv -nv ${IRCB}/plugins/plugin.so plugin_new.so	
	@echo building irc client
	CGO_ENABLED=1 go get -v -d github.com/aerth/ircb/cmd/ircb
	CGO_ENABLED=1 go install -v .
	CGO_ENABLED=1 go build -v -o ircb github.com/aerth/ircb/cmd/ircb


	test -f config.json || ( cp -nv ${IRCB}/default.json config.json && echo "new default config" )
plugin:
	CGO_ENABLED=1 make -C ${IRCB}/plugins/
	mv -v ${IRCB}/plugins/plugin.so plugin.so     

all: plugin rebuild
	@echo complete
	
run:
	test -x ./ircb || ${MAKE} rebuild
	test -x ./ircb || exit 111
	test -f config.json || ( cp -nv ${IRCB}/default.json config.json && echo "new default config" )
	./ircb

test:
	CGO_ENABLED=1 go test -race -v ./...

fast:
	CGO_ENABLED=0 go install -v
	CGO_ENABLED=0 go build -v -o ircb ./cmd/ircb
