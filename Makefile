.PHONY: build build-server clean run-server run-client

BINARY_CLIENT=parcel
BINARY_SERVER=parcel-server

build:
	go build -o bin/$(BINARY_CLIENT) ./client/...

build-server:
	go build -o bin/$(BINARY_SERVER) ./server/...

clean:
	rm -rf bin/

run-server:
	go run ./server/main.go

run-client:
	go run ./client/main.go

tidy:
	go mod tidy
e