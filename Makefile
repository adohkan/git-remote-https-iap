CMD_PATH := cmd/git-remote-https+iap
CMD_NAME := git-remote-https+iap

BIN_PATH := dist/bin/

version := $(shell git describe --match "v*.*" --abbrev=7 --tags --dirty)
build_args := -ldflags "-X main.version=${version}"

.PHONY: all
all: build

$(BIN_PATH):
	mkdir -p $@

build: $(BIN_PATH)
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${build_args} -o $(BIN_PATH)${CMD_NAME}-darwin-amd64 ${CMD_PATH}/*.go
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${build_args} -o $(BIN_PATH)${CMD_NAME}-linux-amd64 ${CMD_PATH}/*.go

.PHONY: version
version:
	@echo "$(version)"

.PHONY: clean
clean:
	rm -rf $(BIN_PATH)
