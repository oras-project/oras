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

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download
RUN go mod vendor

# Copy the entire source code
COPY . .

# Run stage
FROM golang:1.25-alpine

# Install runtime dependencies including build tools for CGO (required for -race flag)
RUN apk add --no-cache ca-certificates git curl wget bash gcc musl-dev make file

# Set working directory
WORKDIR /workspace

# Copy from builder
COPY --from=builder /go/pkg /go/pkg
COPY --from=builder /workspace /workspace

# Ensure module cache is properly set up
RUN go mod download
RUN go mod vendor

# Install ginkgo test runner
RUN cd test/e2e && go install github.com/onsi/ginkgo/v2/ginkgo@latest
ENV PATH="${PATH}:/go/bin"

# Set plain HTTP mode for tests (registries in k8s use HTTP not HTTPS)
ENV ORAS_E2E_PLAIN_HTTP="true"

# Create entrypoint script that builds oras and runs tests
RUN cat > /entrypoint.sh <<'EOF'
#!/bin/sh
set -e

echo "==================================="
echo "ORAS E2E Test Environment Setup"
echo "==================================="
echo ""
echo "Building oras binary..."

cd /workspace

# Build oras using make (determines correct OS and ARCH automatically)
make > out || cat out

# Construct the binary path using Makefile logic: bin/$(OS)/$(ARCH)/$(CLI_EXE)
ORAS_BIN="/workspace/bin/linux/amd64/oras"

if [ ! -f "$ORAS_BIN" ]; then
    echo "Error: oras binary not found at expected path: $ORAS_BIN"
    echo "Available binaries:"
    find /workspace/bin -name oras -type f 2>/dev/null || echo "  None found"
    exit 1
fi

echo "✓ Built oras binary: $ORAS_BIN"
echo ""

# Export ORAS_PATH for tests to use
export ORAS_PATH="$ORAS_BIN"

# Change to test directory
cd /workspace/test/e2e

# Run provided command or default to ginkgo tests
if [ $# -eq 0 ]; then
    echo "Running ginkgo tests..."
    echo "-----------------------------------"
    exec ginkgo -r -p --race suite
else
    # Execute the provided command with ORAS_PATH set
    exec "$@"
fi
EOF

RUN chmod +x /entrypoint.sh

# Set working directory
WORKDIR /workspace/test/e2e

# Use the entrypoint script
ENTRYPOINT ["/entrypoint.sh"]
CMD []
