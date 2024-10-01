BINARY_NAME=ydbops
TODAY=$(shell date --iso=minutes)

build:
	go get -u
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build  -ldflags='-X main.buildInfo=${TODAY}' -o bin/${BINARY_NAME} main.go 
	strip bin/${BINARY_NAME}
clear:
	rm -rf bin/${BINARY_NAME}

dep:
	go mod download

