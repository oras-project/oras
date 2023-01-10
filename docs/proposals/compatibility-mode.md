# Proposal: Provide a compatibility mode for ORAS

This document is for adding a proposal to enable compatibility mode for ORAS CLI and solve [issue #720](https://github.com/oras-project/oras/issues/720).

## Background

OCI group announced the release of v1.1 for [Image-spec](https://github.com/opencontainers/image-spec/blob/main/artifact.md) and [Distribution-spec](https://github.com/opencontainers/distribution-spec) in Sep 2022. It supports the OCI artifact manifest and provides a new referrers discovery API that allows artifact references.

Since 0.16.0, ORAS supports pushing OCI artifact manifest to OCI v1.0 compliant registries. However, the new manifest type may not be supported on the consumer side (e.g. self-crafted scripts) or in those OCI v1.0 compliant registries. To enable ORAS to work with popular registries, it provides backward compatibility which supports two types of manifest and allows fallback to upload the OCI image manifest to OCI v1.0 compliant registries or enabled OCI image manifest storage. 

## Challenge

However, the ORAS fallback may fail as there is no deterministic way to confirm if a registry supporting OCI artifact manifest and no consistent error response. On the other hand, users may want to force-push a specific manifest type to a registry for some reason.

![This table is captured from [https://toddysm.com/2023/01/05/oci-artifct-manifests-oci-referrers-api-and-their-support-across-registries-part-1/](https://toddysm.com/2023/01/05/oci-artifct-manifests-oci-referrers-api-and-their-support-across-registries-part-1/)](Proposal%20Provide%20Compatibility%20mode%20for%20ORAS%20a1e29245ea014b10a9711bae0d04db3c/Untitled.png)

This table is captured from [https://toddysm.com/2023/01/05/oci-artifct-manifests-oci-referrers-api-and-their-support-across-registries-part-1/](https://toddysm.com/2023/01/05/oci-artifct-manifests-oci-referrers-api-and-their-support-across-registries-part-1/)

The current workaround for enabling the compatibility mode is to specify a `--config` flag when using `oras push`. Since the OCI Artifact Manifest does not have a `config`, it will push an OCI image manifest instead. It is not user-friendly and is a bit hacky. It would be better if we can provide a compatibility mode for switching the manifest uploading behavior.

## Goals

- Enable ORAS to work with more registries
- Provide different options to allow users to customize the behaviors of uploading manifest to the registry
- Provide an easy-to-use and secure user experience when switching the behaviors

## Solution

Adding a new flag `--compatibility` under CLI commands `oras push` and `oras attach` with different variables to configure the behaviors of uploading manifest. We will only use `oras push` to demonstrate the examples below.

### Use case A

If users want to force-push the OCI Artifact Manifest to registries whether they are compliant with OCI v1.0 or v1.1, using `--compatibility artifact-manifest` will only upload OCI Image Manifest to registries. Users might choose it for security requirements, such as pushing a signature to a registry without changing its digest.

```bash
oras push localhost:5000/hello-artifact:v1 --artifact-type sbom/example --compatibility artifact-manifest sbom.json 
```

### Use case B

If users want to force-push the OCI Image Manifest to registries whether they are compliant with OCI v1.0 or v1.1, using `--compatibility image-manifest` will only upload OCI Image Manifest to registries. This option is helpful when users have concerns to use OCI Artifact Manifest or migrate content to OCI v1.0 compliant registries.

```bash
oras push localhost:5000/hello-artifact:v1 --artifact-type sbom/example --compatibility image-manifest sbom.json 
```

### Use case C

Disable backward compatibility and only upload OCI Artifact Manifest to a registry. This flag `--compatibility min` will be commonly used with the OCI v1.1 compliant registries or registries that support storing OCI Artifact Manifest, such as Zot, Azure Container Registry, and Docker Hub. This is the most strict option for the behavior of uploading manifest. Users might choose it for security requirements. 

```bash
oras push localhost:5000/hello-artifact:v1 --artifact-type sbom/example --compatibility min sbom.json 
```

### Use case D

If users want to upload manifest to OCI v1.0 compliant registries like Harbor, GitHub Container Registry, etc, using `--compatibility max` will upload OCI Image Manifest first and Referrals tag schema. This option enables maximum backward compatibility for registries. If it uploads fails, it will try to upload OCI Artifact Manifest. If it still uploads fails, which means ORAS is not compatible with the registry.

```bash
oras push localhost:5000/hello-artifact:v1 --artifact-type sbom/example --compatibility max sbom.json 
```