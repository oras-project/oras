#!/bin/bash -ex

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

LOCAL_REGISTRY_HOSTNAME="${LOCAL_REGISTRY_HOSTNAME:-localhost}"

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

# Wait for a connection to port 5000 (timeout after 1 minute)
WAIT_TIME=0
while true; do
  if nc -w 1 -z "${LOCAL_REGISTRY_HOSTNAME}" 5000; then
    echo "Able to connect to ${LOCAL_REGISTRY_HOSTNAME} port 5000"
    break
  else
    if (( ${WAIT_TIME} >= 60 )); then
      echo "Timed out waiting for connection to ${LOCAL_REGISTRY_HOSTNAME} on port 5000. Exiting."
      exit 1
    fi
    echo "Waiting to connect to ${LOCAL_REGISTRY_HOSTNAME} on port 5000. Sleeping 5 seconds.."
    sleep 5
    WAIT_TIME=$((WAIT_TIME + 5))
  fi
done

# Run the example binary
bin/oras-acceptance

# Ensure hello.txt exists and contains expected content
grep '^Hello World!$' hello.txt
