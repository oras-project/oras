#!/bin/bash
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

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA_DIR="$SCRIPT_DIR/../testdata/zot"

echo "Generating cache.db for zot testdata..."
echo ""

# Check if namespace exists, if not deploy
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Namespace does not exist. Deploying registries..."
    "$SCRIPT_DIR/deploy.sh"
else
    echo "Namespace exists. Checking if zot-registry is deployed..."
    if ! kubectl get deployment zot-registry -n oras-e2e-tests &> /dev/null; then
        echo "Deploying zot-registry..."
        kubectl apply -f "$SCRIPT_DIR/../k8s/zot-registry.yaml"
    fi
fi

echo "Waiting for zot-registry to be ready..."
kubectl wait --for=condition=available --timeout=120s deployment/zot-registry -n oras-e2e-tests

# Give zot some time to process deduplication
echo "Waiting for zot to process deduplication (10 seconds)..."
sleep 10

# Create debug pod to access the cache.db
echo "Creating debug pod to extract cache.db..."
POD_NAME="zot-cache-extractor"

# Clean up any existing pod
kubectl delete pod "$POD_NAME" -n oras-e2e-tests --ignore-not-found=true --wait=true

# Create debug pod
kubectl run "$POD_NAME" \
    --image=alpine:latest \
    --restart=Never \
    --namespace=oras-e2e-tests \
    --overrides='
{
  "spec": {
    "containers": [{
      "name": "debug",
      "image": "alpine:latest",
      "command": ["sleep", "infinity"],
      "volumeMounts": [{
        "name": "zot-data",
        "mountPath": "/zot-data",
        "readOnly": true
      }]
    }],
    "volumes": [{
      "name": "zot-data",
      "hostPath": {
        "path": "/tmp/oras-e2e-zot-data",
        "type": "DirectoryOrCreate"
      }
    }]
  }
}'

# Wait for pod to be ready
echo "Waiting for extractor pod to be ready..."
kubectl wait --for=condition=Ready --timeout=60s pod/"$POD_NAME" -n oras-e2e-tests

# Check if cache.db exists
echo "Checking if cache.db exists..."
if ! kubectl exec "$POD_NAME" -n oras-e2e-tests -- test -f /zot-data/cache.db; then
    echo "Error: cache.db not found in /zot-data/"
    echo "Listing contents of /zot-data:"
    kubectl exec "$POD_NAME" -n oras-e2e-tests -- ls -la /zot-data/
    kubectl delete pod "$POD_NAME" -n oras-e2e-tests --wait=false
    exit 1
fi

# Get cache.db file size
echo "Cache.db file info:"
kubectl exec "$POD_NAME" -n oras-e2e-tests -- ls -lh /zot-data/cache.db

# Copy cache.db from pod to local
echo "Copying cache.db to testdata directory..."
kubectl cp "oras-e2e-tests/$POD_NAME:/zot-data/cache.db" "$TESTDATA_DIR/cache.db"

# Verify the file was copied
if [ -f "$TESTDATA_DIR/cache.db" ]; then
    echo ""
    echo "Success! cache.db has been generated and saved to:"
    echo "  $TESTDATA_DIR/cache.db"
    echo ""
    ls -lh "$TESTDATA_DIR/cache.db"
else
    echo "Error: Failed to copy cache.db"
    exit 1
fi

# Clean up debug pod
echo ""
echo "Cleaning up extractor pod..."
kubectl delete pod "$POD_NAME" -n oras-e2e-tests --wait=false

echo ""
echo "Done!"
