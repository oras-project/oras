#!/bin/bash -ex

# Copyright The ORAS Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

rm -rf .cover/ .test/
mkdir .cover/ .test/
trap "rm -rf .test/" EXIT

export CGO_ENABLED=0
for pkg in `go list ./...`; do
    go test -v -covermode=atomic \
        -coverprofile=".cover/$(echo $pkg | sed 's/\//_/g').cover.out" $pkg
done

echo "mode: set" > .cover/cover.out && cat .cover/*.cover.out | grep -v mode: | sort -r | \
   awk '{if($1 != last) {print $0;last=$1}}' >> .cover/cover.out

go tool cover -html=.cover/cover.out -o=.cover/coverage.html