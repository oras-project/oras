#!/bin/sh -e

export ORAS_REGISTRY_PORT="5000"
export ORAS_REGISTRY_HOST="localhost:${ORAS_REGISTRY_PORT}"
export ORAS_REGISTRY_PORT="6000"
export ORAS_REGISTRY_FALLBACK_HOST="localhost:${ORAS_REGISTRY_FALLBACK_PORT}"

repo_root=$1
if [ -z "$repo_root" ]; then
    echo "repository root path is not provided."
    echo "Usage"
    echo "  ci.sh <repo-root>"
    exit 1
fi

  # install deps
repo_root=$(realpath --canonicalize-existing ${repo_root})
cd $repo_root/test/e2e && go install github.com/onsi/ginkgo/v2/ginkgo

  # start registries
source $repo_root/test/e2e/scripts/common.sh
run-registry \
  $repo_root/test/e2e/testdata/distribution/mount \
  ghcr.io/oras-project/registry:v1.0.0-rc.4 \
  oras-e2e \
  $ORAS_REGISTRY_PORT

run-registry \
  $repo_root/test/e2e/testdata/distribution/mount_fallback \
  registry:2.8.1 \
  oras-e2e-fallback \
  $ORAS_REGISTRY_FALLBACK_PORT

# run tests
ginkgo -r -p --succinct suite