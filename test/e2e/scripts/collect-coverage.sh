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

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../.."

# KinD node names are Docker container names. Resolve the node from the Job
# instead of assuming the default cluster name or a single-node cluster.
POD_INFO=$(kubectl get pods -n oras-e2e-tests -l job-name=oras-e2e-tests \
    --sort-by=.metadata.creationTimestamp \
    -o jsonpath='{.items[-1].spec.nodeName}{" "}{.items[-1].metadata.uid}{" "}{.items[-1].status.phase}')
read -r NODE_NAME POD_UID POD_PHASE <<< "$POD_INFO"

case "$POD_PHASE" in
    Succeeded|Failed) ;;
    *)
        echo "Error: The e2e test Pod must have terminated before collecting coverage." >&2
        exit 1
        ;;
esac

COVERAGE_DIR=$(mktemp -d)
trap 'rm -rf "$COVERAGE_DIR"' EXIT

# The Job container has terminated, but its KinD node is still running.
# Stream from the node: docker cp cannot read KinD's /tmp tmpfs mount.
docker exec "$NODE_NAME" tar -C "/tmp/oras-e2e-coverage/${POD_UID}" -cf - . \
    | tar -C "$COVERAGE_DIR" -xf -

cd "$PROJECT_ROOT"
go tool covdata textfmt -i="$COVERAGE_DIR" -o="$COVERAGE_DIR/coverage.txt"
go tool cover -func="$COVERAGE_DIR/coverage.txt" | tee "$COVERAGE_DIR/functions.txt"

# covdata can succeed with no counters. Fail rather than publish an empty or
# zero-coverage artifact when instrumentation or collection is broken.
if ! awk '$1 ~ /^oras.land\/oras\/cmd\/oras\// && $NF + 0 > 0 { covered = 1 }
    END { exit !covered }' "$COVERAGE_DIR/functions.txt"; then
    echo "Error: No non-zero cmd/oras coverage was collected." >&2
    exit 1
fi

mv "$COVERAGE_DIR/coverage.txt" test/e2e/coverage.txt
echo "Coverage report saved to test/e2e/coverage.txt"
