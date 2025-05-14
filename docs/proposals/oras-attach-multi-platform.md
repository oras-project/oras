# Support Attaching Files to a Multi-Platform Image

## Overview

ORAS has supported multi-platform image creation and management as an experimental feature since v1.3.0-beta.1, as detailed in the [specification](multi-arch-image-mgmt.md). This proposal outlines the scenarios for attaching files to a multi-platform image and proposes a solution to support related scenarios using the `oras` CLI.

## Problem Statement & Motivation

Related GitHub issue: https://github.com/oras-project/oras/issues/1531

When a multi-platform image is created using the [OCI image index format](https://github.com/opencontainers/image-spec/blob/v1.1.1/image-index.md), users may want to attach a file and propagate it to platform-specific images. However, the current [oras attach](https://oras.land/docs/commands/oras_attach) command only allows users to attach a file to the image index or a single platform-specific image. Several scenarios require attaching a file as a referrer to both a multi-platform image and platform-specific images.

## Scenarios

### Attach an End-of-Life (EoL) Annotation as a Referrer to a Multi-Platform Image

- Scenario A: attach a refer to a platform-specific image to upward propagate to its parent index

A security engineer, Alice, wants to use annotations to store EoL metadata to indicate that an image is no longer valid. Consider a multi-platform image `demo/alpine:a1a1` with multiple platform-specific images. When a vulnerability is detected in a platform-specific image `demo/alpine:b1b1`, it is patched, generating a new digest. The outdated image is marked as invalid using an EoL annotation:

```json
"vnd.demo.artifact.lifecycle.end-of-life.date": "2025-03-20T01:20:30Z"
```

Since the patched image has a new digest, the parent image index also receives an updated digest. When a platform-specific image is marked as EoL, dependent services stop using it, and vulnerability scanning tools can recognize it as deprecated through the lifecycle metadata.

![multi-arch image](./img/oras-attach-EoL.drawio.svg)

After patching `demo/alpine:b1b1`, Alice has to manually retrieve the new digests and run `oras attach` twice: once for the old digest of the parent image index and once for the platform-specific image manifest. If multiple platform-specific images require updates, she has execute multiple commands.

```console
demo/alpine:a1a1 (image index)
-> demo/alpine:b1b1  <-- VULNERABLE
-> demo/alpine:c1c1
```

After patching, Alice attaches the EoL annotation to the outdated image index and platform-specific image:

```sh
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date=2025-03-20T01:20:30Z"
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date=2025-03-20T01:20:30Z" --platform linux/amd64
```

Resulting image structure:

```console
demo/alpine:a1a1 (image index) <-- VULNERABLE, EoL attached
-> demo/alpine:b1b1 (linux/amd64)  <-- VULNERABLE, EoL attached
-> demo/alpine:c1c1 (linux/arm64)
```

A new image index is created:

```console
demo/alpine:z1z1 (image index)
-> demo/alpine:r1r1 (linux/amd64)
-> demo/alpine:c1c1 (linux/arm64)
```

- Scenario B: attach a refer to an index to downward propagate to all child images

In addition, if a vulnerability is detected and affects images of all platforms, the parent index and each child image are patched, generating new digest of each. The outdated multi-platform image and each child image are marked as invalid using an EoL annotation similar as above.

```console
demo/alpine:a1a1 (image index)
-> demo/alpine:b1b1  <-- VULNERABLE
-> demo/alpine:c1c1  <-- VULNERABLE
```

Alice has to manually retrieve the new digests and run `oras attach` against each platform-specific image and the parent image index. There is no approach to attach the EoL annotation to the parent image index and propagate to all platform-specific images recursively.

```sh
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date=2025-03-20T01:20:30Z"
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date=2025-03-20T01:20:30Z" --platform linux/amd64
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date=2025-03-20T01:20:30Z" --platform linux/arm64
```


