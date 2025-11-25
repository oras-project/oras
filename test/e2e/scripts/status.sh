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

echo "ORAS e2e Test Registries Status"
echo "================================"
echo ""

# Check if namespace exists
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Namespace 'oras-e2e-tests' does not exist."
    echo "Run './test/e2e/scripts/deploy.sh' to deploy the registries."
    exit 0
fi

echo "Deployments:"
echo "------------"
kubectl get deployments -n oras-e2e-tests

echo ""
echo "Pods:"
echo "-----"
kubectl get pods -n oras-e2e-tests

echo ""
echo "Services:"
echo "---------"
kubectl get services -n oras-e2e-tests

echo ""
echo "PersistentVolumeClaims:"
echo "-----------------------"
kubectl get pvc -n oras-e2e-tests

echo ""
echo "Registry Health Checks:"
echo "-----------------------"

# Helper function to check pod readiness
check_pod_health() {
    local app_label=$1
    local registry_name=$2

    local pod_status=$(kubectl get pods -n oras-e2e-tests -l app=$app_label -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)

    echo -n "$registry_name"
    if [ "$pod_status" = "True" ]; then
        echo "✓ Healthy"
    elif [ -z "$pod_status" ]; then
        echo "✗ Not Found"
    else
        echo "✗ Unhealthy"
    fi
}

# Check all registries using pod readiness status
check_pod_health "docker-registry" "Docker Registry v2:   "
check_pod_health "fallback-registry" "Fallback Registry:    "
check_pod_health "zot-registry" "Zot Registry:         "
