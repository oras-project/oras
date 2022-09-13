# ORAS End-to-End Testing Dev Guide

## Setting up
Minimal setup: Run the script in **step 3**

### 1. Common dev setup for ORAS CLI
https://hackmd.io/_nRHGW8WRfOOvngWc6u-sQ

### 2. [Optional] Install Ginkgo
This will enable you use`ginkgo` directly in CLI.
```
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```
If you skip step 2, you can only run tests via `go test`. 

### 3. Run distribution
The backend of E2E test is a [oras-distribution](https://github.com/oras-project/distribution).
```bash
REPO_ROOT=$(pwd) # Set REPO_ROOT as root folder of oras CLI
PORT=5000
docker run -dp $PORT:5000 --rm --name oras-e2e \
    --mount type=bind,source=$REPO_ROOT/test/e2e/testdata/distribution/config-example-with-extensions.yml,target=/etc/docker/registry/config.yml \
    --mount type=bind,source=$REPO_ROOT/test/e2e/testdata/distribution/passwd_bcrypt,target=/etc/docker/registry/passwd \
    ghcr.io/oras-project/registry:latest
```
If the image cannot be pulled, try create a Github PAT and docker/oras login.

### 4. [Optional] Setup backend
```bash
export ORAS_REGISTRY_HOST="localhost:$PORT" # replace with right os/arch
# for PowerShell, use $env:ORAS_REGISTRY_HOST = "localhost:$PORT"
```
If you skip step 4, E2E test will look for distribution ran in `localhost:5000`

### 5. [Optional] Setup ORAS binary for testing
The ORAS CLI binary will be the frontend of E2E test.
```bash
# Set REPO_ROOT as root folder of oras CLI
cd $REPO_ROOT
make build
```
### 6. [Optional] Setup frontend
You need to setup **both** environmental variables as below to debug a pre-build ORAS binary
```bash
export ORAS_PATH="bin/linux/amd64/oras" # change target platform if needed
export GITHUB_WORKSPACE=$REPO_ROOT
```
If you skip step 5 or 6, Gomega will build a temp binary, which will include all the CLI code changes in the working directory.

## Debugging
### 1. Constant Build & Watch
This is a good choice if you want to debug certain re-runnable specs
```bash=
cd $REPO_ROOT/test/e2e
ginkgo watch -r
```

### 2. Debug certain action
Use [focused spec](https://onsi.github.io/ginkgo/#focused-specs)


### 3. Trouble-shooting CLI
Executed command should be shown in the ginkgo logs after `[It]`,