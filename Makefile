BINARY_NAME=sjrpc
VERSION=v0.1.3

build:
	go build -o bin/${BINARY_NAME} cmd/main.go
	chmod +x bin/${BINARY_NAME}

build-mac-m1:
	GOOS=darwin GOARCH=arm64  go build -o downloads/${BINARY_NAME}-${VERSION}-mac-silicon cmd/main.go

build-mac-intel:
	GOOS=darwin GOARCH=amd64  go build -o downloads/${BINARY_NAME}-${VERSION}-mac-intel cmd/main.go

build-linux:
	GOOS=linux GOHOSTOS=linux GOARCH=amd64 go build -o downloads/${BINARY_NAME}-${VERSION}-linux-amd64 cmd/main.go

build-windows:
	GOOS=windows GOARCH=amd64 go build -o downloads/${BINARY_NAME}-${VERSION}-windows-amd64.exe cmd/main.go

build-distro: clean-distro build-mac-m1 build-mac-intel build-linux build-windows

clean-distro:
	rm downloads/*

clean:
	rm database/data/*

run: build 
	./bin/${BINARY_NAME}

