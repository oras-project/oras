# Support attaching files to a multi-platform image

## Overview 

ORAS supported multi-platform image creation and management as an experimental feature since its v1.3.0-beta.1, as articulated in the [specification](https://github.com/oras-project/oras/pull/1514). This proposal document outlines the scenarios of attaching files to a multi-platform image and proposes a solution to address this problem with `oras` CLI.

## Problem Statement & Motivation

Related GitHub issue: https://github.com/oras-project/oras/issues/1531

Assuming there is a multi-platform image created with the [OCI image index format](https://github.com/opencontainers/image-spec/blob/main/image-index.md), the user wants to attach a file to the multi-platform image and propogate it to platform-specific images. However, [oras attach](https://oras.land/docs/commands/oras_attach) only allows the user to attach the file to the image index or to a single platform-specific image only. There are a few scenarios that the user may want to attach a file as a referrer to a multi-platform image and a part of the platform-specific images.

## Scenarios 

### Attach an End-of-Life (EoL) annotation as a referrer to a multi-platform image

A security engineer Alice wants to use annotations to store EoL metadata of an image to indicate that the image is no longer valid. There is a multi-platform image `demo/alpine:a1a1` with multiple platform-specific images. When there is a vulnerability detected in a certain platform `demo/alpine:b1b1` of the multi-platform image, this vulnerable image will be patched and a new digest will be created accordingly. The older vulnerable image will be marked as an invalid image with an EoL annotation `"vnd.demo.artifact.lifecycle.end-of-life.date": "2025-03-20T01:20:30Z"` attached. Meanwhile, as the digest of the platform-specifc image has been patched and updated with a new digest, the parent image index is supposed to be updated with a new digest as well. When the platform-specific image is marked as EoL, it will no longer be used by other dependent services, the vulnerability scanning tool can also detect that this image will not be scanned anymore by recognizing the image lifecycle metadata from the annotations. 

After the vulnerable platform-specific image `demo/alpine:b1b1` is patched, Alice wants to attach the EoL annotation to a parent image index `demo/alpine:a1a1` and the patched image, she has to manually retrieve the new digest of the patched image and the new image index, then run `oras attach` commands twice to attach the EoL annotation to the old digest of a multi-platform image and its platform-specific image manifest. If there are multiple platform-images need to be patched and updated, Alice has to run multiple commands.

```console
demo/alpine:a1a1 (image index)
-> demo/alpine:b1b1  <-- VULNERABLE
-> demo/alpine:c1c1
```

After the vulnerable image is patched, the EoL annotation is attached to the old image index and its platform-specific image respectively:

```
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date": "2025-03-20T01:20:30Z"
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.artifact.lifecycle" --annotation "vnd.demo.artifact.lifecycle.end-of-life.date": "2025-03-20T01:20:30Z" --platform linux/amd64
```

The image result is as follows:

```console
demo/alpine:a1a1 (image index) <-- VULNERABLE, EoL attached
-> demo/alpine:b1b1 (linux/amd64)  <-- VULNERABLE, EoL attached
-> demo/alpine:c1c1 (linux/arm64) 
```

A new image index will be created:

```console
demo/alpine:z1z1 (image index) 
-> demo/alpine:r1r1 (linux/amd64) 
-> demo/alpine:c1c1 (linux/arm64) 
```

### Attach a signature

A DevOps engineer Bob wants to attach the cryptographic signature as a referrer to a multi-platform image to ensure its integrity and authenticity. Each image signature is supposed to be attached to the image index and each platform-specific image in case any compromise happended. There is a multi-platform image `demo/alpine:a1a1` with multiple platform-specific images. Bob has to run the `oras attach --platform os[/arch][/variant][:os_version]` multiple times to attach the signature to the image index and each platform-specific image.

```
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.test.signature" a1a1.sig 
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.test.signature" b1b1.sig --platform linux/amd64
oras attach $registry/demo/alpine:a1a1 --artifact-type "application/vnd.demo.test.signature" c1c1.sig --platform linux/arm64
```


```console
demo/alpine:a1a1 (image index) <-- signed with an attached signature a1a1.sig
-> demo/alpine:b1b1 (linux/amd64)  <-- signed with an attached signature b1b1.sig
-> demo/alpine:c1c1 (linux/arm64)  <-- signed with an attached signature c1c1.sig
```


