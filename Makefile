BINARY_NAME=sjrpc

build:
	go build -o bin/${BINARY_NAME} cmd/main.go
	chmod +x bin/${BINARY_NAME}

run: build 
	./bin/${BINARY_NAME}