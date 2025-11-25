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

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="${SCRIPT_DIR}/../k8s"

echo "Deploying ORAS e2e test registries to Kubernetes..."

# Create namespace
echo "Creating namespace..."
kubectl apply -f "${K8S_DIR}/namespace.yaml"

# Deploy Docker Registry
echo "Deploying Docker Registry v2..."
kubectl apply -f "${K8S_DIR}/docker-registry.yaml"

# Deploy Fallback Registry
echo "Deploying Fallback Registry..."
kubectl apply -f "${K8S_DIR}/fallback-registry.yaml"

# Deploy Zot Registry
echo "Deploying Zot Registry..."
kubectl apply -f "${K8S_DIR}/zot-registry.yaml"

# Wait for deployments to be ready
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/docker-registry \
  deployment/fallback-registry \
  deployment/zot-registry \
  -n oras-e2e-tests

echo ""
echo "Deployment complete!"
echo ""
echo "Registry endpoints (within cluster):"
echo "  Docker Registry:   docker-registry.oras-e2e-tests.svc.cluster.local:5000"
echo "  Fallback Registry: fallback-registry.oras-e2e-tests.svc.cluster.local:5000"
echo "  Zot Registry:      zot-registry.oras-e2e-tests.svc.cluster.local:5000"
echo ""
echo "Check status with:"
echo "  ./test/e2e/scripts/status.sh"
