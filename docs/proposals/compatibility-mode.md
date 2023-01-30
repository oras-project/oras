# Compatibility mode for ORAS

This document is for adding a document to elaborate the compatibility mode for ORAS CLI which was proposed in [issue #720](https://github.com/oras-project/oras/issues/720).

## Background

OCI group announced the release of v1.1 for [Image-spec](https://github.com/opencontainers/image-spec/blob/v1.1.0-rc1/artifact.md) and [Distribution-spec](https://github.com/opencontainers/distribution-spec) in Sep 2022. It supports the OCI artifact manifest and provides a new referrers discovery API that allows artifact references.

Since 0.16.0, ORAS supports pushing OCI artifact manifest to OCI v1.0 compliant registries. However, the new manifest type may not be supported on the consumer side (e.g. self-crafted scripts) or in those OCI v1.0 compliant registries. To enable ORAS to work with popular registries, it provides backward compatibility which supports two types of manifest and allows fallback to upload the OCI image manifest to OCI v1.0 compliant registries or those which enabled OCI image manifest storage. 

## Challenge

The fallback attempted by ORAS may fail since there is no deterministic way to confirm if a registry supporting OCI artifact manifest due to no consistent error response. 

You can find the testing result of the implementation result for OCI Spec from this [blog](https://toddysm.com/2023/01/05/oci-artifct-manifests-oci-referrers-api-and-their-support-across-registries-part-1/). On the other hand, users may want to force-push a specific manifest type to a registry for some reason.

The current workaround for enabling a kind of compatibility mode is to specify a `--config` flag when using `oras push`. Since the OCI artifact manifest does not have a `config`, it will push an OCI image manifest instead. It is not user-friendly and is a bit hacky. It would be better if we can provide a compatibility mode to easily customize and switch the manifest uploading behavior, and enable users to handle the incompatibility problem when using ORAS across different registries. 

## Goals

- Provide different options to allow users to customize the manifest uploading behavior to the registry
- Provide an easy-to-use and secure user experience when switching the behaviors
- Enable ORAS to work with more registries flexibly 

## Solution

Adding new flags to `oras push` and `oras attach` respectively with different variables to configure the manifest uploading behaviors. 

- Adding a flag `--image-spec` to `oras push` and `oras attach` to force uploading a specific manifest type to registry
- Adding a flag `--distribution-spec` to `oras attach`, `oras attach`, `oras cp`, and `oras manifest push` to configure compatibility with registry when pushing or copying an image/artifact manifest. This flag is also applicable to `oras discover` for filtering the referrers.

### Force uploading a specific manifest type using a flag `--image-spec`

It follows `--image-spec <spec version>-<manifest type>` to enable configuration of using which spec version and manifest type. Currently, it only supports specifying v1.1 as the spec version. 

| registry support                        | v1.1-artifact | v1.1-image | 
| :-------------------------------------- | ----------------- | -------------- | 
| OCI spec 1.0                            | no                | yes            |
| OCI spec 1.1 without referrers API      | yes               | yes            | 
| OCI spec 1.1 with referrers API support | yes               | yes            | 

If users want to force pushing a specific version of OCI artifact manifest to a registry, they can use `--image-spec v1.1-artifact`. An OCI artifact manifest will be packed and uploaded. Users might choose it for security requirements, such as pushing a signature to a registry without changing its digest. For example:

```bash
oras push localhost:5000/hello-artifact:v1 \
--image-spec v1.1-artifact \
--artifact-type sbom/example \
  sbom.json 
```

If users want to force pushing an OCI image manifest, no matter whether the registry is compliant with the OCI Spec v1.0 or v1.1, using `--image-spec v1.1-image` will only upload the OCI image manifest to a registry. This option is useful when users have concerns to use OCI artifact manifest or need to migrate content to OCI v1.0 compliant registry. For example:

```bash
oras push localhost:5000/hello-artifact:v1 \
--image-spec v1.1-image \
--artifact-type sbom/example \
  sbom.json
```

### Configure compatibility with OCI registry using a flag `--distribution-spec`

Based on the Referrers API status in the registry, users can use flag `--distribution-spec <spec version>-<api>-<option>` to configure compatibility with registry. 

| registry support                        |  v1.1-referrers-api | v1.1-referrers-tag |
| :-------------------------------------- | --- | --- | 
| OCI spec 1.0                            | no  | yes |
| OCI spec 1.1 without referrers API      | no  | yes |
| OCI spec 1.1 with referrers API support | yes | yes |

Using a flag `--distribution-spec v1.1-referrers-api` to disable backward compatibility. It only allows uploading OCI artifact manifest to OCI v1.1 compliant registry with Referrers API enabled. This is the most strict option for setting compatibility with the registry. Users might choose it for security requirements. 

For example, using this flag, ORAS will attach OCI artifact manifest only to an OCI v1.1 compliant registry with Referrers API enabled and no further actions for maintaining artifact references in registries.  

```bash
oras attach localhost:5000/hello-artifact:v1 \
--artifact-type sbom/example \
--distribution-spec v1.1-referrers-api \
  sbom.json 
```

Using `--distribution-spec v1.1-referrers-tag` to enable maximum backward compatibility with the registry. It will upload OCI image manifest and [referrers tag schema](https://github.com/opencontainers/distribution-spec/blob/v1.1.0-rc1/spec.md#referrers-tag-schema) regardless of whether the registry complies with the OCI Spec v1.0 or v1.1 or support Referrers API. For example: 

```bash
oras attach localhost:5000/hello-artifact:v1 \
--artifact-type sbom/example \
--distribution-spec v1.1-referrers-tag \
  sbom.json 
```

Similarly, users can use `oras cp`, and `oras manifest push` with the flag `--distribution-spec` to configure compatibility with registry when pushing or copying an image/artifact manifest, or use `oras discover` with the flag `--distribution-spec` for filtering the referrers in the view.