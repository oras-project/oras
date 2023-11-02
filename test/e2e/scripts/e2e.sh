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

# help
help () {
    echo "Usage"
    echo "  e2e.sh <repo_root> [--clean]"
    exit 1
}

# 1. Prepare
repo_root=$1
if [ -z "${repo_root}" ]; then
    echo "repository root path is not provided."
    help
fi

clean=$2
if [ "${clean}" != '--clean' ] && [ -n "${clean}" ]; then
    echo "invalid flag found: ${clean}"
    help
fi

. ${repo_root}/test/e2e/scripts/prepare.sh $1 $2

if [ "${clean}" = '--clean' ]; then
    echo " === setting deferred clean up jobs  === "
    trap "try_clean_up $ORAS_CTR_NAME $UPSTREAM_CTR_NAME $ZOT_CTR_NAME" EXIT
fi

# 2. Test
echo " === run tests === "
if ! ginkgo -r -p --succinct suite; then 
  echo " === retriving registry error logs === "
  echo '-------- oras distribution trace -------------'
  docker logs -t --tail 200 $ORAS_CTR_NAME
  echo '-------- upstream distribution trace -------------'
  docker logs -t --tail 200 $UPSTREAM_CTR_NAME
  echo '-------- zot trace -------------'
  docker logs -t --tail 200 $ZOT_CTR_NAME
  exit 1
fi
