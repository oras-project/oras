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

echo "Starting ORAS e2e test pod interactively in Kubernetes..."
echo ""

# Check if namespace exists
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Error: Namespace 'oras-e2e-tests' does not exist."
    echo "Please deploy the registries first:"
    echo "  ./test/e2e/scripts/deploy.sh"
    exit 1
fi

# Check if registries are ready
echo "Checking if registries are ready..."
if ! kubectl wait --for=condition=available --timeout=10s \
    deployment/docker-registry \
    deployment/fallback-registry \
    deployment/zot-registry \
    -n oras-e2e-tests &> /dev/null; then
    echo "Warning: Some registries may not be ready."
    echo "Check status with: ./test/e2e/scripts/status.sh"
    echo ""
fi

# Clean up any existing interactive pod
POD_NAME="oras-e2e-interactive"
if kubectl get pod "$POD_NAME" -n oras-e2e-tests &> /dev/null; then
    echo "Cleaning up existing interactive pod..."
    kubectl delete pod "$POD_NAME" -n oras-e2e-tests --wait=true
fi

echo "Creating interactive test pod..."
echo ""

# Create the interactive pod with volume mount
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
  namespace: oras-e2e-tests
spec:
  restartPolicy: Never
  containers:
  - name: interactive
    image: oras-e2e-tests:latest
    imagePullPolicy: IfNotPresent
    command: ["/bin/sh", "-c", "sleep infinity"]
    env:
    - name: CGO_ENABLED
      value: "1"
    - name: DOCKER_REGISTRY_HOST
      value: "docker-registry.oras-e2e-tests.svc.cluster.local:5000"
    - name: FALLBACK_REGISTRY_HOST
      value: "fallback-registry.oras-e2e-tests.svc.cluster.local:5000"
    - name: ZOT_REGISTRY_HOST
      value: "zot-registry.oras-e2e-tests.svc.cluster.local:5000"
    - name: ORAS_REGISTRY_HOST
      value: "docker-registry.oras-e2e-tests.svc.cluster.local:5000"
    - name: ORAS_REGISTRY_FALLBACK_HOST
      value: "fallback-registry.oras-e2e-tests.svc.cluster.local:5000"
    - name: ORAS_PATH
      value: "/workspace/bin/linux/amd64/oras"
    - name: PATH
      value: "/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/go/bin:/workspace/bin/linux/amd64"
    - name: ORAS_E2E_PLAIN_HTTP
      value: "true"
    - name: ORAS_E2E_TIMEOUT
      value: "10m"
    volumeMounts:
    - name: zot-data
      mountPath: /zot-data
  volumes:
  - name: zot-data
    hostPath:
      path: /tmp/oras-e2e-zot-data
      type: Directory
EOF

# Wait for pod to be ready
echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready --timeout=60s pod/"$POD_NAME" -n oras-e2e-tests

echo ""
echo "=================================================="
echo "Interactive pod is ready!"
echo ""
echo "Environment variables set:"
echo "  CGO_ENABLED=1"
echo "  ORAS_REGISTRY_HOST=docker-registry.oras-e2e-tests.svc.cluster.local:5000"
echo "  ORAS_REGISTRY_FALLBACK_HOST=fallback-registry.oras-e2e-tests.svc.cluster.local:5000"
echo "  ZOT_REGISTRY_HOST=zot-registry.oras-e2e-tests.svc.cluster.local:5000"
echo "  ORAS_E2E_PLAIN_HTTP=true"
echo ""
echo "Volume mounts:"
echo "  /zot-data -> zot registry storage (read-write)"
echo ""
echo "Working directory: /workspace/test/e2e"
echo ""
echo "To build oras and run tests automatically:"
echo "  /entrypoint.sh"
echo ""
echo "To build and run:"
echo "  cd /workspace && make >out 2>&1 || cat out"
echo "  cd /workspace/test/e2e"
echo "  ginkgo -r suite | tee out"
echo ""
echo "To run a specific test:"
echo "  ginkgo --focus=\"test pattern\" suite/command"
echo ""
echo "To inspect zot registry data:"
echo "  ls -la /zot-data"
echo "  find /zot-data -type f"
echo ""
echo "To exit the shell, type 'exit' or press Ctrl+D"
echo ""
echo "The pod will be automatically deleted when you exit."
echo "=================================================="
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up interactive pod..."
    kubectl delete pod "$POD_NAME" -n oras-e2e-tests --wait=false &> /dev/null || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Connect to the pod interactively
kubectl exec -it "$POD_NAME" -n oras-e2e-tests -- /bin/sh
