#!/bin/bash -ex

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

# Cleanup from previous runs
rm -f hello.txt
rm -f bin/oras-acceptance || true
docker rm -f oras-acceptance-registry || true

# Build the example into a binary
CGO_ENABLED=0 go build -v -o bin/oras-acceptance ./examples/

# Run a test registry and expose at localhost:5000
trap "docker rm -f oras-acceptance-registry" EXIT
docker run -d -p 5000:5000 \
  --name oras-acceptance-registry \
  index.docker.io/registry

# Run the example binary
bin/oras-acceptance

# Ensure hello.txt exists and contains expected content
grep '^Hello World!$' hello.txt
