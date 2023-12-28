# Formatted ORAS CLI output

## Table of Contents

- [Background](#background)
- [Guidance](#guidance)
- [Scenarios](#scenarios)
  - [Scripting](#scripting)
  - [CI/CD](#cicd)
- [Proposal and desired user experience](#proposal-and-desired-user-experience)
  - [oras pull](#oras-pull)
  - [oras push](#oras-push)
  - [oras attach](#oras-attach)
  - [oras discover](#oras-discover)
- [FAQ](#faq)

## Background

ORAS has prettified output designed for humans. However, for machine processing, especially in automation scenarios, like scripting and CI/CD pipelines, developers want to perform batch operations and chain different commands with ORAS, as well as filtering, modifying, and sorting objects based on the outputs that are emitted by the ORAS command. Developers expect that ORAS output can be emitted as machine-readable text instead of only the prettified or tabular data, so that it can be used to perform other operations.

The formatted output is not intended to supersede the prettified human-readable and friendly output text of ORAS CLI. It aims to provide a programming-friendly experience for developers who want to automate their workflows without needing to parse unstructured text.

## Guidance

Provide two major options to enable users to define the output format of ORAS commands:

- Use `--output` to output a file or directory in the filesystem
- Use `--format` to format the output of ORAS commands into different data formats or enable process output data using the given Go template
  - Use `--format json|tree|table` to print the output in prettified JSON, tree view, or table view
  - Use `--format '{{ GO_TEMPLATE_FUNCTION }}'` to enable extract or compose the output data using Go template functions 

> [!NOTE]
> - `--output -` and `--format` can not be used at the same time due to conflicts.

See sample use cases of formatted output for `oras manifest fetch`:

- If user doesn't specify `--format` flag, the default output should be raw JSON data. This is the current behavior of `oras manifest fetch`.

- Use `--output <file>` and `--format` at the same time, a manifest file should be produced in the filesystem and the `mediaType` value should be outputted on the console:

```console
$ oras manifest fetch $REGISTRY/$REPO:$TAG --output sample-manifest.json --format {{.config.MediaType}}
application/vnd.oci.empty.v1+json
```

View the contents of the generated manifest within specified `sample-manifest.json` file. The contents should be raw JSON data:

```console
$ cat sample-manifest.json
{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.unknown.artifact.v1","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:6cb759c4296e67e35b0367f3c0f51dfdb776a0c99a45f39d0476e43d82696d65","size":14477,"annotations":{"org.opencontainers.image.title":"sbom.spdx"}},{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:54c0e84503c8790e03afe34bfc05a5ce45c933430cfd9c5f8a99d2c89f1f1b69","size":6639,"annotations":{"org.opencontainers.image.title":"scan-test-verify-image.json"}}],"annotations":{"org.opencontainers.image.created":"2023-12-15T09:41:54Z"}}
```

- Use `--format json` to print the output in prettified JSON:

```bash
oras manifest fetch $REGISTRY/$REPO:$TAG --format json
```

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.unknown.artifact.v1",
  "config": {
    "mediaType": "application/vnd.oci.empty.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2,
    "data": "e30="
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:6cb759c4296e67e35b0367f3c0f51dfdb776a0c99a45f39d0476e43d82696d65",
      "size": 14477,
      "annotations": {
        "org.opencontainers.image.title": "sbom.spdx"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:54c0e84503c8790e03afe34bfc05a5ce45c933430cfd9c5f8a99d2c89f1f1b69",
      "size": 6639,
      "annotations": {
        "org.opencontainers.image.title": "scan-test-verify-image.json"
      }
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "2023-12-15T09:41:54Z"
  }
}
```

> [!NOTE]
> - The formatted output of `oras manifest fetch` only supports OCI/Docker Image Manifest in this release. [OCI image index](https://github.com/opencontainers/image-spec/blob/v1.1.0-rc5/image-index.md) and other media types of artifact will be supported in future releases.
> - In general, in the prettified JSON format, the first letter of all output fields are supposed to be upper case except for `oras manifest fetch`. It's appropriate to pretty print the raw JSON only when using the flag `--format json` with `oras manifest fetch` because the default output is raw JSON format.

## Scenarios

### Scripting

Alice is a developer who wants to batch operations with ORAS in her Shell script. In order to automate a portion of her workflow, she would like to obtain the image digest from the JSON output objects produced by the `oras push` command and then use shell variables or utilities like [xargs](https://en.wikipedia.org/wiki/Xargs) to enable an ORAS command to act on the output of another command and perform further steps. In this way, she can chain commands together. For example, she can use `oras attach` to attach an SBOM to the image using its image digest as a argument outputted from `oras push`.

For example, push an artifact to a registry and generate the artifact reference in the standard output. Then, attach an SBOM to the artifact using the artifact reference (`$REGISTRY/$REPO@$DIGEST`) outputted from the first command. Finally, sign the attached SBOM with another tool against the reference of the SBOM file (`$REGISTRY/$REPO@$DIGEST`) that was obtained in the proceeding step.

- Use shell variables on Unix

```bash
REFERENCE_A=$(oras push $REGISTRY/$REPO:$TAG hello.txt --format '{{.Ref}}')
REFERENCE_B=$(oras attach --artifact-type sbom/example $REFERENCE_A sbom.spdx --format '{{.Ref}}') 
notation sign $REFERENCE_B
```

- Use [ConvertFrom-Json](https://learn.microsoft.com/powershell/module/microsoft.powershell.utility/convertfrom-json) on Windows PowerShell

```powershell
$A=oras push $REGISTRY/$REPO:$TAG hello.txt --format json --no-tty | ConvertFrom-Json
$B=oras attach --artifact-type sbom/example $A.Ref sbom.spdx --format json --no-tty | ConvertFrom-Json
notation sign $B.Ref
```

### CI/CD

Bob is a DevOps engineer. He uses the ORAS GitHub Actions [Setup action](https://github.com/oras-project/setup-oras) to install ORAS in his CI/CD workflow. He wants to chain several ORAS commands in a Shell script to perform multiple operations.

For example, pull multiple files (layers) from a repository and filter out the file path of its first layer in the standard output. Then, pass the pulled first layer to the second command (`docker import`) to perform further operations. 

```yaml
jobs:
  example-job:
    steps:
      - uses: oras-project/setup-oras@v1
      - run: |
          PATH=`oras pull $REGISTRY/$REPO:$TAG --format '{{.first .Files.Path}}'`
          docker import $PATH
```

## Proposal and desired user experience

Enable users to use the `--format <type>` flag to format output into structured data (e.g. JSON) and use the `--format` with the [Go template](https://pkg.go.dev/text/template) language to manipulate the output fields. Users can still use `--format '{{toRawJson .}}'` to get raw JSON output.

### oras pull 

When using `oras pull` with the flag `--format`, the following fields should be formatted into JSON output:

- `Ref`: full artifact reference by digest, e.g, `$REGISTRY/$REPO@$DIGEST`
-  `Files`: a list of downloaded files
    - `Path`: the absolute file path of the pulled file (layer)
    - `Ref`: full reference by digest of the pulled file (layer)
    - `MediaType`: media type of the pulled file (layer) 
    - `Digest`: digest of the pulled file (layer) 
    - `Size`: file size in bytes
    - `Annotations`: contains arbitrary metadata for the image manifest

For example, pull an artifact that contains multiple layers (files) and show their descriptor metadata as pretty JSON in standard output:

```bash
oras pull $REGISTRY/$REPO:$TAG --artifact-type example/sbom sbom.spdx  --artifact-type example/vul-scan vul-scan.json --format json
```

```json
{
  "Ref": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186111",
  "Files": [
    {
      "Path": "/home/user/oras-install/sbom.spdx",
      "Ref": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222",
      "MediaType": "application/vnd.oci.image.manifest.v1+json",
      "Digest": "sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222",
      "Size": 820,
      "Annotations": {
        "org.opencontainers.image.title": "sbom.spdx"
      }
    },
    {
      "Path": "/home/user/oras-install/vul-scan.json",
      "Ref": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b",
      "MediaType": "application/vnd.oci.image.manifest.v1+json",
      "Digest": "sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b",
      "Size": 820,
      "Annotations": {
        "org.opencontainers.image.title": "vul-scan.json"
      }
    }
  ]
}
```

> [!NOTE]
> When pulling a folder to filesystem, the value of `Path` should be an absolute path of the folder and should be end with slash `/` or backslash `\`, for example, `/home/Bob/sample-folder/` on Unix or `C:\Users\Bob\sample-folder\` on Windows. Other fields are the same as the example of pulling files as above.

For example, pull an artifact that contains multiple layers (files) and show their descriptor metadata as raw JSON as standard output.

```bash
oras pull $REGISTRY/$REPO:$TAG --format '{{toRawJson .}}'
```

```
{"Ref":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186111","Files":[{"Path":"/home/user/oras-install/sbom.spdx","Ref":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222","MediaType":"application/vnd.oci.image.manifest.v1+json","Digest":"sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222","Size":820},{"Path":"/home/user/oras-install/vul-scan.json","Ref":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b","MediaType":"application/vnd.oci.image.manifest.v1+json","Digest":"sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b","Size":820}]}
```

### oras push

Push two files to a repository and show the descriptor of the image manifest in pretty JSON format. 

```bash
oras push $REGISTRY/$REPO:$TAG sbom.spdx vul-scan.json --format json 
```

```json
{
  "Ref": "localhost:5000/oras@sha256:4a5b8c83d153f52afdfcb422db56c2349aae3bd5ecf8338a58353b5eb6681c45",
  "MediaType": "application/vnd.oci.image.manifest.v1+json",
  "Digest": "sha256:4a5b8c83d153f52afdfcb422db56c2349aae3bd5ecf8338a58353b5eb6681c45",
  "Size": 820,
  "Annotations": {
    "org.opencontainers.image.created": "2023-12-15T09:41:54Z"
  }
}
```

> [!NOTE]
> When pushing a folder to filesystem, the output fields are the same as the example of pushing files as above.

Push a folder to a repository and filter out the value of `reference` and `artifactType` of the pushed artifact in the standard output.

```bash
oras push $REGISTRY/$REPO:$TAG sample-folder --format '{{.Ref}}, {{.MediaType}}'
```

```console
localhost:5000@sha256:85438e6598bf35057962fff34399a362d469ca30a317939427fca6b7a289e70d, application/vnd.oci.image.manifest.v1+json
```

### oras attach

Attach two files to an image and show the descriptor metadata of the referrer in JSON format.

```bash
oras attach $REGISTRY/$REPO:$TAG --artifact-type example/report-and-sbom vul-report.json:example/vul-scan sbom.spdx:example/sbom --format json
```

```json
{
  "Ref": "localhost:5000/oras@sha256:0afd0f0c35f98dcb607de0051be7ebefd942eef1e3a6d26eefd1b2d80f2affbe",
  "MediaType": "application/vnd.oci.image.manifest.v1+json",
  "Digest": "sha256:0afd0f0c35f98dcb607de0051be7ebefd942eef1e3a6d26eefd1b2d80f2affbe",
  "Size": 923,
  "Annotations": {
    "org.opencontainers.image.created": "2023-12-15T08:59:21Z"
  }
}
```

### oras discover

View an artifact's referrers. The default output should be listed in a tree view.

```bash
oras discover $REGISTRY/$REPO:$TAG --format tree
```

```console
localhost:localhost:5000/hello@sha256:5cb894d0c94c56894e160ad2eeb19a123b4d2155374e7709f43f8c0c2f249fe2
├── application/vnd.oci.empty.v1+json
│   └── sha256:0db683b656132cede5360f42bc52541f3386b30ce685e6e63ff93ced54423fb8
└── application/vnd.cncf.notary.signature
    └── sha256:476d43120d3799fa76d7706f741beca73f5ff4149c8b6db3bd516a73d4a82fc1
```

View an artifact's referrers manifest in pretty JSON output. The following fields should be outputted:

- `Manifests`: the list of referrers
  - `Ref`: full reference by digest of the referrer
  - `MediaType`: media type of the referrer
  - `Size`: referrer file size in bytes
  - `Digest`: digest of the referrer
  - `ArtifactType`: artifact type of a referrer
  - `Annotations`: contains arbitrary metadata in a referrer

See an example:

```bash
oras discover localhost:5000/hello:v1 --format json
```

```json
{
  "Manifests": [
    {
      "MediaType": "application/vnd.oci.image.manifest.v1+json",
      "Digest": "sha256:0db683b656132cede5360f42bc52541f3386b30ce685e6e63ff93ced54423fb8",
      "Size": 964,
      "Annotations": {
        "org.opencontainers.image.created": "2023-12-14T13:48:32Z"
      },
      "ArtifactType": "application/vnd.oci.empty.v1+json"
    },
    {
      "MediaType": "application/vnd.oci.image.manifest.v1+json",
      "Digest": "sha256:476d43120d3799fa76d7706f741beca73f5ff4149c8b6db3bd516a73d4a82fc1",
      "Size": 728,
      "Annotations": {
        "io.cncf.notary.x509chain.thumbprint#S256": "[\"792265ec6b22f0a87c7b3d980319d51a76a382de1b7a47bd877bb4e5a9beb637\"]",
        "org.opencontainers.image.created": "2023-12-14T14:41:56Z"
      },
      "ArtifactType": "application/vnd.cncf.notary.signature"
    }
  ]
}
```

> [!NOTE]
> The `--format` flag will replace the existing `--output` flag. The `--output` will be marked as "deprecated" in ORAS v1.2.0 and will be removed in the future releases. 

## FAQ

**Q:** Why choose to use `--format` flag to enable JSON formatted output instead of extending the existing `--output` flag?
**A:** ORAS follows [GNU](https://www.gnu.org/prep/standards/html_node/Option-Table.html#Option-Table) design principles. ORAS uses `--output` to specify a file or directory content should be created within and `--format` to format the output into JSON or using the given Go template. Popular tools, like Docker, Podman, and Skopeo also follow this design principle within their formatted output feature.

**Q:** Why ORAS chooses [Go template](https://pkg.go.dev/text/template)?
**A:** Go template is a powerful method to customize output you want It allows users to manipulate the output format of certain commands. It provides access to data objects and additional functions that are passed into the template engine programmatically. It also has some useful libraries that have strong functions for Go’s template language to manipulate the output data, such as [Sprig](https://masterminds.github.io/sprig/).

**Q:** What is the difference of prettified JSON and raw JSON?

In the context of ORAS output, raw JSON means display the output of ORAS command in the string format, while prettified JSON means display the output of ORAS command in a pretty format.