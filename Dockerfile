FROM golang:1.22 as builder
RUN mkdir /app
WORKDIR /app
COPY . /app
RUN cd /app && make all
