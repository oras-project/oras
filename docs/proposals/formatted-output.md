# Formatted ORAS CLI output 

ORAS has prettified output designed for humans. However, for machine processing, especially in some automation scenarios like scripting and CI/CD pipelines, developers want to perform batch operations and chain different commands with ORAS, as well as filtering, modifying, and sorting objects based on the outputs that are emitted by the ORAS command. Developers expect the ORAS output can be generated as machine-readable text not only the prettified or tabular data, so that they can use the formatted outputs like JSON output to perform further advanced operations. 

The formatted output is not intended to supersede the prettified human-readable and friendly output text of ORAS CLI. It aims to provide programming-friendly experience for developers who want to automate their workflows on machines especially on Unix and most Unix-like operating systems, without parsing unstructured text. It will increase the developer experience for ORAS in automation and scripting scenarios. 

## Scenarios

### Scripting

Alice is a developer who wants to batch operations with ORAS in her Shell script. In order to automate some routine workflow in containers secure supply chain scenario, she wants the machine to get the image digest from the JSON output objects that are emitted by `oras push`, then use utility like [xargs](https://en.wikipedia.org/wiki/Xargs) to enable an ORAS command to act on the output of another command and perform further steps. In this way, she can chain commands together. For example, she can use `oras attach` to attach an SBOM to the image using its image digest as a argument outputted from `oras push`.

For example, push an artifact to a registry and generate the artifact reference in the standard output, then attach an SBOM to the artifact using the artifact reference (`$REGISTRY/$REPO:$DIGEST`) outputted from the first command, finally sign the attached SBOM with another tool against the SBOM file's reference (`$REGISTRY/$REPO:$DIGEST`) outputted from the last step.

```shell
oras push $REGISTRY/$REPO:$TAG hello.txt --format '{{.Reference}}' |\
xargs -I _ oras attach --artifact-type sbom/example _ sbom.spdx --format '{{.Reference}}' |\
xargs -I _ notation sign _ 
```

### CI/CD

Bob is a DevOps engineer. He uses the ORAS GitHub Actions [Setup action](https://github.com/oras-project/setup-oras) to install ORAS in his CI/CD workflow. He wants to chain multiple ORAS commands in a Shell script to perform multiple operations.

```yaml
jobs:
  example-job:
    steps:
      - uses: oras-project/setup-oras@v1
      - run: |
          oras pull $REGISTRY/$REPO:$TAG --format '{{.Files[0].Path}}' |\
          docker import _
```

## Proposal: format output into structured data

- Use the `--format json` flag to change the default human-readable prettified output to machine-readable raw JSON. Users can still use `--format '{{jsonPretty .}}'` to get prettified output for some commands.
- Use the `--format` with Go template to custom the output fields. 

## Use cases

### oras pull 

Pull an artifact and display its metadata as formatted JSON in standard output. The following fields should be formatted in a JSON output:

- `reference`: full artifact reference by digest, e.g, `$REGISTRY/$REPO:$DIGEST`
-  `files`: a list of downloaded files
    -  mediaType: media type of the pulled file (layer) 
    -  digest: digest of the pulled file (layer) 
    -  size: file size in bytes
    -  annotations: annotations of the pulled file (layer)
    -  path: the absolute file path of the pulled file (layer)
    -  reference: full reference by digest of the pulled file (layer)


Pull a single file and show the manifest of the pulled file as pretty JSON in standard output:

```shell
$ oras pull $REGISTRY/$REPO:$TAG --format json {{ toPrettyJson }}
```

```json
{
    "reference": "$REGISTRY/$REPO:$DIGEST",
    "files" : [
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.created":"2023-11-29T06:32:43Z"
            },
            "path":"path1/artifact1.json"
            "reference": "$REGISTRY/$REPO:$layer0_digest"
        }
    ]
}
```

Pull an artifact and display its manifest as raw JSON in standard output.

```shell
$ oras pull $REGISTRY/$REPO:$TAG --format json
```

```
{"mediaType":"application/vnd.oci.image.manifest.v1.tar","digest":"sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a2","size":12,"annotations":{"org.opencontainers.image.created":"2023-11-29T06:32:43Z"},"path","path1/artifact1.json","reference":"$REGISTRY/$REPO:$digest"}
```

pull multiple files and show their manifests as pretty JSON in standard output.

```shell
$ oras pull $REGISTRY/$REPO:$TAG --format '{{jsonPretty .}}'
```

```json
{
    "reference": "$REGISTRY/$REPO:$DIGEST",
    "files" : [
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.created":"2023-11-29T06:32:43Z"
            },
            "path":"path1/artifact1.json"
            "reference": "$REGISTRY/$REPO:$layer0_digest"
        },
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:4add5a911ba64df27458ed8229da804a26d2a84f4b8b650937ec8f73cd8be2c7",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.created":"2023-11-30T06:32:43Z"
            },
            "path":"path2/artifact2.json"
            "reference": "$REGISTRY/$REPO:$layer1_digest"
        }
    ]
}
```

### oras attach

Attach an artifact to an image and show the manifest of the attached file in JSON format.

```shell
$ oras attach --artifact-type example/sbom $REGISTRY/$REPO:$TAG sbom.spdx --format json {{ toPrettyJson }}
```

```json
{
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
    "size": 12,
    "annotations": {
        "org.opencontainers.image.created":"2023-11-29T06:32:43Z"
    },
    "artifactType" : "example/sbom",
    "reference": "$REGISTRY/$REPO:$digest"
}
```

### oras push

Push an artifact to a repository and show the metadata of the pushed artifact.

```shell
$ oras push $REGISTRY/$REPO:$TAG --format '{{jsonPretty .}}'
```

```json
{
    "mediaType": "application/vnd.oci.image.layer.v1.tar",
    "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
    "size": 12,
    "annotations": {
        "org.opencontainers.image.title": "hello.txt"
    },
    "artifactType": "application/vnd.example+type",
    "reference": "$REGISTRY/$REPO:$digest"
}
```

### oras discover

Discover an artifact's referrers. The default output should be listed in a tree view.

```shell
$ oras discover localhost:5000/hello:v1
```

```
localhost:5000/hello/demo@sha256:04beb34cd24389147b4642a828b47fabefa722dea794dc3834567cf014ab0fe6
└── application/vnd.oci.empty.v1+json
    ├── sha256:1b82e249d83eb4881b8bf4ff9cf13a28799907ddc624b4c3c9140fa77d54fa42
    ├── sha256:28653e2bb5b5a75393c3a8b58ed9998796299b41dc1ff1f55b9f0844ad7ba39c
```

Discover an artifact's referrers manifest in JSON. Consider `oras discover` is more likely used by terminal (human) than machine, the default formatted output with `--format json` should be pretty JSON.

```
$ oras discover localhost:5000/hello:v1 --format json
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
      "reference": "localhost:5000/hello@sha256:1b82e249d83eb4881b8bf4ff9cf13a28799907ddc624b4c3c9140fa77d54fa42"
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:28653e2bb5b5a75393c3a8b58ed9998796299b41dc1ff1f55b9f0844ad7ba39c",
      "size": 630,
      "annotations": {
        "org.opencontainers.image.created": "2023-11-25T10:32:54Z"
      },
      "artifactType": "application/vnd.oci.empty.v1+json",
      "reference": "localhost:5000/hello@sha256:28653e2bb5b5a75393c3a8b58ed9998796299b41dc1ff1f55b9f0844ad7ba39c"
    }
  ]
}
```

### oras manifest fetch

Fetch a specified layer and show the formatted JSON of the fetched manifest in formatted JSON.

```shell
oras manifest fetch $REGISTRY/$REPO:$TAG --format {{.Layers[0].Reference}}
```

```json
{
    "reference": "$REGISTRY/$REPO:$manifest_digest",
    "layers" : [
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26",
            "size": 12,
            "annotations": {
                "org.opencontainers.image.created":"2023-11-29T06:32:43Z"
            },
            "reference": "$REGISTRY/$REPO:$layer0_digest",
        }
    ]
}
```

### Format the output using the given Go template

In order to custom the manifest fields in the output, format the output using the given [Go template](https://golang.org/pkg/text/template/). The keys of the returned JSON can be used as the values for the `--format` flag.

For example, push an artifact to a repository and show the `reference` and `artifactType` of the pushed artifact in the standard output.

```shell
$ oras push $REGISTRY/$REPO:$TAG --format "{{.Reference}}', '{{.ArtifactType}}"
```

```console
"localhost:5000@sha256:85438e6598bf35057962fff34399a362d469ca30a317939427fca6b7a289e70d, "application/vnd.example+type"  
```

For example, pull a file and display the specified fields `mediaType`, `reference`, `size` of the pulled file in the standard output.

```shell
$ oras pull $REGISTRY/$REPO:$TAG --format "{{.MediaType}}, {{.Reference}}, {{.Size}}"
```


```console
"application/vnd.oci.image.layer.v1.tar","sha256:85438e6598bf35057962fff34399a362d469ca30a317939427fca6b7a289e70d", 12
```

For example, get the specified annotation value of an artifact by the key name `org.opencontainers.image.created`, with the [index function](https://pkg.go.dev/text/template#pkg-functions) defined in Go template.

```shell
$ oras discover localhost:5000/hello:v1 --format '{{index .Manifest.Annotations "org.opencontainers.image.created"}}'

```

```console
"2023-11-29T06:32:43Z"
```


