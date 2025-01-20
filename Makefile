BINARY_NAME=ydbops
BUILD_DIR=bin
PREFIX ?= ~/ydb/bin

TODAY=$(shell date --iso=minutes)
GIT_COMMIT=$(shell git rev-parse HEAD)

# Detect OS and architecture
ifeq ($(OS),Windows_NT)
    SYSTEM_OS = win32
    ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
        SYSTEM_ARCH = amd64
    endif
    ifeq ($(PROCESSOR_ARCHITECTURE),x86)
        SYSTEM_ARCH = x86
    endif
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        SYSTEM_OS = linux
    endif
    ifeq ($(UNAME_S),Darwin)
        SYSTEM_OS = darwin
    endif
    UNAME_P := $(shell uname -p)
    ifeq ($(UNAME_P),x86_64)
        SYSTEM_ARCH = amd64
    endif
    ifneq ($(filter %86,$(UNAME_P)),)
        SYSTEM_ARCH = x86
    endif
    ifneq ($(filter arm%,$(UNAME_P)),)
        SYSTEM_ARCH = arm64
    endif
endif

# APP_VERSION gets supplied from outside in CI as an env variable.
# By default you can still build `ydbops` manually without specifying it, 
# but there won't be a nice tag in `--version`. There will be a commit SHA anyway.
APP_VERSION ?= no-APP_VERSION-supplied-in-buildtime
LDFLAGS="-X github.com/ydb-platform/ydbops/cmd/version.BuildVersion=$(APP_VERSION) -X github.com/ydb-platform/ydbops/cmd/version.BuildTimestamp=${TODAY} -X github.com/ydb-platform/ydbops/cmd/version.BuildCommit=${GIT_COMMIT}"

all: build build-macos

lint:
	@echo "Linting code..."
	@go vet ./...

pre-build:
	@mkdir -p $(BUILD_DIR)

build-macos: lint pre-build
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}_darwin_amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags=${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}_darwin_arm64 main.go

build: lint pre-build
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} main.go
	strip ${BUILD_DIR}/${BINARY_NAME}

clear:
	rm -rf ${BUILD_DIR}/${BINARY_NAME}

dep:
	go mod download

docker:
	docker build --force-rm -t $(BINARY_NAME) --build-arg APP_VERSION=${APP_VERSION} .

test:
	if [ "$(shell uname)" = "Linux" ]; then make build; else make build-macos; fi
	ginkgo test ./...

build-in-docker: clear docker
	docker rm -f $(BINARY_NAME) || true
	docker create --name $(BINARY_NAME) $(BINARY_NAME)
	docker cp '$(BINARY_NAME):/app/bin/' $(BUILD_DIR)
	docker rm -f $(BINARY_NAME)

install:
ifeq ($(UNAME_S),Darwin)
	cp ${BUILD_DIR}/${BINARY_NAME}_$(SYSTEM_OS)_$(SYSTEM_ARCH) $(PREFIX)/ydbops
else
	cp ${BUILD_DIR}/${BINARY_NAME} $(PREFIX)/ydbops
endif
	chmod +x $(PREFIX)/ydbops