# Formatted ORAS CLI output 

ORAS has prettified output designed for humans. However, for machine processing, especially in some automation scenarios like scripting and CI/CD pipelines, developers want to perform batch operations and chain different commands with ORAS, as well as filtering, modifying, and sorting objects based on the outputs that are emitted by the ORAS command. Developers expect the ORAS output can be generated as machine-readable text not only the prettified or tabular data, so that they can use the formatted outputs like JSON format to perform further advanced operations. 

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
          PATH=`oras pull $REGISTRY/$REPO:$TAG --format '{{.first .Files.Path}}'`
          docker import $PATH
```

## Proposal: format output into structured data

1. Use the `--format json` flag to output prettified JSON. 
2. Use the `--format` with [Go template](https://pkg.go.dev/text/template) to custom the output fields. Users can still use `--format '{{toRawJson .}}'` to get raw JSON output.

## Desired user experience for proposal 1

For review convenience, this doc shows the output in most of the sample ORAS commands with prettified JSON format.

### oras pull 

Pull a repository and display its metadata as formatted JSON in standard output. The following fields should be formatted in a JSON output:

- `Ref`: full artifact reference by digest, e.g, `$REGISTRY/$REPO@$DIGEST`
-  `Files`: a list of downloaded files
    -  `Path`: the absolute file path of the pulled file (layer)
    -  `Ref`: full reference by digest of the pulled file (layer)
    -  `MediaType`: media type of the pulled file (layer) 
    -  `Digest`: digest of the pulled file (layer) 
    -  `Size`: file size in bytes

Pull a repository that contains multiple layers (files) and show their descriptor metadata as pretty JSON in standard output.

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

Pull a repository that contains multiple layers (files) and display its descriptor metadata as raw JSON in standard output.

```bash
oras pull $REGISTRY/$REPO:$TAG --format '{{toRawJson .}}'
```

```
{"Ref":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186111","Files":[{"Path":"/home/user/oras-install/sbom.spdx","Ref":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222","MediaType":"application/vnd.oci.image.manifest.v1+json","Digest":"sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222","Size":820},{"Path":"/home/user/oras-install/vul-scan.json","Ref":"localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b","MediaType":"application/vnd.oci.image.manifest.v1+json","Digest":"sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b","Size":820}]}
```

### oras attach

Attach two files to an image and show the descriptor metadata of the attached files in JSON format.

```bash
oras attach $REGISTRY/$REPO:$TAG --artifact-type example/vul-scan vul-report.json --artifact-type example/sbom sbom.spdx --format json
```

```json
{
    "Files" : [
        {
  "Ref": "localhost:5000/pipe/demo@sha256:5a23319624a3cea05aea5f9dfaf716fba1e7edf8c60d8389af35cebd6f605d30",
  "MediaType": "application/vnd.oci.image.manifest.v1+json",
  "Digest": "sha256:5a23319624a3cea05aea5f9dfaf716fba1e7edf8c60d8389af35cebd6f605d30",
  "Size": 939,
  "Annotations": {
        "org.opencontainers.image.title": "vul-report.json"
      }
        },
  {
  "Ref": "localhost:5000/pipe/demo@sha256:5a23319624a3cea05aea5f9dfaf716fba1e7edf8c60d8389af35cebd6f605333",
  "MediaType": "application/vnd.oci.image.manifest.v1+json",
  "Digest": "sha256:5a23319624a3cea05aea5f9dfaf716fba1e7edf8c60d8389af35cebd6f605333",
  "Size": 929,
  "Annotations": {
        "org.opencontainers.image.title": "sbom.spdx"
      }
  }
    ]
}
```

### oras push

Push two files to a repository and show the descriptor of the pushed files in pretty JSON format.

```bash
oras push $REGISTRY/$REPO:$TAG  --format json
```

```json
"Ref": "localhost:5000/pipe/demo@sha256:80da7a36f42ab62eeac5382e99fd203149749cf2a861167ada620075e4e6edd4",
"Files": [
    {
      "Ref": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222",
      "MediaType": "application/vnd.oci.image.layer.v1.tar",
      "Digest": "sha256:6cb759c4296e67e35b0367f3c0f51dfdb776a0c99a45f39d0476e43d82696d65",
      "Size": 14477,
      "Annotations": {
        "org.opencontainers.image.title": "sbom.spdx"
      }
    },
    {
      "Ref": "localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd186222",
      "MediaType": "application/vnd.oci.image.layer.v1.tar",
      "Digest": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "Size": 0,
      "Annotations": {
        "org.opencontainers.image.title": "hello.txt"
      }
    }
  ]
