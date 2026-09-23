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

echo "Tearing down ORAS e2e test resources from Kubernetes..."

# Delete e2e test job
echo "Deleting e2e test job..."
kubectl delete -f "${K8S_DIR}/e2e-test-job.yaml" --ignore-not-found=true

# Delete Zot Registry
echo "Deleting Zot Registry..."
kubectl delete -f "${K8S_DIR}/zot-registry.yaml" --ignore-not-found=true

# Delete Fallback Registry
echo "Deleting Fallback Registry..."
kubectl delete -f "${K8S_DIR}/fallback-registry.yaml" --ignore-not-found=true

# Delete Docker Registry
echo "Deleting Docker Registry..."
kubectl delete -f "${K8S_DIR}/docker-registry.yaml" --ignore-not-found=true

# Delete namespace (this will also delete any remaining resources)
echo "Deleting namespace..."
kubectl delete -f "${K8S_DIR}/namespace.yaml" --ignore-not-found=true

echo ""
echo "Teardown complete!"
