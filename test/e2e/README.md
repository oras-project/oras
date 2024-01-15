# ORAS End-to-End Testing Dev Guide
**KNOWN LIMITATION**: E2E tests are designed to run in the CI and currently only support running on linux platform.
## Prerequisites
Install [git](https://git-scm.com/download/linux), [docker](https://docs.docker.com/desktop/install/linux-install), [go](https://go.dev/doc/install).

## Run E2E Script
```shell
# in root folder of oras CLI code
make teste2e
```

If the tests fails with errors like `ginkgo: not found`, use below command to add GOPATH into the PATH variable
```shell
PATH+=:$(go env GOPATH)/bin
```

## Development
### 1. Prepare E2E Test Environment
You may use the prepare script to setup an E2E test environment before developing E2E tests:
```shell
REPO_ROOT=$(git rev-parse --show-toplevel) # REPO_ROOT is root folder of oras CLI code
$REPO_ROOT/test/e2e/scripts/prepare.sh $REPO_ROOT
```

### 2. Using IDE
Since E2E test suites are added as an nested module, the module file and checksum file are separated from oras CLI. To develop E2E tests, it's better to set the working directory to `$REPO_ROOT/test/e2e/` or open your IDE at it.

### 3. Testing pre-built ORAS Binary
By default, Gomega builds a temp binary every time before running e2e tests, which makes sure that latest code changes in the working directory are covered. If you are making changes to E2E test code only, set `ORAS_PATH` towards your pre-built ORAS binary to skip building and speed up the test.

### 4. Debugging via `go test`
E2E specs can be ran natively without `ginkgo`:
```shell
# run below command in the target test suite folder
go test oras.land/oras/test/e2e/suite/${suite_name}
```
This is super handy when you want to do step-by-step debugging from command-line or via an IDE. If you need to debug certain specs, use [focused specs](https://onsi.github.io/ginkgo/#focused-specs) but don't check it in.

### 5. Testing Registry Services
The backend of E2E tests are three registry services: 
- [oras-distribution](https://github.com/oras-project/distribution): registry service supports artifact and image types in [image-spec 1.1.0 rc2](https://github.com/opencontainers/image-spec/tree/v1.1.0-rc2) and referrer API. Will be deprecated when [image-spec 1.1.0 rc2](https://github.com/opencontainers/image-spec/tree/v1.1.0-rc2) is not supported by oras CLI.
- [upstream distribution](https://github.com/distribution/distribution): registry service supports image media type with subject and provide referrers via [tag schema](https://github.com/opencontainers/distribution-spec/blob/v1.1.0-rc1/spec.md#referrers-tag-schema). 
- [zot](https://github.com/project-zot/zot): registry service supports artifact and image types in [image-spec 1.1.0 rc4](https://github.com/opencontainers/image-spec/tree/v1.1.0-rc4) and referrer API

You can run scenario test suite against your own registry services via setting `ORAS_REGISTRY_HOST`, `ORAS_REGISTRY_FALLBACK_HOST` and `ZOT_REGISTRY_HOST` environmental variables.

### 6. Constant Build & Watch
This is a good choice if you want to debug certain re-runnable specs:
```shell
cd $REPO_ROOT/test/e2e
ginkgo watch -r
```

### 7. Trouble-shooting CLI
The executed commands should be shown in the ginkgo logs after `[It]`, with full execution output in the E2E log.

### 8. Adding New Tests
Three suites will be maintained for E2E testing:
- command: contains test specs for single oras command execution
- auth: contains test specs similar to command specs but specific to auth. It cannot be ran in parallel with command suite specs
- scenario: contains featured scenarios with several oras commands execution

Inside a suite, please follow below model when building the hierarchical collections of specs:
```
Describe: <Role>
  When: Scenario or command specific description
    It: <Action>
      By: <Result> (per-command execution)
       Expect: <Result> (detailed checks for execution results)
```

### 9. Adding New Test Data

#### 9.1 Command Suite
Command suite uses two kinds of pre-baked test data:
- Layered distribution archive files: test data compressed from registry runtime storage directly and stored in `$REPO_ROOT/test/e2e/testdata/distribution/`. ORAS distribution uses sub-folder `mount` and upstream distribution uses sub-folder `mount_fallback`. For both registries, the repository name should follow the convention of `command/$repo_suffix`. To add a new layer to the test data, use the below command to compress the `docker` folder from the root directory of the registry storage and copy it to the corresponding subfolder in `$REPO_ROOT/test/e2e/testdata/distribution/mount`.
    ```shell
    tar -cvzf ${repo_suffix}.tar.gz --owner=0 --group=0 docker/
    ```
- OCI layout files: test data stored in `$REPO_ROOT/test/e2e/testdata/zot/` and used by ZOT registry service. You may use stable release of ORAS CLI to build it. When adding new artifacts in, please make sure the repository folder is excluded in `$REPO_ROOT/.gitignore`.


##### Test Data for ORAS-Distribution
```mermaid
graph TD;
subgraph "repository: command/images"
    subgraph "file: images.tar.gz"
        direction TB
        A0>tag: multi]-..->A1[oci index]
        A1--linux/amd64-->A2[oci image]
        A1--linux/arm64-->A3[oci image]
        A1--linux/arm/v7-->A4[oci image]
        A2-->A5(config1)
        A3-->A6(config2)
        A4-->A7(config3)
        A2-- hello.tar -->A8(blob)
        A3-- hello.tar -->A8(blob)
        A4-- hello.tar -->A8(blob)

        B0>tag: foobar]-..->B1[oci image]
        B1-- foo1 -->B2(blob1)
        B1-- foo2 -->B2(blob1)
        B1-- bar -->B3(blob2)
    end
end
```

```mermaid
graph TD;
subgraph "repository: command/artifacts"
    subgraph "file: artifacts.tar.gz"
        direction TB
        C0>tag: foobar]-..->C1[oci image]
        
        direction TB
        E1["test.sbom.file(artifact)"] -- subject --> C1
        E2["test.signature.file(artifact)"] -- subject --> E1
        direction TB
        D1["test/sbom.file(image)"] -- subject --> C1
        D2["test/signature.file(image)"] -- subject --> D1
    end
    subgraph "file: artifacts_index.tar.gz"
        direction TB
        F0>tag: multi]-..->F1[oci index]
        F1--linux/amd64-->F2[oci image]
        F1--linux/arm64-->F3[oci image]
        F1--linux/arm/v7-->F4[oci image]
        G1["referrer.index(image)"] -- subject --> F1
        G2["referrer.image(image)"] -- subject --> F2
    end
end
```

##### Test Data for Upstream Distribution
```mermaid
graph TD;
    subgraph "repository: command/artifacts"
        subgraph "file: artifacts_fallback.tar.gz"
            direction TB
            A0>tag: foobar]-..->A1[oci image]
            A1-- foo1 -->A2(blob1)
            A1-- foo2 -->A2(blob1)
            A1-- bar -->A3(blob2)

            E1["test/sbom.file(image)"] -- subject --> A1
            E2["test/signature.file(image)"] -- subject --> E1
        end
    end
```

##### Test Data for ZOT
```mermaid
graph TD;
    subgraph "repository: command/images"
        direction TB
        A0>tag: multi]-..->A1[oci index]
        A1--linux/amd64-->A2[oci image]
        A1--linux/arm64-->A3[oci image]
        A1--linux/arm/v7-->A4[oci image]
        A2-->A5(config1)
        A3-->A6(config2)
        A4-->A7(config3)
        A2-- hello.tar -->A8(blob)
        A3-- hello.tar -->A8(blob)
        A4-- hello.tar -->A8(blob)

        B0>tag: foobar]-..->B1[oci image]
        B1-- foo1 -->B2(blob1)
        B1-- foo2 -->B2(blob1)
        B1-- bar -->B3(blob2)
    end
```

```mermaid
graph TD;
    subgraph "repository: command/artifacts"
        direction TB
        C0>tag: foobar]-..->C1[oci image]
        
        direction TB
        direction TB
        D1["test.sbom.file(image)"] -- subject --> C1
        D2["test.signature.file(image)"] -- subject --> D1
        direction TB
        F0>tag: multi]-..->F1[oci index]
        F1--linux/amd64-->F2[oci image]
        F1--linux/arm64-->F3[oci image]
        F1--linux/arm/v7-->F4[oci image]
        G1["referrer.index(image)"] -- subject --> F1
        G2["referrer.image(image)"] -- subject --> F2
        G3["index"] -- subject --> F1
        
        H0>tag: unnamed]-..->H1["artifact contains unnamed layer"]
        I0>tag: empty]-..->I1["artifact contains only one empty layer"]
    end
```

#### 9.2 Scenario Suite
Test files used by scenario-based specs are placed in `$REPO_ROOT/test/e2e/testdata/files`.