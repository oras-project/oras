#!/bin/sh -e

function help {
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
function run-registry {
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

    # prepare test data
    rm -rf $mnt_root/docker
    for layer in $(ls -rt $mnt_root/*.tar.gz); do
        tar -xvzf $layer -C $mnt_root
    done

    try-clean-up $ctr_name
    echo "--------------------"
    docker run --pull always -d -p $ctr_port:5000 --rm --name $ctr_name \
    --env REGISTRY_STORAGE_DELETE_ENABLED=true \
    --env REGISTRY_AUTH_HTPASSWD_REALM=test-basic \
    --env REGISTRY_AUTH_HTPASSWD_PATH=/etc/docker/registry/passwd \
    --mount type=bind,source=$mnt_root/docker,target=/var/lib/registry/docker \
    --mount type=bind,source=$mnt_root/passwd_bcrypt,target=/etc/docker/registry/passwd \
    $img_name
    defer-clean-up $ctr_name
}

# deferred clean up
function defer-clean-up {
    trap "try-clean-up $1" EXIT
}


# clean up
function try-clean-up {
    ctr_name=$1
    if [ -z "$ctr_name" ]; then
        echo "container name for cleanning up is not provided."
        exit 1
    fi
    trap "docker kill ${ctr_name} || true"
}