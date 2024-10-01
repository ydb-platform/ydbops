FROM golang:1.22 as builder
COPY go /go/pkg/mod
RUN mkdir /app
WORKDIR /app
COPY . /app
RUN cd /app && make build
