# OCI Registry As Storage

[![GitHub Actions status](https://github.com/oras-project/oras/workflows/build/badge.svg)](https://github.com/oras-project/oras/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/oras-project/oras)](https://goreportcard.com/report/github.com/oras-project/oras)
[![GoDoc](https://godoc.org/github.com/oras-project/oras?status.svg)](https://godoc.org/github.com/oras-project/oras)

![ORAS](https://github.com/oras-project/oras-www/raw/main/docs/assets/images/oras.png)

[Registries are evolving as Cloud Native Artifact Stores](https://stevelasker.blog/2019/01/25/cloud-native-artifact-stores-evolve-from-container-registries/). To enable this goal, Microsoft has donated ORAS as a means to enable various client libraries with a way to push [OCI Artifacts][artifacts] to [OCI Conformant](https://github.com/opencontainers/oci-conformance) registries.

ORAS is both a [CLI](#oras-cli) for initial testing and a [Go Module](#oras-go-module) to be included with your CLI, enabling a native experience: `myclient push artifacts.azurecr.io/myartifact:1.0 ./mything.thang`

## Table of Contents

- [ORAS Background](#oras-background)
- [Supported Registries](https://oras.land/implementors/#registries-supporting-artifacts)
- [Artifacts Implementing ORAS](https://oras.land/implementors/#artifact-types-using-oras)
- [Getting Started](#getting-started)
- [ORAS CLI](#oras-cli)
- [ORAS Go Module](#oras-go-module)
- [Contributing](#contributing)
- [Maintainers](./OWNERS)

## ORAS Background

- [OCI Image Support Comes to Open Source Docker Registry](https://opencontainers.org/posts/blog/2018-10-11-oci-image-support-comes-to-open-source-docker-registry/)
- [Registries Are Evolving as Cloud Native Artifact Stores](https://stevelasker.blog/2019/01/25/cloud-native-artifact-stores-evolve-from-container-registries/)
- [OCI Adopts Artifacts Project](https://opencontainers.org/posts/blog/2019-09-10-new-oci-artifacts-project/)
- [GitHub: OCI Artifacts Project](https://github.com/opencontainers/artifacts)

## Getting Started

[Select from one the registries that support OCI Artifacts](https://oras.land/implementors/). Each registry identifies how they support authentication.

## ORAS CLI

ORAS is both a [CLI](#oras-cli) for initial testing and a [Go Module](#oras-go-module) to be included with your CLI, enabling a native experience: `myclient push artifacts.azurecr.io/myartifact:1.0 ./mything.thang`

### CLI Installation

- Install `oras` using [GoFish](https://gofi.sh/):

  ```sh
  gofish install oras
  ==> Installing oras...
  🐠  oras 0.12.0: installed in 65.131245ms
  ```

- Install from the latest [release artifacts](https://github.com/oras-project/oras/releases):

  - Linux

    ```sh
    curl -LO https://github.com/oras-project/oras/releases/download/v0.12.0/oras_0.12.0_linux_amd64.tar.gz
    mkdir -p oras-install/
    tar -zxf oras_0.12.0_*.tar.gz -C oras-install/
    mv oras-install/oras /usr/local/bin/
    rm -rf oras_0.12.0_*.tar.gz oras-install/
    ```

  - macOS

    ```sh
    curl -LO https://github.com/oras-project/oras/releases/download/v0.12.0/oras_0.12.0_darwin_amd64.tar.gz
    mkdir -p oras-install/
    tar -zxf oras_0.12.0_*.tar.gz -C oras-install/
    mv oras-install/oras /usr/local/bin/
    rm -rf oras_0.12.0_*.tar.gz oras-install/
    ```

  - Windows

    Add `%USERPROFILE%\bin\` to your `PATH` environment variable so that `oras.exe` can be found.

    ```sh
    curl.exe -sLO  https://github.com/oras-project/oras/releases/download/v0.12.0/oras_0.12.0_windows_amd64.tar.gz
    tar.exe -xvzf oras_0.12.0_windows_amd64.tar.gz
    mkdir -p %USERPROFILE%\bin\
    copy oras.exe %USERPROFILE%\bin\
    set PATH=%USERPROFILE%\bin\;%PATH%
    ```

  - Docker Image

    A public Docker image containing the CLI is available on [GitHub Container Registry](https://github.com/orgs/oras-project/packages/container/package/oras):

    ```sh
    docker run -it --rm -v $(pwd):/workspace ghcr.io/oras-project/oras:v0.12.0 help
    ```

    > Note: the default WORKDIR  in the image is `/workspace`.

### ORAS Authentication

Run `oras login` in advance for any private registries. By default, this will store credentials in `~/.docker/config.json` *(same file used by the docker client)*. If you have previously authenticated to a registry using `docker login`, the credentials will be reused.

Use the `-c`/`--config` option to specify an alternate location.

> While ORAS leverages the local docker client config store, ORAS does NOT have a dependency on Docker Desktop running or being installed. ORAS can be used independently of a local docker daemon.

`oras` also accepts explicit credentials via options, for example,

```sh
oras pull -u username -p password myregistry.io/myimage:latest
```

See [Supported Registries](https://oras.land/implementors/) for registry specific authentication usage.

### Pushing Artifacts with Single Files

Pushing single files involves referencing the unique artifact type and at least one file.
Defining an Artifact uses the `config.mediaType` as the unique artifact type. If a config object is provided, the `mediaType` extension defines the config filetype. If a `null` config is passed, the config extension must be removed.

See: [Defining a Unique Artifact Type](https://github.com/opencontainers/artifacts/blob/master/artifact-authors.md#defining-a-unique-artifact-type)

The following sample defines a new Artifact Type of **Acme Rocket**, using `application/vnd.acme.rocket.config` as the `manifest.config.mediaType`.

- Create a sample file to push/pull as an artifact

  ```sh
  echo "hello world" > artifact.txt
  ```

- Push the sample file to the registry:

  ```sh
  oras push localhost:5000/hello-artifact:v1 \
  --manifest-config /dev/null:application/vnd.acme.rocket.config \
  ./artifact.txt
  ```

- Pull the file from the registry:

  ```sh
  rm -f artifact.txt # first delete the file
  oras pull localhost:5000/hello-artifact:v1
  cat artifact.txt  # should print "hello world"
  ```

- Push the sample file, with a layer `mediaType`, using the format `filename[:type]`:

  ```sh
  oras push localhost:5000/hello-artifact:v2 \
  --manifest-config /dev/null:application/vnd.acme.rocket.config \
    artifact.txt:text/plain
  ```

### Pushing Artifacts with Config Files

The [OCI distribution-spec][distribution-spec] provides for storing optional config objects. These can be used by the artifact to determine how or where to process and/or route the blobs. When providing a config object, the version and file type is required.

- Create a config file

  ```sh
  echo "{\"name\":\"foo\",\"value\":\"bar\"}" > config.json
  ```

- Push an the artifact, with the `config.json` file

  ```sh
  oras push localhost:5000/hello-artifact:v2 \
  --manifest-config config.json:application/vnd.acme.rocket.config.v1+json \
    artifact.txt:text/plain
  ```

### Pushing Artifacts with Multiple Files

Just as container images support multiple "layers" represented as blobs, ORAS supports pushing multiple layers. The layer type is up to the artifact author. You may push `.tar` representing a collection of files, individual files like `.yaml`, `.txt` or whatever your artifact should be represented as. Each layer type should have a `mediaType` representing the type of blob content.
In this example, we'll push a collection of files.

- A single file (`artifact.txt`) that represents overview content that might be displayed as a repository overview
- A collection of files (`docs/*`) that represents detailed content. When specifying a directory, ORAS will automatically tar the contents.

See [OCI Artifacts][artifacts] for more details.

- Create additional blobs

  ```sh
  mkdir docs
  echo "Docs on this artifact" > ./docs/readme.md
  echo "More content for this artifact" > ./docs/readme2.md
  ```

- Create a config file, referencing the entry doc file

  ```sh
  echo "{\"doc\":\"readme.md\"}" > config.json
  ```

- Push multiple files with different `mediaTypes`:

  ```sh
  oras push localhost:5000/hello-artifact:v2 \
    --manifest-config config.json:application/vnd.acme.rocket.config.v1+json \
    artifact.txt:text/plain \
    ./docs/:application/vnd.acme.rocket.docs.layer.v1+tar
  ```

- The push would generate the following manifest:

  ```json
  {
    "schemaVersion": 2,
    "config": {
      "mediaType": "application/vnd.acme.rocket.config.v1+json",
      "digest": "sha256:7aa5d0dee9a3a73c81db4356cf7aa5666e175d96e68ee763eeb977bd7ba59ee5",
      "size": 20
    },
    "layers": [
      {
        "mediaType": "text/plain",
        "digest": "sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447",
        "size": 12,
        "annotations": {
          "org.opencontainers.image.title": "artifact.txt"
        }
      },
      {
        "mediaType": "application/vnd.acme.rocket.docs.layer.v1+tar",
        "digest": "sha256:20ae7d51e2365405e6942439140d897548e1d4610db60354aef8a5ce1f1699a7",
        "size": 196,
        "annotations": {
          "io.deis.oras.content.digest": "sha256:4329ea6c620ca4e9cedc5f5e8040432114cb5d64fc53107ea870db149e3d2b9e",
          "io.deis.oras.content.unpack": "true",
          "org.opencontainers.image.title": "docs"
        }
      }
    ]
  }
  ```

### Pulling Artifacts

Pulling artifacts involves specifying the content addressable artifact, along with the type of artifact.
> See: [Issue 130](https://github.com/oras-project/oras/issues/130) for eliminating `-a` and `--media-type`

```sh
oras pull localhost:5000/hello-artifact:v2 -a
```

### Using cache when pulling artifacts

In order to save unnecessary network bandwidth and disk I/O oras should provides a solution to pull the artifacts into a local content-address storage (CAS) if the content does not exist, and then copy the artifact to the desired storage.

The cache directory is specified by using the environment variable `ORAS_CACHE`. If not specified, cache is not used.

```sh
# Set cache root
export ORAS_CACHE=~/.oras/cache

# Pull artifacts as usual
oras pull localhost:5000/hello:latest
```

## ORAS Go Module

See https://github.com/oras-project/oras-go

## Contributing

Want to reach the ORAS community and developers?
We're very interested in feedback and contributions for other artifacts.

[Join us](https://slack.cncf.io/) at [CNCF Slack](https://cloud-native.slack.com) under the **#oras** channel

[artifacts]:            https://github.com/opencontainers/artifacts
[distribution-spec]:    https://github.com/opencontainers/distribution-spec/

## Code of Conduct

This project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for further details.

