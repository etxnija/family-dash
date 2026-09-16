#!/bin/sh
set -e
go test ./... -race -count=1
