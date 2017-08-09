#!/bin/sh

export GOPATH=/go/
export GOOS=linux
export GOARCH=amd64
export GOBIN=/go/app/

cd /go/src/pfencoder
go get -v
go install
