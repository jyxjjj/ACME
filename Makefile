BINARY_NAME := ACME
SRC := ./src
BIN_DIR := ./bin

fmt:
	go fmt ./src

tidy: fmt
	go mod tidy

all: linux darwin
	@chmod +x $(BIN_DIR)/$(BINARY_NAME) $(BIN_DIR)/$(BINARY_NAME)-darwin

linux: tidy
	GOOS=linux GOARCH=amd64 GOAMD64=v3 CGO_ENABLED=0 \
		go build -o $(BIN_DIR)/$(BINARY_NAME) $(SRC)

darwin: tidy
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
		go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin $(SRC)

clean:
	rm -rf $(BIN_DIR)/*

install: all
	./release.sh
