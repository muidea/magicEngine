# Makefile for TCP server and client
.PHONY: all server client clean

all: server client

server:
	@mkdir -p bin
	go build -o bin/tcp-server ./server/main.go

client:
	@mkdir -p bin
	go build -o bin/tcp-client ./client/main.go

clean:
	@rm -rf bin