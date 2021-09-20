#!/bin/bash

if [ -z "$1" ]; then
    echo 'Usage: ./discover_demo <repo>'
    exit 1
fi

export PORT=5000
export REGISTRY=localhost:${PORT}
export REPO=$1
export IMAGE=${REGISTRY}/${REPO}:v1


echo '{"version": "0.0.0.0", "artifact": "'${IMAGE}'", "contents": "good"}' > sbom.json
./oras push $REGISTRY/$REPO \
    --artifact-type sbom/example \
    --subject $IMAGE \
    sbom.json:application/json

echo '{"version": "0.0.0.0", "artifact": "'${IMAGE}'", "signature": "signed"}' > signature.json
./oras push $REGISTRY/$REPO \
    --artifact-type signature/example \
    --subject $IMAGE \
    signature.json:application/json

./oras discover -o tree --artifact-type=sbom/example $IMAGE

digest=$(./oras discover -o json --artifact-type sbom/example ${IMAGE} | jq -r '.references[0].digest')

./oras pull -a "${REGISTRY}/${REPO}@${digest}"

