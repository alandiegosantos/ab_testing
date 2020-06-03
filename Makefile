GOFILES=$(shell find . -name *.go)

TARGET=webserver

all: ${TARGET}

webserver: ${GOFILES} clean
	go mod vendor
	CGO_ENABLED=0 GOFLAGS="-mod=vendor" go build -o $@ -v

.PHONY: clean
clean:
	@rm -f ${TARGET}