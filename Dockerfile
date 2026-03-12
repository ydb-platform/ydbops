FROM golang:1.23 as builder
ARG APP_VERSION=no-APP_VERSION-supplied-in-buildtime
RUN mkdir /app
WORKDIR /app
COPY . /app
RUN cd /app && make all
