#!/bin/sh
# Build dfd, then run the ob-dfd ERT suite against that binary.
set -e
cd "$(dirname "$0")"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
go build -o "$tmp/dfd" ../cmd/dfd
DFD_BIN="$tmp/dfd" emacs -Q --batch -L . -l ob-dfd.el -l ob-dfd-tests.el \
  -f ert-run-tests-batch-and-exit
