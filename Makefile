APP_VERSION=$(shell cicd/version.sh)
BINARY_NAME=ydbops
BUILD_DIR=bin
TODAY=$(shell date --iso=minutes)

all: build build-macos

lint:
	@echo "Linting code..."
	@go vet ./...

pre-build:
	@mkdir -p $(BUILD_DIR)

build-macos: lint pre-build
	GOOS=darwin GOARCH=amd64 go build  -ldflags='-X github.com/ydb-platform/ydbops/cmd.buildVersion=${APP_VERSION}' -o ${BUILD_DIR}/${BINARY_NAME}_darwin_amd64 main.go
	GOOS=darwin GOARCH=arm64 go build  -ldflags='-X github.com/ydb-platform/ydbops/cmd.buildVersion=${APP_VERSION}' -o ${BUILD_DIR}/${BINARY_NAME}_darwin_arm64 main.go

build: lint pre-build
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build  -ldflags='-X github.com/ydb-platform/ydbops/cmd.buildVersion=${APP_VERSION}' -o ${BUILD_DIR}/${BINARY_NAME} main.go 
	strip bin/${BINARY_NAME}

clear:
	rm -rf bin/${BINARY_NAME}

dep:
	go mod download

docker:
	docker build --force-rm -t $(BINARY_NAME) .

build-in-docker: docker
	docker rm -f $(BINARY_NAME) || true
	docker create --name $(BINARY_NAME) $(BINARY_NAME)
	docker cp '$(BINARY_NAME):/app/bin/' $(BUILD_DIR)
	docker rm -f $(BINARY_NAME)

clean:
	@echo "Cleaning..."
	@rm -Rf $(BUILD_DIR)
