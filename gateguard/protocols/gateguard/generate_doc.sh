#!/bin/sh

protoc --plugin="$GOPATH"/src/github.com/pseudomuto/protoc-gen-doc/protoc-gen-doc -I ./ --doc_out=. --doc_opt=markdown,README.md *.proto