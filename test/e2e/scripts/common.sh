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

help () {
    echo "Usage"
    echo "  run-registry <mount-root> <image-name> <container-name> <container-port>"
    echo ""
    echo "Arguments"
    echo "  mount-root       root mounting directory for pre-baked registry storage files."
    echo "  image-name       image name of the registry."
    echo "  container-name   container name of the registry service."
    echo "  container-port   port to export the registry service."
}

# run registry service for testing
run_registry () {
    # check arguments
    mnt_root=$1
    if [ -z "$mnt_root" ]; then
        echo "mount root directory path is not provided."
        help
        exit 1
    fi
    mnt_root=$(realpath --canonicalize-existing ${mnt_root})
    img_name=$2
    if [ -z "$img_name" ]; then
        echo "distribution image name is not provided."
        help
        exit 1
    fi
    ctr_name=$3
    if [ -z "$ctr_name" ]; then
        echo "container name is not provided."
        help
        exit 1
    fi
    ctr_port=$4
    if [ -z "$ctr_port" ]; then
        echo "distribution port is not provided."
        help
        exit 1
    fi

    rm -rf $mnt_root/docker
    for layer in $(ls -rt $mnt_root/*.tar.gz); do
        tar -xvzf $layer -C $mnt_root
    done

    try_clean_up $ctr_name
    docker run --pull always -d -p $ctr_port:5000 --rm --name $ctr_name \
        -u $(id -u $(whoami)) \
        --env REGISTRY_STORAGE_DELETE_ENABLED=true \
        --env REGISTRY_AUTH_HTPASSWD_REALM=test-basic \
        --env REGISTRY_AUTH_HTPASSWD_PATH=/etc/docker/registry/passwd \
        --mount type=bind,source=$mnt_root/docker,target=/var/lib/registry/docker \
        --mount type=bind,source=$mnt_root/passwd_bcrypt,target=/etc/docker/registry/passwd \
        $img_name
}


# clean up
try_clean_up () {
    echo " === stopping below containers ==="
    for ctr_name in "$@"
    do
        docker kill ${ctr_name} || true
    done
}