

.PHONY:  ALL tools

build_date = $(shell date '+%Y-%m-%d %H:%M:%S')
version = $(shell git describe --tags --always | sed 's/-/+/' | sed 's/^v//')
goversion = $(shell go version)
ldflags = -ldflags "-X 'main.Version=$(version)' -X 'main.BuildDate=$(build_date)' -X 'main.GoVersion=$(goversion)'"

ALL:


tools: gen_broccoli


gen_broccoli:
	GOOS=linux go build $(ldflags) -o tools/bin/ ./tools/gen-broccoli
	GOOS=windows go build $(ldflags) -o tools/bin/ ./tools/gen-broccoli

errdef:
	gen-broccoli -onlybroccolierr -errdef errors/errdef.proto -dest .
