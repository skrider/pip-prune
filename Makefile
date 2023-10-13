OUT=$(shell pwd)/build

build:
	go build -o $(OUT)/pip-prune cmd/pip-prune/main.go 
.PHONY: build

