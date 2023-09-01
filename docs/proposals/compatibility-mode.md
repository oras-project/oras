# Compatibility mode for ORAS

OCI group announced the [Image-spec v1.1.0-rc4](https://github.com/opencontainers/image-spec/blob/v1.1.0-rc4/manifest.md) and [Distribution-spec v1.1.0-rc3](https://github.com/opencontainers/distribution-spec/releases/tag/v1.1.0-rc3) in July 2023. A notable breaking change is that the OCI Artifact Manifest no longer exists in the OCI Image-spec v1.1.0-rc4. 

Two experimental flags `--image-spec` and `--distribution-spec` were introduced to commands `oras push` and `oras attach` in ORAS CLI v1.0.0 as explained in [this doc](https://github.com/oras-project/oras/blob/v1.0.1/docs/proposals/compatibility-mode.md). To align with the OCI Image-spec v1.1.0-rc4, the flag `--image-spec` and its options are changed in ORAS v1.1.0 accordingly.

This document elaborates on the major changes of ORAS CLI v1.1.0 proposed in [issue #1043](https://github.com/oras-project/oras/issues/1043).

## Goals

- Provide different options to allow users to customize the manifest build and distribution behavior
- Provide an easy-to-use and secure user experience when push content to OCI registries
- Enable ORAS to work with more OCI registries

## Solution

Using the following flags in `oras push` and `oras attach` respectively with different variables to configure the manifest build and distribution behaviors. 

- Using a flag `--image-spec` with `oras push`
- Using a flag `--distribution-spec` with `oras attach`, `oras cp`, and `oras manifest push` to configure compatibility with registry when pushing or copying an OCI image manifest. This flag is also applicable to `oras discover` for viewing and filtering the referrers.

### Build and push OCI image manifest using a flag `--image-spec`

Use the flag `--image-spec <spec version>` in `oras push` to specify which version of the OCI Image-spec when building and pushing an OCI image manifest. It supports specifying the option `v1.0` or `v1.1` as the spec version. The option `v1.1` is the default behavior in `oras push` since ORAS CLI v1.1.0 so users don't need to manually specify this option.

If users want to build an OCI image manifest to a registry that compliant with OCI Spec v1.0, they can specify `--image-spec v1.0`. An OCI image manifest that conforms the OCI Image-spec v1.0.2 will be packed and uploaded. For example

```bash
oras push localhost:5000/hello-world:v1 \
  --image-spec v1.0 \
  --artifact-type application/vnd.me.config \
  sbom.json
```

### Configure compatibility with OCI registry using a flag `--distribution-spec`

Based on the Referrers API status in the registry, users can use flag `--distribution-spec <spec version>-<api>-<option>` to configure compatibility with registry. 

| registry support                        |  v1.1-referrers-api | v1.1-referrers-tag |
| :-------------------------------------- | --- | --- | 
| OCI spec 1.0                            | no  | yes |
| OCI spec 1.1 without referrers API      | no  | yes |
| OCI spec 1.1 with referrers API support | yes | yes |

Using a flag `--distribution-spec v1.1-referrers-api` to disable backward compatibility. It only allows uploading OCI image manifest to OCI v1.1 compliant registry with Referrers API enabled. This is the most strict option for setting compatibility with the registry. Users might choose it for security requirements. 

For example, using this flag, ORAS will attach OCI image manifest only to an OCI v1.1 compliant registry with Referrers API enabled and no further actions for maintaining references in OCI registries.  

```bash
oras attach localhost:5000/hello-world:v1 \
  --artifact-type sbom/example \
  --distribution-spec v1.1-referrers-api \
  sbom.json 
```

Using `--distribution-spec v1.1-referrers-tag` to enable maximum backward compatibility with the registry. It will first attempt to upload the OCI image manifest with the [referrers tag schema](https://github.com/opencontainers/distribution-spec/blob/v1.1.0-rc3/spec.md#referrers-tag-schema) regardless of whether the registry complies with the OCI Spec v1.0 or v1.1 or supports Referrers API. For example: 

```bash
oras attach localhost:5000/hello-world:v1 \
  --artifact-type sbom/example \
  --distribution-spec v1.1-referrers-tag \
  sbom.json 
```

Similarly, users can use `oras cp`, and `oras manifest push` with the flag `--distribution-spec` to configure compatibility with registry when pushing or copying an OCI image manifest, or use `oras discover` with the flag `--distribution-spec` for filtering the referrers in the view.