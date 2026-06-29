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
PROJECT_ROOT="${SCRIPT_DIR}/../../.."

# Default image name and tag
IMAGE_NAME="${IMAGE_NAME:-oras-e2e-tests}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
PLATFORM="${PLATFORM:-$(uname -m)}"

# Normalize platform names
case "$PLATFORM" in
    x86_64|amd64)
        PLATFORM="linux/amd64"
        ;;
    aarch64|arm64)
        PLATFORM="linux/arm64"
        ;;
    linux/*)
        # Already in correct format
        ;;
    *)
        echo "Warning: Unknown platform '$PLATFORM', using as-is"
        ;;
esac

# Determine build context
BUILD_CONTEXT="${PROJECT_ROOT}"
DOCKERFILE="${SCRIPT_DIR}/../Dockerfile"

echo "Building ORAS e2e test image..."
echo "  Image:      ${FULL_IMAGE}"
echo "  Platform:   ${PLATFORM}"
echo "  Dockerfile: ${DOCKERFILE}"
echo "  Context:    ${BUILD_CONTEXT}"
echo ""

# Build the image
docker build \
  --platform "${PLATFORM}" \
  -f "${DOCKERFILE}" \
  -t "${FULL_IMAGE}" \
  "${BUILD_CONTEXT}"

echo ""
echo "✓ Image built successfully: ${FULL_IMAGE}"
echo ""

# Detect Kubernetes environment and load image if needed
if command -v kind &> /dev/null; then
    # Check if kind cluster exists
    if kind get clusters 2>/dev/null | grep -q .; then
        echo "Detected kind cluster(s). Loading image into kind..."
        # Use the first available cluster or specified cluster name
        CLUSTER_NAME="${KIND_CLUSTER_NAME:-$(kind get clusters 2>/dev/null | head -1)}"
        kind load docker-image "${FULL_IMAGE}" --name "${CLUSTER_NAME}"
        echo "✓ Image loaded into kind cluster: ${CLUSTER_NAME}"
    fi
elif command -v minikube &> /dev/null; then
    # Check if minikube is running
    if minikube status &> /dev/null; then
        echo "Detected minikube. Loading image into minikube..."
        minikube image load "${FULL_IMAGE}"
        echo "✓ Image loaded into minikube"
    fi
elif command -v k3d &> /dev/null; then
    # Check if k3d cluster exists
    if k3d cluster list 2>/dev/null | grep -q .; then
        echo "Detected k3d cluster(s). Loading image into k3d..."
        k3d image import "${FULL_IMAGE}"
        echo "✓ Image loaded into k3d cluster"
    fi
else
    echo "Note: Local Kubernetes environment not detected (kind/minikube/k3d)."
    echo "If using a remote cluster, you may need to push the image to a registry:"
    echo "  docker tag ${FULL_IMAGE} <your-registry>/${IMAGE_NAME}:${IMAGE_TAG}"
    echo "  docker push <your-registry>/${IMAGE_NAME}:${IMAGE_TAG}"
    echo ""
    echo "Then update the image in test/e2e/k8s/e2e-test-job.yaml"
fi

echo ""
echo "Next steps:"
echo "  1. Deploy registries (if not already deployed):"
echo "     ./test/e2e/scripts/deploy.sh"
echo ""
echo "  2. Run e2e tests as a Kubernetes job:"
echo "     ./test/e2e/scripts/run-tests.sh"
echo ""
echo "To build for a different architecture:"
echo "  PLATFORM=linux/amd64 ./test/e2e/scripts/build-image.sh"
echo "  PLATFORM=linux/arm64 ./test/e2e/scripts/build-image.sh"