```

### oras manifest fetch

Fetch a manifest and filter out its reference and media type in standard output:

```bash
oras manifest fetch $REGISTRY/$REPO:$TAG --format '{{.Ref}}', '{{.Config.mediaType}}'
```

```json
localhost:5000/oras@sha256:7414904f07f515f48fe4afeaf876e3151039a81e7177b9c66e9e7ed6dd18669b, application/vnd.oci.empty.v1+json
```

### oras discover

Discover an artifact's referrers. The default output should be listed in a tree view.

```bash
oras discover $REGISTRY/$REPO:$TAG
```

```console
localhost:localhost:5000/hello@sha256:5cb894d0c94c56894e160ad2eeb19a123b4d2155374e7709f43f8c0c2f249fe2
├── application/vnd.oci.empty.v1+json
│   └── sha256:0db683b656132cede5360f42bc52541f3386b30ce685e6e63ff93ced54423fb8
└── application/vnd.cncf.notary.signature
    └── sha256:476d43120d3799fa76d7706f741beca73f5ff4149c8b6db3bd516a73d4a82fc1
```

Discover an artifact's referrers manifest in pretty JSON. 

```bash
oras discover localhost:5000/hello:v1 --format json
```

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:0db683b656132cede5360f42bc52541f3386b30ce685e6e63ff93ced54423fb8",
      "size": 964,
      "annotations": {
        "org.opencontainers.image.created": "2023-12-14T13:48:32Z"
      },
      "artifactType": "application/vnd.oci.empty.v1+json"
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:476d43120d3799fa76d7706f741beca73f5ff4149c8b6db3bd516a73d4a82fc1",
      "size": 728,
      "annotations": {
        "io.cncf.notary.x509chain.thumbprint#S256": "[\"792265ec6b22f0a87c7b3d980319d51a76a382de1b7a47bd877bb4e5a9beb637\"]",
        "org.opencontainers.image.created": "2023-12-14T14:41:56Z"
      },
      "artifactType": "application/vnd.cncf.notary.signature"
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
oras discover $REGISTRY/$REPO:$TAG --format '{{index .Manifest.Annotations "org.opencontainers.image.created"}}'
```

```console
"2023-11-29T06:32:43Z"
```

## FAQ

- Why choose to use `--format` flag to enable JSON formatted output instead of extending the existing `--output` flag?

ORAS follows the [GNU](https://www.gnu.org/prep/standards/html_node/Option-Table.html#Option-Table) design principles. ORAS uses `--output` to output a file or directory and uses `--format` to format the output into JSON or using the given Go template. Popular tools like Docker, Podman, and Skopeo also follow this kind of design principles in the similar formatted output feature. 

In addition, it will be a breaking change if we extend the `--output` flag to enable JSON formatted output. 

- Why ORAS chooses [Go template](https://pkg.go.dev/text/template)?

Go template is a powerful method to customize output you want It allows users to manipulate the output format of certain commands. It provides access to data objects and additional functions that are passed into the template engine programmatically. It also has some useful libraries that have strong functions for Go’s template language to manipulate the output data, such as [Sprig](https://masterminds.github.io/sprig/).

- What's the difference of the output when use `--output` and `--format` in `oras manifest fetch`?

`--output` can generate a file with raw JSON data of the image manifest. `format` can display prettified JSON in the standard out.

```bash
oras manifest fetch $REGISTRY/$REPO:$TAG --output manifest.json
```

See the content in the `manifest.json`. It should be raw JSON data of the fetched image manifest.

```json
{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.unknown.artifact.v1","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:6cb759c4296e67e35b0367f3c0f51dfdb776a0c99a45f39d0476e43d82696d65","size":14477,"annotations":{"org.opencontainers.image.title":"sbom.spdx"}},{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:54c0e84503c8790e03afe34bfc05a5ce45c933430cfd9c5f8a99d2c89f1f1b69","size":6639,"annotations":{"org.opencontainers.image.title":"scan-test-verify-image.json"}}],"annotations":{"org.opencontainers.image.created":"2023-12-13T15:08:49Z"}}
```

See the prettified JSON output of an image manifest when use `--format json`:

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
    "org.opencontainers.image.created": "2023-12-13T15:08:49Z"
  }
}
```
