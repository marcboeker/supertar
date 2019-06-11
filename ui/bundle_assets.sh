#!/bin/sh

yarn build
cd dist
go run ../bundle.go . ../../server/assets.go
cd ../