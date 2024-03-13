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
export ZOT_REGISTRY_PORT="7000"
export ZOT_REGISTRY_HOST="localhost:${ZOT_REGISTRY_PORT}"
export ORAS_CTR_NAME="oras-e2e"
export UPSTREAM_CTR_NAME="oras-e2e-fallback"
export ZOT_CTR_NAME="oras-e2e-zot"

repo_root=$1
if [ -z "${repo_root}" ]; then
    echo "repository root path is not provided."
    echo "Usage"
    echo "  prepare.sh <repo_root>"
    exit 1
fi

echo " === installing ginkgo  === "
repo_root=$(realpath --canonicalize-existing ${repo_root})
cwd=$(pwd)
cd ${repo_root}/test/e2e && go install github.com/onsi/ginkgo/v2/ginkgo@latest
trap "cd $cwd" EXIT

# start registries
. ${repo_root}/test/e2e/scripts/common.sh
echo " >>> preparing: oras distribution >>> "
e2e_root="${repo_root}/test/e2e"
run_registry \
  ${e2e_root}/testdata/distribution/mount \
  ghcr.io/oras-project/registry:v1.0.0-rc.4 \
  $ORAS_CTR_NAME \
  $ORAS_REGISTRY_PORT
echo " <<< prepared : oras distribution <<< "

echo " >>> preparing: upstream distribution >>> "
run_registry \
  ${e2e_root}/testdata/distribution/mount_fallback \
  registry:2.8.1 \
  $UPSTREAM_CTR_NAME \
  $ORAS_REGISTRY_FALLBACK_PORT
echo "  prepared : upstream distribution  "

echo " >>> preparing: zot >>> "
try_clean_up $ZOT_CTR_NAME
docker run --pull always -dp $ZOT_REGISTRY_PORT:5000 \
  --name $ZOT_CTR_NAME \
  -u $(id -u $(whoami)) \
  --mount type=bind,source="${e2e_root}/testdata/zot/",target=/etc/zot \
  --rm ghcr.io/project-zot/zot-linux-amd64:v2.0.1
echo " <<< prepared : zot <<< "
