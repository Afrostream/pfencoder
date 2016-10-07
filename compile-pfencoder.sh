#!/bin/sh

export GOPATH=/go/src/pfencoder
export GOOS=linux
export GOARCH=amd64
export GOBIN=/go/app

cd /go/src/pfencoder
go get -v
