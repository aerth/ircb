export CGO_ENABLED=1
all:
	make -C plugins && mv plugins/plugin.so plugin.so
	go build -v -x
