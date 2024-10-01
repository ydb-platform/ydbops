#!/bin/bash

docker run --name ydbops-build \
 -v $(pwd):/usr/local/go/src/ydbops \
 -w /usr/local/go/src/ydbops golang:1.22 make build
docker rm -f ydbops-build

exit 0