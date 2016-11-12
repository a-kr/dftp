BIN=dftp

.PHONY: $(BIN) test run fmt clean

export GOPATH:=$(shell pwd)/_vendor:$(shell pwd)
export CWD:=$(shell pwd)

$(BIN):
	go install $(BIN)

run: $(BIN)
	bin/$(BIN) --dfsroot=/storage/www

_vendor:
	go get github.com/goftp/server

clean:
	rm -f bin/$(BIN)

test:
	go test $(BIN)

fmt:
	go fmt $(BIN)
