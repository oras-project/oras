#!/bin/sh -e

# Copyright The ORAS Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export ORAS_REGISTRY_PORT="5000"
export ORAS_REGISTRY_HOST="localhost:${ORAS_REGISTRY_PORT}"
export ORAS_REGISTRY_FALLBACK_PORT="6000"
export ORAS_REGISTRY_FALLBACK_HOST="localhost:${ORAS_REGISTRY_FALLBACK_PORT}"

repo_root=$1
if [ -z "${repo_root}" ]; then
    echo "repository root path is not provided."
    echo "Usage"
    echo "  e2e.sh <repo_root> [--clean]"
    exit 1
fi
clean_up=$2

echo " === installing ginkgo  === "
repo_root=$(realpath --canonicalize-existing ${repo_root})
cwd=$(pwd)
cd ${repo_root}/test/e2e && go install github.com/onsi/ginkgo/v2/ginkgo@latest
trap "cd $cwd" EXIT

# start registries
. ${repo_root}/test/e2e/scripts/common.sh

if [ "$clean_up" = '--clean' ]; then
    echo " === setting deferred clean up jobs  === "
    trap "try_clean_up oras-e2e oras-e2e-fallback" EXIT
fi

oras_container_name="oras-e2e"
upstream_container_name="oras-e2e-fallback"
e2e_root="${repo_root}/test/e2e"
echo " === preparing oras distribution === "
run_registry \
  ${e2e_root}/testdata/distribution/mount \
  ghcr.io/oras-project/registry:v1.0.0-rc.4 \
  $oras_container_name \
  $ORAS_REGISTRY_PORT \
  true

echo " === preparing upstream distribution === "
run_registry \
  ${e2e_root}/testdata/distribution/mount_fallback \
  registry:2.8.1 \
  $upstream_container_name \
  $ORAS_REGISTRY_FALLBACK_PORT \
  false

echo " === run tests === "
if ! ginkgo -r -p --succinct suite; then 
  echo " === retriving registry error logs === "
  echo '-------- oras distribution trace -------------'
  docker logs -t --tail 200 $oras_container_name
  echo '-------- upstream distribution trace -------------'
  docker logs -t --tail 200 $upstream_container_name
  exit 1
fi
