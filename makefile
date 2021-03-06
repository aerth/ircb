export CGO_ENABLED=1
GOPATH:=${shell go env GOPATH}
IRCB=${GOPATH}/src/github.com/aerth/ircb
define GetPlugins:
endef
rebuild:
	@echo building irc client
	CGO_ENABLED=1 go get -v -d github.com/aerth/ircb/cmd/ircb
	CGO_ENABLED=1 go build -tags plugins -v -o ircb github.com/aerth/ircb/cmd/ircb
	@echo built: ./ircb
	@test -f config.json || ( cp -nv ${IRCB}/default.json config.json && echo "new default config" )

all: rebuild plugins

plugins:
	## plugins go here
	CGO_ENABLED=1 go get -v -d github.com/aerth/ircb-plugins/...
	@echo $(foreach plugin,${shell go list github.com/aerth/ircb-plugins/...}, go build -o ${shell basename $(plugin)}.so -buildmode=plugin $(plugin);)

run:
	test -x ./ircb || ${MAKE} rebuild
	test -x ./ircb || exit 111
	test -f config.json || ( cp -nv ${IRCB}/default.json config.json && echo "new default config" )
	./ircb

test:
	CGO_ENABLED=1 go test -race -v ./...

static: fast

fast:
	CGO_ENABLED=0 go install -v
	CGO_ENABLED=0 go build -v -ldflags='-w -s' -o ircb ./cmd/ircb
