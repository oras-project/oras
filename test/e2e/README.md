# ORAS E2E Testing Infrastructure

This directory contains the infrastructure for running end-to-end (e2e) tests against OCI-compliant registries running in Kubernetes.

## Overview

The e2e testing infrastructure deploys three OCI-compliant registries to a Kubernetes cluster:

1. **Docker Registry v2** - The standard OCI distribution registry
2. **Fallback Registry** - A secondary Docker Registry v2 instance for testing fallback scenarios
3. **Zot Registry** - A modern, lightweight OCI registry with additional features

## Prerequisites

- A running Kubernetes cluster (e.g., kind, minikube, Docker Desktop, or a cloud provider)
- `kubectl` configured to access your cluster
- Sufficient cluster resources (at least 2 CPU cores and 4GB RAM recommended)

If you do not have a cluster yet, create one with [kind](https://kind.sigs.k8s.io/):

```bash
kind create cluster
```

## Quick Start

### Deploy Registries

```bash
./test/e2e/scripts/deploy.sh
```

This will:
- Create the `oras-e2e-tests` namespace
- Deploy Docker Registry v2, Fallback Registry, and Zot Registry
- Set up shared storage for all registries
- Wait for all deployments to be ready

### Run E2E Tests (Recommended for CI/CD)

The easiest way to run e2e tests is as a Kubernetes Job within the cluster:

```bash
# 1. Build the test container image
./test/e2e/scripts/build-image.sh

# 2. Run tests as a Kubernetes job
./test/e2e/scripts/run-tests.sh
```

The test job will:
- Run in the same namespace as the registries
- Access registries using in-cluster service endpoints
- Stream test logs to your console
- Report pass/fail status

This approach is ideal for CI/CD pipelines as it doesn't require port forwarding.

### Check Status

```bash
./test/e2e/scripts/status.sh
```

This displays:
- Deployment status
- Pod status
- Service endpoints
- PersistentVolumeClaim status
- Health check results

### Teardown

To remove all deployed resources:

```bash
./test/e2e/scripts/teardown.sh
```

This will delete all registries, services, test jobs, and the namespace.

## Registry Endpoints

### Within the Kubernetes Cluster

The registries are accessible at these endpoints from within the cluster:

- **Docker Registry**: `docker-registry.oras-e2e-tests.svc.cluster.local:5000`
- **Fallback Registry**: `fallback-registry.oras-e2e-tests.svc.cluster.local:5000`
- **Zot Registry**: `zot-registry.oras-e2e-tests.svc.cluster.local:5000`

## Testing Examples

### Test with ORAS CLI

Run tests using the provided scripts, which will execute tests inside the cluster:

```bash
./test/e2e/scripts/run-tests.sh
```

For manual testing with ORAS CLI:

```bash
# Push an artifact
oras push localhost:5000/myrepo/myartifact:latest ./file.txt

# Pull an artifact
oras pull localhost:5000/myrepo/myartifact:latest
```

## Directory Structure

```
test/e2e/
├── README.md              # This file
├── Dockerfile             # Container image for running e2e tests
├── go.mod                 # Separate Go module holding the e2e suites
├── go.sum                 # Checksums for the e2e module's dependencies
├── go.work                # Workspace file linking this module to the root module
├── suite/                 # Ginkgo test suites
│   ├── auth/                  # Login and credential specs
│   ├── command/               # ORAS CLI command specs
│   └── scenario/              # Multi-command scenario specs
├── internal/              # Helpers shared by the suites
│   ├── testdata/              # Go fixture packages (artifact, feature, foobar) used by specs
│   └── utils/                 # Command execution, output matching, suite setup
├── k8s/                   # Kubernetes manifests
│   ├── namespace.yaml          # Namespace definition
│   ├── docker-registry.yaml    # Docker Registry v2 deployment
│   ├── fallback-registry.yaml  # Fallback Registry deployment
│   ├── zot-registry.yaml       # Zot Registry deployment
│   └── e2e-test-job.yaml       # Job definition for running tests
├── scripts/               # Management scripts
│   ├── deploy.sh          # Deploy all registries
│   ├── teardown.sh        # Remove all registries
│   ├── status.sh          # Check status of registries
│   ├── build-image.sh     # Build test container image
│   ├── generate-cache-db.sh # Seed the registry content cache
│   ├── run-tests.sh       # Run e2e tests as Kubernetes job
│   └── run-interactive.sh # Start a pod for running tests interactively
└── testdata/              # Artifact fixtures consumed by the specs
    ├── distribution/          # Images and indexes used by the specs
    ├── files/                 # Payloads pushed by the specs
    ├── zot/                   # Pre-seeded OCI layout and cache.db served by zot
    ├── empty.registry.config  # Config with no credentials
    └── legacy.registry.config # Config using the legacy auth layout
```

## Registry Configurations

### Docker Registry v2

- **Image**: `registry:2`
- **Port**: 5000
- **Storage**: Shared hostPath volume (`/tmp/oras-e2e-zot-data`)
- **Features**:
  - OCI Distribution Spec compliant
  - Delete operations enabled
  - Filesystem storage backend

### Fallback Registry

- **Image**: `registry:2`
- **Port**: 5000
- **Storage**: Shared hostPath volume (`/tmp/oras-e2e-zot-data`)
- **Features**:
  - OCI Distribution Spec compliant
  - Delete operations enabled
  - Used for testing registry fallback scenarios

### Zot Registry

- **Image**: `ghcr.io/project-zot/zot:v2.1.11`
- **Port**: 5000
- **Storage**: Shared hostPath volume (`/tmp/oras-e2e-zot-data`)
- **Features**:
  - OCI Distribution Spec v1.1.0 compliant
  - Search extension enabled
  - Advanced artifact management
  - Blob deduplication with hard links
  - Additional API features

## Troubleshooting

### Pods not starting

Check pod logs:
```bash
kubectl logs -n oras-e2e-tests -l app=docker-registry
kubectl logs -n oras-e2e-tests -l app=zot-registry
```

### Storage issues

Check PVC status:
```bash
kubectl get pvc -n oras-e2e-tests
kubectl describe deployment docker-registry -n oras-e2e-tests
kubectl describe deployment fallback-registry -n oras-e2e-tests
kubectl describe deployment zot-registry -n oras-e2e-tests
```

### Registry not healthy

Check the health status:
```bash
./test/e2e/scripts/status.sh
```

View detailed pod information:
```bash
kubectl describe pod -n oras-e2e-tests -l app=docker-registry
kubectl describe pod -n oras-e2e-tests -l app=fallback-registry
kubectl describe pod -n oras-e2e-tests -l app=zot-registry
```

## Running Tests as Kubernetes Jobs

### Overview

The e2e testing infrastructure supports running tests as Kubernetes Jobs, which offers several advantages:

- **No port forwarding required** - Tests access registries directly via in-cluster service endpoints
- **CI/CD friendly** - Ideal for automated testing pipelines
- **Isolated environment** - Each test run gets a fresh container
- **Resource management** - Kubernetes handles resource allocation and cleanup

### Building the Test Image

Before running tests, build the container image:

```bash
./test/e2e/scripts/build-image.sh
```

This script will:
1. Build a Docker image with all source code and dependencies
2. Automatically detect your Kubernetes environment (kind/minikube/k3d)
3. Load the image into your cluster

For remote clusters, the script provides instructions to push to a registry.

### Running Tests

Execute tests as a Kubernetes Job:

```bash
./test/e2e/scripts/run-tests.sh
```

This script will:
1. Verify registries are deployed and ready
2. Delete any previous test job
3. Create a new test job
4. Stream test logs to your console
5. Report the final test status (pass/fail)

### Viewing Test Results

To view logs from a completed test job:

```bash
kubectl logs -n oras-e2e-tests job/oras-e2e-tests
```

To view job status:

```bash
kubectl get jobs -n oras-e2e-tests
kubectl describe job oras-e2e-tests -n oras-e2e-tests
```

### Environment Variables

Tests running in the Kubernetes Job automatically receive these environment
variables (see `k8s/e2e-test-job.yaml`):

- `ORAS_REGISTRY_HOST` - Docker Registry endpoint (docker-registry.oras-e2e-tests.svc.cluster.local:5000)
- `ORAS_REGISTRY_FALLBACK_HOST` - Fallback Registry endpoint (fallback-registry.oras-e2e-tests.svc.cluster.local:5000)
- `ZOT_REGISTRY_HOST` - Zot Registry endpoint (zot-registry.oras-e2e-tests.svc.cluster.local:5000)
- `ORAS_PATH` - Path to the `oras` binary under test
- `ORAS_E2E_PLAIN_HTTP` - Set to "true" for plain HTTP communication
- `ORAS_E2E_TIMEOUT` - Test timeout (default: 10m)

The host variables, `ORAS_PATH`, and `ORAS_E2E_PLAIN_HTTP` are the keys read by
`internal/utils`; `ORAS_E2E_TIMEOUT` is consumed only by the Job's shell, which
passes it to `ginkgo --timeout`. When a host variable is unset, the suites fall
back to `localhost:5000`, `localhost:6000`, and `localhost:7000` respectively,
and build `oras` from source into a temporary binary if `ORAS_PATH` is not set.

> Note: the Job manifest also sets `DOCKER_REGISTRY_HOST` and
> `FALLBACK_REGISTRY_HOST`, which are used by the Job's registry-readiness wait
> loop and by `scripts/run-interactive.sh`. The specs do not read them.

## Writing E2E Tests

The e2e tests are [Ginkgo](https://onsi.github.io/ginkgo/) suites that exercise the
`oras` CLI as a subprocess rather than driving the library directly. Each package
under `suite/` owns a `RunSpecs` entry point and lives in the separate
`oras.land/oras/test/e2e` module:

| Suite | Entry point | Covers |
| --- | --- | --- |
| `suite/auth/` | `auth_test.go` | `oras login`, credential handling |
| `suite/command/` | `command_suite_test.go` | individual CLI commands |
| `suite/scenario/` | `scenario_test.go` | multi-command scenarios |

Shared setup lives in `internal/utils/init.go`. It resolves the registry endpoints
from the environment, builds (or locates) the `oras` binary under test, and logs
in to each registry before any spec runs. Helpers for building and asserting on
commands are in `internal/utils/exec.go`.

A spec, for example from `suite/command/push.go`, looks like this:

```go
var _ = Describe("Remote registry users:", func() {
	tag := "e2e"
	When("pushing to OCI spec v1.0 registries", func() {
		statusKeys := []match.StateKey{
			foobar.ImageConfigStateKey("application/vnd.oci.empty.v1+json"),
			foobar.FileBarStateKey,
		}
		It("should push files without customized media types", func() {
			repo := pushTestRepo("no-mediatype")
			tempDir := PrepareTempFiles()
			ref := RegistryRef(ZOTHost, repo, tag)

			ORAS("push", ref, foobar.FileBarName).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
		})
	})
})
```

Note that `ORAS(...)` builds the command, the matcher methods attach assertions,
and `Exec()` runs it. `ExpectFailure()` inverts the exit-code check,
`MatchErrKeyWords` asserts on stderr, and `RegistryRef`/`LayoutRef` construct
references. Registry endpoints are reached through the package-level `Host`,
`FallbackHost`, and `ZOTHost` variables rather than read from the environment in
each spec.

### Running a Single Suite or Spec

`CopyZOTRepo` copies straight into zot's **storage filesystem**, so the suites
need the `/zot-data` volume mounted — network access to a port-forwarded
registry is not enough. Use the interactive pod, which mounts it and sets
`ORAS_PATH`:

```bash
./test/e2e/scripts/run-interactive.sh
```

Ginkgo wants its flags *before* the package list, and `--focus` is a regex over
the full spec text. Inside the pod:

```bash
# Run everything, as CI does
ginkgo -r -p --race suite

# Run one suite
ginkgo -r -p suite/command

# Run one spec by name
ginkgo -r -p --focus "Remote registry users.*without customized media types" suite/command
```

## CI/CD Integration

### Recommended Approach (Kubernetes Jobs)

For CI/CD environments, using Kubernetes Jobs is the recommended approach:

1. Set up a Kubernetes cluster (e.g., using kind, k3s, or minikube)
2. Deploy registries
3. Build test image
4. Run tests as a Kubernetes Job
5. Clean up

Example GitHub Actions workflow:

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up kind cluster
      uses: helm/kind-action@v1.10.0

    - name: Deploy registries
      run: ./test/e2e/scripts/deploy.sh

    - name: Build test image
      run: ./test/e2e/scripts/build-image.sh

    - name: Run e2e tests
      run: ./test/e2e/scripts/run-tests.sh

    - name: Show test logs on failure
      if: failure()
      run: kubectl logs -n oras-e2e-tests job/oras-e2e-tests

    - name: Teardown
      if: always()
      run: ./test/e2e/scripts/teardown.sh
```

### GitLab CI Example

```yaml
e2e-tests:
  image: ubuntu:latest
  services:
    - docker:dind
  before_script:
    - apt-get update && apt-get install -y curl
    - curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    - install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    # Install kind or similar
  script:
    - ./test/e2e/scripts/deploy.sh
    - ./test/e2e/scripts/build-image.sh
    - ./test/e2e/scripts/run-tests.sh
  after_script:
    - ./test/e2e/scripts/teardown.sh
```

## License

Copyright The ORAS Authors. Licensed under the Apache License, Version 2.0.
