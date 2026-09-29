#!/bin/bash -e

# Install deps
gobinpath="$(go env GOPATH)/bin"
curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b "$gobinpath" v2.11.4
export PATH=$gobinpath:$PATH

# Run verify steps
make verify
