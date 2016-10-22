BIN=dftp

.PHONY: $(BIN) test run fmt clean

export GOPATH:=$(shell pwd)
export CWD:=$(shell pwd)

$(BIN):
	go install $(BIN)

run: $(BIN)
	bin/$(BIN) --dfsroot=/storage/www

clean:
	rm -f bin/$(BIN)

test:
	go test $(BIN)

fmt:
	go fmt $(BIN)
