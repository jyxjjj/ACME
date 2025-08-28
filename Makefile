BINARY_NAME := ACME
SRC := ./src
BIN_DIR := ./bin

fmt:
	go fmt ./src

tidy: fmt
	go mod tidy

all: linux
	@chmod +x $(BIN_DIR)/$(BINARY_NAME)

linux: tidy
	GOOS=linux GOARCH=amd64 GOAMD64=v3 CGO_ENABLED=0 \
		go build -o $(BIN_DIR)/$(BINARY_NAME) $(SRC)

clean:
	rm -rf $(BIN_DIR)/*

install: all
	./release.sh

test: tidy
	./test.sh
