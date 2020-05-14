#!/bin/bash

# Why we are wrapping gofmt?
# - ignore files in vendor direcotry
# - gofmt doesn't exit with error code when there are errors
err_files=$(find . -path ./vendor -prune -o -name '*.go' -print | xargs gofmt -l)

if [ -z "$err_files" ]; then
	echo "gofmt OK"
else
	echo "gofmt ERROR - These files are not formated by gofmt:"
	echo "$err_files"
	exit 1
fi
