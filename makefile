export CGO_ENABLED=1
all:
	CGO_ENABLED=1 make -C plugins && mv plugins/plugin.so plugin.so	
	CGO_ENABLED=1 go build 
