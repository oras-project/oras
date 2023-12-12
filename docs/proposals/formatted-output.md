# Formatted ORAS CLI output 

ORAS has prettified output designed for humans. However, for machine processing, especially in some automation scenarios like scripting and CI/CD pipelines, developers want to perform batch operations and chain different commands with ORAS, as well as filtering, modifying, and sorting objects based on the outputs that are emitted by the ORAS command. Developers expect the ORAS output can be generated as machine-readable text not only the prettified or tabular data, so that they can use the formatted outputs like JSON output to perform further advanced operations. 

The formatted output is not intended to supersede the prettified human-readable and friendly output text of ORAS CLI. It aims to provide programming-friendly experience for developers who want to automate their workflows on machines especially on Unix and most Unix-like operating systems, without parsing unstructured text. It will increase the developer experience for ORAS in automation and scripting scenarios. 

## Scenarios

### Scripting

Alice is a developer who wants to batch operations with ORAS in her Shell script. In order to automate some routine workflow in containers secure supply chain scenario, she wants the machine to get the image digest from the JSON output objects that are emitted by `oras push`, then use utility like [xargs](https://en.wikipedia.org/wiki/Xargs) or use environmental variables to enable an ORAS command to act on the output of another command and perform further steps. In this way, she can chain commands together. For example, she can use `oras attach` to attach an SBOM to the image using its image digest as a argument outputted from `oras push`.

For example, push an artifact to a registry and generate the artifact reference in the standard output, then attach an SBOM to the artifact using the artifact reference (`$REGISTRY/$REPO@$DIGEST`) outputted from the first command, finally sign the attached SBOM with another tool against the SBOM file's reference (`$REGISTRY/$REPO@$DIGEST`) outputted from the last step.

- Use xargs utility on Unix

```bash
oras push $REGISTRY/$REPO:$TAG hello.txt --format '{{.Ref}}' |\
xargs -I _ oras attach --artifact-type sbom/example _ sbom.spdx --format '{{.Ref}}' |\
xargs -I _ notation sign _ 
```

- Use environmental variables on Unix

```bash
REFERENCE_A=$(oras push $REGISTRY/$REPO:$TAG hello.txt --format '{{.Ref}}')
REFERENCE_B=$(oras attach --artifact-type sbom/example $REFERENCE_A sbom.spdx --format '{{.Ref}}') 
notation sign $REFERENCE_B
```

- Use [ConverFrom-Json](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/convertfrom-json) on Windows PowerShell

```powershell
$REFERENCE_A=oras push $REGISTRY/$REPO:$TAG hello.txt --format json --no-tty | ConvertFrom-Json
$REFERENCE_B=oras attach --artifact-type sbom/example $REFERENCE_A.Ref sbom.spdx --format json --no-tty | ConvertFrom-Json
notation sign $REFERENCE_B.Ref
```

### CI/CD

Bob is a DevOps engineer. He uses the ORAS GitHub Actions [Setup action](https://github.com/oras-project/setup-oras) to install ORAS in his CI/CD workflow. He wants to chain multiple ORAS commands in a Shell script to perform multiple operations.

For example, pull multiple files (layers) from a repository and filter out the file path of its first layer in the standard output, then pass the pulled first layer to the second command `docker import` for further operation. 

```yaml
jobs:
  example-job:
    steps:
      - uses: oras-project/setup-oras@v1
      - run: |
          oras pull $REGISTRY/$REPO:$TAG --format '{{.first .Files.Path}}' |
          docker import
```

## Proposal: format output into structured data

1. Use the `--format json` flag to change the default human-readable prettified output to machine-readable raw JSON. Users can still use `--format '{{toPrettyJson .}}'` or `--pretty` to get prettified output for some commands.
2. Use the `--format` with [Go template](https://pkg.go.dev/text/template) to custom the output fields. 

## Desired user experience for proposal 1

For review convenience, this doc shows the output in most of the sample ORAS commands with prettified JSON format.

### oras pull 

Pull an artifact and display its metadata as formatted JSON in standard output. The following fields should be formatted in a JSON output:

- `ref`: full artifact reference by digest, e.g, `$REGISTRY/$REPO@$DIGEST`
-  `files`: a list of downloaded files
    -  `path`: the absolute file path of the pulled file (layer)
    -  `ref`: full reference by digest of the pulled file (layer)
    -  `mediaType`: media type of the pulled file (layer) 
    -  `digest`: digest of the pulled file (layer) 
    -  `size`: file size in bytes

Pull a single file and show the its descriptor data including `path` and `ref` as pretty JSON in standard output:

```bash
oras pull $REGISTRY/$REPO:$TAG --format '{{toPrettyJson .}}'
```

```json
{
    "ref": "$REGISTRY/$REPO@$DIGEST",
    "files" : [
            "path":"/home/user/path1/",
            "ref": "$REGISTRY/$REPO@$layer0_digest",
            {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12
            }
        }
    ]
}
```

Pull an artifact and display its descriptor as raw JSON in standard output.

```bash
oras pull $REGISTRY/$REPO:$TAG --format json
```

```
{"ref":"$REGISTRY/$REPO@$DIGEST","files":[{"path":"/home/user/path1/","ref":"$REGISTRY/$REPO@$layer0_digest","mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:42e2c5e85dd5a21dd516dd6f5a043db9ae549b8f464b049d165fc5765ebb4cad","size":591}]}
```

Pull multiple files and show their descriptor data including `path` and `ref` as pretty JSON in standard output.

```bash
oras pull $REGISTRY/$REPO:$TAG --format '{{toPrettyJson .}}'
```

```json
{
    "ref": "$REGISTRY/$REPO@$DIGEST",
    "files" : [
        {
            "path":"path1/artifact1.json",
            "ref": "$REGISTRY/$REPO:$layer0_digest",
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12,
        },
        {
            "path":"path2/artifact2.json",
            "ref": "$REGISTRY/$REPO:$layer1_digest",
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:4add5a911ba64df27458ed8229da804a26d2a84f4b8b650937ec8f73cd8be2c7",
            "size": 12,
        }
        }
    ]
}
```

### oras attach

Attach two files to an image and show the descriptor of the attached files in JSON format.

```bash
oras attach $REGISTRY/$REPO:$TAG --artifact-type example/sbom sbom.spdx --artifact-type example/vul-scan vul-report.json  --format '{{toPrettyJson .}}'
```

```json
{
    "files" : [
        {
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.created":"2023-11-29T06:32:43Z"
            },
            "artifactType" : "example/sbom",
            "ref": "$REGISTRY/$REPO@$DIGEST_1"
        },
        {
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a27",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.created":"2023-11-29T06:32:43Z"
            },
            "artifactType" : "example/vul-scan",
            "ref": "$REGISTRY/$REPO@$DIGEST_2"
        }
    ]
}
```

### oras push

Push two files to a repository and show the descriptor of the pushed files in pretty JSON format.

```bash
oras push $REGISTRY/$REPO:$TAG  --format '{{toPrettyJson .}}'
```

```json
{
    "files" : [
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.title": "hello.txt"
            },
            "artifactType": "application/vnd.example+type",
            "ref": "$REGISTRY/$REPO@$DIGEST"
        },
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a27",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.title": "hello.txt"
            },
            "artifactType": "application/vnd.example+type",
            "ref": "$REGISTRY/$REPO@$DIGEST"
        }
    ]
}
```

### oras discover

Discover an artifact's referrers. The default output should be listed in a tree view.

```bash
oras discover localhost:5000/hello:v1
```

```console
localhost:5000/hello/demo@sha256:04beb34cd24389147b4642a828b47fabefa722dea794dc3834567cf014ab0fe6
└── application/vnd.oci.empty.v1+json
    ├── sha256:1b82e249d83eb4881b8bf4ff9cf13a28799907ddc624b4c3c9140fa77d54fa42
    ├── sha256:28653e2bb5b5a75393c3a8b58ed9998796299b41dc1ff1f55b9f0844ad7ba39c
```

Discover an artifact's referrers manifest in pretty JSON. 

```bash
oras discover localhost:5000/hello:v1 --format '{{toPrettyJson .}}'
```

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "artifactReference": "localhost:5000/hello@sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:1b82e249d83eb4881b8bf4ff9cf13a28799907ddc624b4c3c9140fa77d54fa42",
      "size": 731,
      "annotations": {
        "org.opencontainers.image.created": "2023-11-22T07:27:41Z"
      },
      "artifactType": "application/vnd.oci.empty.v1+json",
      "ref": "localhost:5000/hello@sha256:1b82e249d83eb4881b8bf4ff9cf13a28799907ddc624b4c3c9140fa77d54fa42"
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:28653e2bb5b5a75393c3a8b58ed9998796299b41dc1ff1f55b9f0844ad7ba39c",
      "size": 630,
      "annotations": {
        "org.opencontainers.image.created": "2023-11-25T10:32:54Z"
      },
      "artifactType": "application/vnd.oci.empty.v1+json",
      "ref": "localhost:5000/hello@sha256:28653e2bb5b5a75393c3a8b58ed9998796299b41dc1ff1f55b9f0844ad7ba39c"
    }
  ]
}
```

## Desired user experience for proposal 2

In order to filter out the specified fields of the descriptor in the output, format the output using the given [Go template](https://golang.org/pkg/text/template/). The keys of the returned JSON can be used as the values for the `--format` flag.

### Format the output using the given Go template

For example, push an artifact to a repository and filter out the value of `reference` and `artifactType` of the pushed artifact in the standard output.

```bash
oras push $REGISTRY/$REPO:$TAG --format "{{.Ref}}', '{{.ArtifactType}}"
```

```console
"localhost:5000@sha256:85438e6598bf35057962fff34399a362d469ca30a317939427fca6b7a289e70d, "application/vnd.example+type"  
```

For example, pull a file and filter out the specified fields `mediaType`, `reference`, `size` of the pulled file in the standard output.

```bash
oras pull $REGISTRY/$REPO:$TAG --format "{{.MediaType}}, {{.Ref}}, {{.Size}}"
```

```console
"application/vnd.oci.image.layer.v1.tar","sha256:85438e6598bf35057962fff34399a362d469ca30a317939427fca6b7a289e70d", 12
```

For example, filter out the specified annotation value of an artifact by the key name `org.opencontainers.image.created`, with the [index function](https://pkg.go.dev/text/template#pkg-functions) defined in Go template.

```bash
oras discover localhost:5000/hello:v1 --format '{{index .Manifest.Annotations "org.opencontainers.image.created"}}'
```

```console
"2023-11-29T06:32:43Z"
```

## FAQ

- Why not consider extending the existing `--output` flag to enable JSON formatted output?

`--output` has been used in other oras commands like `oras pull`, `oras manifest fetch` to output the file directory or file, it will be a breaking change if we extend the
`--output` flag to enable JSON formatted output. 

- Why ORAS chooses [Go template](https://pkg.go.dev/text/template)?

[Go template] is a powerful method to customize output you want It allows users to manipulate the output format of certain commands. It provides access to data objects and additional functions that are passed into the template engine programmatically. It also has some useful libraries that have strong functions for Go’s template language to manipulate the output data, such as [Sprig](https://masterminds.github.io/sprig/).
