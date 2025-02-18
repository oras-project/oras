# Formatted ORAS CLI output

## Table of Contents

- [Formatted ORAS CLI output](#formatted-oras-cli-output)
  - [Table of Contents](#table-of-contents)
  - [Background](#background)
  - [Guidance](#guidance)
  - [Feature status](#feature-status)
  - [Scenarios](#scenarios)
    - [Scripting](#scripting)
    - [CI/CD](#cicd)
  - [Proposal and desired user experience](#proposal-and-desired-user-experience)
    - [oras manifest fetch](#oras-manifest-fetch)
    - [oras pull](#oras-pull)
    - [oras push](#oras-push)
    - [oras attach](#oras-attach)
    - [oras discover](#oras-discover)
  - [FAQ](#faq)

## Background

ORAS has prettified output designed for humans. However, for machine processing, especially in automation scenarios, like scripting and CI/CD pipelines, developers want to perform batch operations and chain different commands with ORAS, as well as filtering, modifying, and sorting objects based on the outputs that are emitted by the ORAS command. Developers expect that ORAS output can be emitted as machine-readable text instead of only the prettified or tabular data, so that it can be used to perform other operations.

The formatted output is not intended to supersede the prettified human-readable and friendly output text of ORAS CLI. It aims to provide multiple views and a programming-friendly experience for developers, DevOps engineers, IT professionals who want to automate their workflows without needing to parse unstructured text.

## Guidance

ORAS allows to use `--output` to output a file or directory in the filesystem. To enable users to format the metadata output of ORAS commands and compute figures based on the formatted output, these two options are proposed as follows:

- Use `--format <DATA_FORMAT>` to format metadata output of ORAS commands into different formats including prettified JSON, tree, table view, and Go template, i.e. `--format json|tree|table|go-template=GO_TEMPLATE`. It supports computing figures within the template using [Sprig](http://masterminds.github.io/sprig/) functions. This is the primary and recommended usage.
- Use `--template GO_TEMPLATE` to compute and manipulate the output data using Go template based on the chosen data format. To avoid ambiguity, this flag can only be used along with `--format go-template`.

## Feature status

Both `--format` and `--template` are marked as "Experimental" in its first iteration since it is still in development and is available in a part of ORAS CLI commands. In future versions, `--format` could be upgraded to "Preview" but `--template` might remain "Experimental" or even being removed in case other types of templates emerged.

## Scenarios

### Scripting

Alice is a developer who wants to batch operations with ORAS in her Shell script. In order to automate a portion of her workflow, she would like to obtain the image digest from the JSON output objects produced by the `oras push` command and then use shell variables or utilities to enable an ORAS command to act on the output of another command and perform further steps. In this way, she can chain commands together. For example, she can use `oras attach` to attach an SBOM to the image using its image digest as a argument outputted from `oras push`. Briefly, the detailed steps are as follows:  

Firstly, push an artifact to a registry and generate the artifact reference in the standard output. Then, attach an SBOM to the artifact using the artifact reference (`$REGISTRY/$REPO@$DIGEST`) outputted from the first command. Finally, sign the attached SBOM with another tool against the reference of the SBOM file (`$REGISTRY/$REPO@$DIGEST`) that was obtained in the proceeding step.

- Use shell variables on Unix

```bash
REFERENCE_A=$(oras push $REGISTRY/$REPO:$TAG hello.txt --format go-template='{{.reference}}')
REFERENCE_B=$(oras attach --artifact-type sbom/example $REFERENCE_A sbom.spdx --format go-template='{{.reference}}') 
notation sign $REFERENCE_B
```

- Use [ConvertFrom-Json](https://learn.microsoft.com/powershell/module/microsoft.powershell.utility/convertfrom-json) on Windows PowerShell

```powershell
$A=oras push $REGISTRY/$REPO:$TAG hello.txt --format json --no-tty | ConvertFrom-Json
$B=oras attach --artifact-type sbom/example $A.reference sbom.spdx --format json --no-tty | ConvertFrom-Json
notation sign $B.reference
```

### CI/CD

Bob is a DevOps engineer. He uses the ORAS GitHub Actions [Setup action](https://github.com/oras-project/setup-oras) to install ORAS in his CI/CD workflow. He wants to chain several ORAS commands in a Shell script to perform multiple operations.

For example, pull multiple files (layers) from a repository and filter out the file path of its first layer in the standard output. Then, pass the pulled first layer to the second command (e.g. `cat`) to perform further operations.

```yaml
jobs:
  example-job:
    steps:
      - uses: oras-project/setup-oras@v1
      - run: |
          PATH=`oras pull $REGISTRY/$REPO:$TAG --format go-template='{{(first .files).path}}'`
          cat $PATH
```

### Verify local files

Carol is a build engineer who needs to check whether some locally copied files are identical to the remote artifact content. she can utilize `oras manifest fetch` to generate a checksum file based on the manifest without pulling the whole artifact. (See related issue [here](https://github.com/oras-project/oras/issues/1368)).

```bash
oras manifest fetch $REGISTRY/$REPO:$TAG --format go-template='{{range .content.layers}}{{if index .annotations "org.opencontainers.image.title"}}{{(split ":" .digest)._1}}  {{index .annotations "org.opencontainers.image.title"}}{{println}}{{end}}{{end}}' | shasum -c
```

## Proposal and desired user experience

Enable users to use the `--format` flag to format metadata output into structured data (e.g. JSON) and optionally use the `--template` with the [Go template](https://pkg.go.dev/text/template) to manipulate the output data.

### oras manifest fetch

For example, when using `oras manifest fetch` with the flag `--format`, the following fields should be formatted into JSON output:

- `reference`: full artifact reference by digest, e.g, `$REGISTRY/$REPO@$DIGEST`
- `mediaType`: media type of the image manifest
- `digest`: digest of the image manifest
- `size`: manifest file size in bytes
- `artifactType`: the type of an artifact when the manifest is used for an artifact
- `content`: content object includes prettified manifest output
  - `config`: a content descriptor describes the disposition of the targeted content
  - `layers`:  array of objects, each object in the array MUST be a descriptor

See sample use cases of formatted output for `oras manifest fetch`:

- Example: use `--output <file>` and `--format` at the same time, a manifest file should be produced in the filesystem and the `mediaType` value should be outputted on the console:

```console
$ oras manifest fetch $REGISTRY/$REPO:$TAG --output sample-manifest --format go-template='{{.content.config.mediaType}}'
application/vnd.oci.empty.v1+json
```

View the content of the generated manifest within specified `sample-manifest` file. The output should be compact JSON data:

```console
$ cat sample-manifest
{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.unknown.artifact.v1","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:6cb759c4296e67e35b0367f3c0f51dfdb776a0c99a45f39d0476e43d82696d65","size":14477,"annotations":{"org.opencontainers.image.title":"sbom.spdx"}},{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:54c0e84503c8790e03afe34bfc05a5ce45c933430cfd9c5f8a99d2c89f1f1b69","size":6639,"annotations":{"org.opencontainers.image.title":"scan-test-verify-image.json"}}],"annotations":{"org.opencontainers.image.created":"2023-12-15T09:41:54Z"}}
```

> [!NOTE]
>
> - `--output -` and `--format` can not be used at the same time due to conflicts.

- Example: use `--format json` to print the metadata output in prettified JSON:

```bash
oras manifest fetch $REGISTRY/$REPO:$TAG --format json
```

```json
{
  "reference": " $REGISTRY/$REPO@sha256:8be4c36a29979c72fdd225654498791fb381a7dd8332ade1981274a16220fe1c",
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "digest": "sha256:8be4c36a29979c72fdd225654498791fb381a7dd8332ade1981274a16220fe1c",
  "artifactType": "application/vnd.unknown.artifact.v1",
  "content": {
    "config": {
      "mediaType": "application/vnd.oci.empty.v1+json",
      "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
      "size": 2
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
}
```

- Example: use `--format go-template` along with `--template GO_TEMPLATE` to fetch the metadata output and render it with Go template, filter out the `config` data of the manifest:

```bash
oras manifest fetch $REGISTRY/$REPO:$TAG --format go-template --template '{{ toPrettyJson .content.config }}'
```

```json
{
  "digest": "sha256:b6f50765242581c887ff1acc2511fa2d885c52d8fb3ac8c4bba131fd86567f2e",
  "mediaType": "application/vnd.docker.container.image.v1+json",
  "size": 3362
}
```

### oras pull

When using `oras pull` with the flag `--format`, the following fields should be formatted into JSON output:

- `reference`: full artifact reference by digest, e.g, `$REGISTRY/$REPO@$DIGEST`
- `files`: a list of downloaded files
  - `path`: the absolute file path of the pulled file (layer)
  - `reference`: full reference by digest of the pulled file (layer)
  - `mediaType`: media type of the pulled file (layer)
  - `digest`: digest of the pulled file (layer)
  - `size`: file size in bytes
  - `annotations`: contains arbitrary metadata for the image manifest

For example, pull an artifact that contains multiple layers (files) and show their descriptor metadata as pretty JSON in standard output:

```bash
oras pull $REGISTRY/$REPO:$TAG --artifact-type example/sbom sbom.spdx  --artifact-type example/vul-scan vul-scan.json --format json
```

```json
{
  "reference": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186111",
  "files": [
    {
      "path": "/home/user/oras-install/sbom.spdx",
      "reference": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222",
      "size": 820,
      "annotations": {
        "org.opencontainers.image.title": "sbom.spdx"
      }
    },
    {
      "path": "/home/user/oras-install/vul-scan.json",
      "reference": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b",
      "size": 820,
      "annotations": {
        "org.opencontainers.image.title": "vul-scan.json"
      }
    }
  ]
}
```

> [!NOTE]
> When pulling a folder to filesystem, the value of `path` should be an absolute path of the folder and should be ended with slash `/` or backslash `\`, for example, `/home/Bob/sample-folder/` on Unix or `C:\Users\Bob\sample-folder\` on Windows. Other fields are the same as the example of pulling files as above.

For example, pull an artifact that contains multiple layers (files) and show their descriptor metadata as compact JSON in the standard output.

```bash
oras pull $REGISTRY/$REPO:$TAG --format go-template='{{toRawJson .}}'
```

```
{"reference":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186111","files":[{"path":"/home/user/oras-install/sbom.spdx","reference":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222","mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222","size":820},{"path":"/home/user/oras-install/vul-scan.json","reference":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b","mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b","size":820}]}
```

### oras push

When using `oras push` with the flag `--format`, the following fields should be formatted into JSON output:

- `reference`: full artifact reference by digest, e.g, `$REGISTRY/$REPO@$DIGEST`
- `referenceByTags`: array, pushed tags by reference, e.g. `$REGISTRY/$REPO@TAG1`
- `mediaType`: media type of the pushed file (layer)
- `digest`: digest of the pushed file (layer)
- `size`: file size in bytes
- `artifactType`: artifact type of the pushed file
- `annotations`: contains arbitrary metadata for the image manifest

For example, push a file and two tags to a repository and show the descriptor of the image manifest in pretty JSON format.

```bash
oras push $REGISTRY/$REPO:$TAG1,$TAG2 sbom.spdx vul-scan.json --format json 
```

```json
{
  "reference": "$REGISTRY/$REPO@sha256:4a5b8c83d153f52afdfcb422db56c2349aae3bd5ecf8338a58353b5eb6681c45",
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "digest": "sha256:4a5b8c83d153f52afdfcb422db56c2349aae3bd5ecf8338a58353b5eb6681c45",
  "size": 820,
  "annotations": {
    "org.opencontainers.image.created": "2023-12-15T09:41:54Z"
  },
  "artifactType": "json/example",
  "referenceByTags": [
    "$REGISTRY/$REPO:$TAG1",
    "$REGISTRY/$REPO:$TAG2"
  ]
}
```

> [!NOTE]
> When pushing a folder to filesystem, the output fields are the same as the example of pushing files as above.

Push a folder to a repository and filter out the value of `reference` and `mediaType` of the pushed artifact in the standard output.

```bash
oras push $REGISTRY/$REPO:$TAG sample-folder --format go-template='{{.reference}}, {{.mediaType}}'
```

```console
$REGISTRY/$REPO@sha256:85438e6598bf35057962fff34399a362d469ca30a317939427fca6b7a289e70d, application/vnd.oci.image.manifest.v1+json
```

### oras attach

When using `oras attach` with the flag `--format`, the following fields should be formatted into JSON output:

- `reference`: full reference by digest of the referrer file
- `mediaType`: media type of the referrer
- `size`: referrer file size in bytes
- `digest`: digest of the attached referrer file
- `artifactType`: artifact type of the referrer
- `annotations`: contains arbitrary metadata in a referrer

For example, attach two files to an image and show the descriptor metadata of the referrer in JSON format.

```bash
oras attach $REGISTRY/$REPO:$TAG --artifact-type example/report-and-sbom vul-report.json:example/vul-scan sbom.spdx:example/sbom --format json
```

```json
{
  "reference": "$REGISTRY/$REPO@sha256:0afd0f0c35f98dcb607de0051be7ebefd942eef1e3a6d26eefd1b2d80f2affbe",
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "digest": "sha256:0afd0f0c35f98dcb607de0051be7ebefd942eef1e3a6d26eefd1b2d80f2affbe",
  "size": 923,
  "annotations": {
    "org.opencontainers.image.created": "2023-12-15T08:59:21Z"
  },
  "artifactType": "example/report-and-sbom"
}
```

### oras discover

View an artifact's referrers. The default output should be listed in a tree view.

```bash
oras discover $REGISTRY/$REPO:$TAG --format tree
```

```console
$REGISTRY/$REPO@sha256:a3785f78ab8547ae2710c89e627783cfa7ee7824d3468cae6835c9f4eae23ff7
├── application/vnd.cncf.notary.signature
│   └── sha256:8dee8cb9a1334595545e3baf15c3eeed13c4b35ae08e3ab32e1df31fb152dc1d
└── sbom/example
    └── sha256:50fd0dc107d84b5e7b402688000a7ed3aaf8a2692d5cb74da5277fa3c4cecf15
```

View an artifact's referrers manifest in pretty JSON output. The following fields should be outputted:

- `manifests`: the list of referrers' manifest
  - `reference`: full reference by digest of the referrer
  - `mediaType`: media type of the referrer
  - `digest`: digest of the referrer
  - `size`: referrer file size in bytes
  - `annotations`: contains arbitrary metadata in a referrer
  - `artifactType`: artifact type of a referrer

See an example:

```bash
oras discover $REGISTRY/$REPO:v1 --format json
```

```json
{
  "manifests": [
    {
      "reference": "$REGISTRY/$REPO@sha256:8dee8cb9a1334595545e3baf15c3eeed13c4b35ae08e3ab32e1df31fb152dc1d",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:8dee8cb9a1334595545e3baf15c3eeed13c4b35ae08e3ab32e1df31fb152dc1d",
      "size": 739,
      "annotations": {
        "io.cncf.notary.x509chain.thumbprint#S256": "[\"79e91aa1e109a16df87d200e493fd3d33c67253f76d41334d7f7c29c00ba55b3\"]",
        "org.opencontainers.image.created": "2024-01-01T10:32:55Z"
      },
      "artifactType": "application/vnd.cncf.notary.signature"
    },
    {
      "reference": "$REGISTRY/$REPO@sha256:50fd0dc107d84b5e7b402688000a7ed3aaf8a2692d5cb74da5277fa3c4cecf15",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:50fd0dc107d84b5e7b402688000a7ed3aaf8a2692d5cb74da5277fa3c4cecf15",
      "size": 739,
      "annotations": {
        "org.opencontainers.image.created": "2024-01-01T07:57:10Z"
      },
      "artifactType": "sbom/example"
    }
  ]
}
```

> [!NOTE]
> The `--format` flag will replace the existing `--output` flag. The `--output` will be marked as "deprecated" in ORAS v1.2.0 and will be removed in the future releases.

If the referrers have associated referrers, ORAS should be able to show the manifest content of the subject image and all referrers. 

When showing the subject image and all referrers' manifests recursively in a tree view output, the following fields should be returned:

- `reference`: full reference by digest of the subject image
- `mediaType`: media type of the subject image
- `digest`: digest of the subject image
- `size`: subject image size in bytes
- `manifests`: the list of referrers' manifest
  - `reference`: full reference by digest of the referrer
  - `mediaType`: media type of the referrer
  - `digest`: digest of the referrer
  - `size`: referrer file size in bytes
  - `artifactType`: artifact type of a referrer
  - `annotations`: contains arbitrary metadata in a referrer
  - `referrerManifests`: the list of referrers' manifest

For example, when there are two refferers lifecycle metadata and in-toto attestation associated with a sample image, the signatures are associated with these two files respectively. The output in a tree view will be:

```bash
oras discover localhost:5000/kubernetes/kubectl@sha256:bece4f4746a39cb39e38451c70fa5a1e5ea4fa20d4cca40136b51d9557918b01
```

```console
localhost:5000/kubernetes/kubectl@sha256:bece4f4746a39cb39e38451c70fa5a1e5ea4fa20d4cca40136b51d9557918b01
├── application/vnd.oci.artifact.lifecycle
│   └── sha256:325129be79f416fe11a9ec44233cfa57f5d89434e6d37170f97e48f7904983e3
│       └── application/vnd.cncf.notary.signature
│           └── sha256:f520330e9f43c05859c532e67a25c9c765b144782ae7b872656192c27fd4e2dd
└── application/vnd.in-toto+json
    └── sha256:a811606b09341bab4bbc0a4deb2c0cb709ec9702635cbe2d36b77d58359ec046
        └── application/vnd.cncf.notary.signature
            └── sha256:04723fd7d00df77c6f226b907667396554bf9418dc48a7a04feb5ff24aa0b9ec
```

When showing the subject image and all referrers' manifests recursively in a pretty JSON output, the following JSON output should be returned:

```bash
oras discover localhost:5000/kubernetes/kubectl@sha256:bece4f4746a39cb39e38451c70fa5a1e5ea4fa20d4cca40136b51d9557918b01 --format json --recursive
```

```

  "reference": "localhost:5000/kubernetes/kubectl@sha256:bece4f4746a39cb39e38451c70fa5a1e5ea4fa20d4cca40136b51d9557918b01",
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "digest": "sha256:bece4f4746a39cb39e38451c70fa5a1e5ea4fa20d4cca40136b51d9557918b01",
  "size": 1788,
  "manifests": [
    {
      "reference": "localhost:5000/kubernetes/kubectl@sha256:325129be79f416fe11a9ec44233cfa57f5d89434e6d37170f97e48f7904983e3",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:325129be79f416fe11a9ec44233cfa57f5d89434e6d37170f97e48f7904983e3",
      "size": 788,
      "annotations": {
        "org.opencontainers.image.created": "2024-03-15T22:49:10Z",
        "vnd.oci.artifact.lifecycle.end-of-life.date": "2024-03-15"
      },
      "artifactType": "application/vnd.oci.artifact.lifecycle",
      "referrerManifests": [
        {
          "reference": "localhost:5000/kubernetes/kubectl@sha256:f520330e9f43c05859c532e67a25c9c765b144782ae7b872656192c27fd4e2dd",
          "mediaType": "application/vnd.oci.image.manifest.v1+json",
          "digest": "sha256:f520330e9f43c05859c532e67a25c9c765b144782ae7b872656192c27fd4e2dd",
          "size": 1080,
          "annotations": {
            "io.cncf.notary.x509chain.thumbprint#S256": "[430ecf0685f8018443f8418f5d7134b146f28862116114925713635d5703fb69,9b1894f223d934cbd6575af3c6e1f6096b9221a7da132185f5a5cdc92235b5dc,23ffe2b8bdb9a1711515d4cffda04bc7f793d513c76c243f1020507d8669b7db]",
            "org.opencontainers.image.created": "2024-03-15T22:54:42Z"
          },
          "artifactType": "application/vnd.cncf.notary.signature"
        }
      ]
    },
    {
      "reference": "localhost:5000/kubernetes/kubectl@sha256:a811606b09341bab4bbc0a4deb2c0cb709ec9702635cbe2d36b77d58359ec046",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:a811606b09341bab4bbc0a4deb2c0cb709ec9702635cbe2d36b77d58359ec046",
      "size": 747,
      "annotations": {
        "org.opencontainers.image.created": "2024-01-18T18:12:41Z"
      },
      "artifactType": "application/vnd.in-toto+json",
      "referrerManifests": [
        {
          "reference": "localhost:5000/kubernetes/kubectl@sha256:04723fd7d00df77c6f226b907667396554bf9418dc48a7a04feb5ff24aa0b9ec",
          "mediaType": "application/vnd.oci.image.manifest.v1+json",
          "digest": "sha256:04723fd7d00df77c6f226b907667396554bf9418dc48a7a04feb5ff24aa0b9ec",
          "size": 1080,
          "annotations": {
            "io.cncf.notary.x509chain.thumbprint#S256": "[430ecf0685f8018443f8418f5d7134b146f28862116114925713635d5703fb69,9b1894f223d934cbd6575af3c6e1f6096b9221a7da132185f5a5cdc92235b5dc,23ffe2b8bdb9a1711515d4cffda04bc7f793d513c76c243f1020507d8669b7db]",
            "org.opencontainers.image.created": "2024-01-18T18:20:09Z"
          },
          "artifactType": "application/vnd.cncf.notary.signature"
        }
      ]
    }
  ]
```


## FAQ

**Q:** Why choose to use `--format` flag to enable JSON formatted output instead of extending the existing `--output` flag?
**A:** ORAS follows [GNU](https://www.gnu.org/prep/standards/html_node/Option-Table.html#Option-Table) design principles. ORAS uses `--output` to specify a file or directory content should be created within and `--format` to format the output into JSON or using the given Go template. Popular tools, like Docker, Podman, and Skopeo also follow this design principle within their formatted output feature.

**Q:** Why ORAS chooses [Go template](https://pkg.go.dev/text/template)?
**A:** Go template is a powerful method to allow users to manipulate and customize output you want. It provides access to data objects and additional functions that are passed into the template engine programmatically. It also has some useful libraries that have strong functions for Go’s template language to manipulate the output data, such as [Sprig](https://masterminds.github.io/sprig/). The basic usage of Go template functions are easy to use.
