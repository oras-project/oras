# Proposal: Portable Backup and Restore of OCI Artifacts, Images, and Repositories

## Overview

As the adoption of referrers and OCI artifacts expands beyond container images to include signatures, SBOMs, Helm charts, and other supply chain metadata, users face increasing challenges in managing and preserving complete repository states across environments. Existing tooling lacks a consistent and efficient way to perform portable, repository-level backups and restores that include all images and associated referrers.

This proposal introduces a holistic solution with two new commands `oras backup` and `oras restore` to the ORAS CLI to address these gaps. The proposed solution enables users to archive entire repositories or specific images with referrers from an OCI registry into a portable, structured format (directory or archive), and to restore them reliably back into any registry. This supports critical scenarios such as disaster recovery, migration between isolated environments, air-gapped deployments, and supply chain integrity validation.

By providing native support for comprehensive backup and restore workflows, this enhancement improves user experience, simplifies operational tooling, and ensures that all artifacts including linked referrers are preserved with integrity and fidelity according to OCI specifications.

## Problem Statement & Motivation

Problem statement and motivation are documented in the [scenario doc](./backup-restore-scenarios.md).

## Scenarios

The [scenario doc](./backup-restore-scenarios.md) illustrates real-world scenarios highlighting these challenges and how unified, structured backup and restore functionality built into `oras` can significantly improve user experience, operational efficiency, and supply chain security.

## Existing Solutions

* `docker save/load` supports exporting and importing images but not referrers or OCI artifacts.
* `oras pull/push` handles single artifacts, but not repository-level operations.
* Users can write scripts to persist multiple artifacts and repositories in local OCI layout and distribute to registries via `oras copy`, but it's error-prone for users to do so.

## Proposal

This document proposes two new command, `oras backup` and `oras restore`, to address the identified problems and support the scenarios outlined above. It also describes the desired user experience for backing up and restoring artifacts, images, and repositories between a registry and the local environment. This proposal meets user expectations of portability, structure, and artifact completeness using OCI specifications.

### New Command/Parameters in the CLI

#### Command: `oras backup`

**Short summary:**
Backup OCI artifacts and repositories from a registry into a structured, portable OCI image layout or archive tarball file locally.

**Syntax:**
```bash
oras backup [flags] --output <path> <registry>/<repository>[:<ref1>[,<ref2>...]]
```

**Output:**
An OCI image layout directory or `.tar` archive containing the images, artifacts, their metadata, and optional referrers.

**New Flags:**

* `--output <path>`: Required. Target directory path or tar file path to write in local filesystem.
* `--include-referrers`: Optional. Back up the image and its linked referrers (e.g., attestations, SBOMs).

> [!NOTE] 
> The file extension determines the output format. `oras` supports `.tar` archive as the default format since OCI and Docker ecosystem uses `.tar` archive. If the output path does not include a file extension, it is assumed that the output should be a directory. When an unsupported extension such as `.zip` or `.tar.gz` is specified, `oras` should display a warning indicating that the format is not supported. In such cases, it will proceed to create a directory at the specified path instead.

**Common flags:**

* `--concurrency <int>`: Number of parallel fetch operations. Default: `3`.
* `--distribution-spec <string>`: [Preview] set OCI distribution spec version and API option for the registry. Options: v1.1-referrers-tag, v1.1-referrers-api.
* `--plain-http`: Allow insecure connections to registry without SSL check.
* `--insecure`: Allow connections to registries without valid TLS certificates.
* `--registry-config <path>`: Path to the authentication configuration file for the registry.
* `--username <string>`: Username for authenticating to the registry.
* `--password <string>`: Password for authenticating to the registry.
* `--password-stdin`: Read password from stdin.
* `--identity-token <string>`: Use bearer token for authentication.
* `--identity-token-stdin`: Read identity token from stdin.
* `--ca-file <path>`: Path to custom CA certificate file.
* `--cert-file <path>`: Path to client TLS certificate file.
* `--key-file <path>`: Path to client private key file.
* `--resolve <host:port:address[:address_port]>`: Customized DNS for registry.
* `--debug`: Output debug logs (implies `--no-tty`).
* `--no-tty`: Disable progress bars

#### Command: `oras restore`

**Short summary:**
Restore OCI artifacts or images from a local OCI image layout or archive into a registry.

**Syntax:**
```bash
oras restore [flags] --input <path> <registry>/<repository>[:<ref1>[,<ref2>...]]
```

**Output:**
Artifacts are uploaded to the target registry/registries as specified.

**New flags:**

- `--input <path>`: Required. Restore from a folder or archive file to registry.
- `--exclude-referrers`: Optional. Restore the image from backup excluding referrers.
- `--dry-run`: Optional. Simulate the restore process without actually uploading any artifacts to the target registry.

**Common flags:**

* `--concurrency <int>`: Number of parallel upload operations. Default: `3`.
* `--distribution-spec <string>`: [Preview] set OCI distribution spec version and API option for the registry. Options: v1.1-referrers-tag, v1.1-referrers-api.
* `--plain-http`: Allow insecure connections to registry without SSL check.
* `--insecure`: Allow connections to registries without valid TLS certificates.
* `--registry-config <path>`: Path to the authentication configuration file for the registry.
* `--username <string>`: Username for authenticating to the registry.
* `--password <string>`: Password for authenticating to the registry.
* `--password-stdin`: Read password from stdin.
* `--identity-token <string>`: Use bearer token for authentication.
* `--identity-token-stdin`: Read identity token from stdin.
* `--ca-file <path>`: Path to custom CA certificate file.
* `--cert-file <path>`: Path to client TLS certificate file.
* `--key-file <path>`: Path to client private key file.
* `--resolve <host:port:address[:address_port]>`: Customized DNS for registry.
* `--debug`: Output debug logs (implies `--no-tty`).
* `--no-tty`: Disable progress bars.

### User Experience in the CLI

The desired end-to-end user experience of using `oras backup` and `oras restore` to address the identified problems and support the outlined user scenarios is illustrated below.

#### Backup an Entire Repository to a Tarball and Restore to Another Registry

Assume two tags `v1` and `v2` are stored in a repository `registry.k8s.io/kube-apiserver` and each tag has one referrer. Backup the entire repo to a tarball and restore it to another registry:

```bash
## Backup a repository from a registry to a local tarball. All tags and their referrers will be included.
oras backup --output backup.tar --include-referrers registry-a.k8s.io/kube-apiserver
```

Alternatively, `oras` should also enables users to choose which tags within a repository to back up. For example, back up `v1` and `v2` tags with their referrers:

```bash
oras backup registry-a.k8s.io/kube-apiserver:v1,v2 --include-referrers --output backup.tar
```

Upon success, the output will be:

```console
## <progress_bar>
Found 2 tag(s) in registry-a.k8s.io/kube-apiserver: v1, v2

✓ Pulled  application/vnd.oci.image.config.v1+json                                                               2.26/2.26 KB  100.00%  447ms
  └─ sha256:45a5868eb9f1dfbce42513000964664014789a43310865b0c8461e773e9972b9
✓ Pulled  application/vnd.oci.image.layer.v1.tar+gzip                                                            25.6/25.6 MB  100.00%     4s
  └─ sha256:149362fdfa6e6a5d9f009b896da3be3172c395ba2287b57d4969f3f46e573055
✓ Pulled  application/vnd.cncf.notary.signature                                                                 1.85/1.85 MB  100.00%  898ms
  └─ sha256:9b666bc868511a0f2d33a738a9ff0bd54eb750a72a832e8b59085d22bbdbaac2
✔ Pulled tag v1 and 1 referrer(s)


✓ Pulled  application/vnd.oci.image.config.v1+json                                                               2.24/2.24 KB  100.00%  353ms
  └─ sha256:f9248aac10f2f82e0970222e36cc7b71215b88e974e001282e5cd89797a82218
✓ Pulled  application/vnd.oci.image.layer.v1.tar+gzip                                                            28.3/28.3 MB  100.00%     3s
  └─ sha256:b08e2ff4391ef70ca747960a731d1f21a75febbd86edc403cd1514a099615808
✓ Pulled  application/vnd.cncf.notary.signature                                                                 1.85/1.85 MB  100.00%  890ms
  └─ sha256:854ad9e87ce93dae54ae1699837b2c812d2f373c3fb62625ea6992efa8f023c4
✔ Pulled tag v2 and 1 referrer(s)

## <status output>
Pulled tag v1 and 1 referrer(s)
Pulled tag v2 and 1 referrer(s)
Exporting to backup.tar
Exported to backup.tar (58.8 MB)
Successfully backed up 2 tag(s) from registry-a.k8s.io/kube-apiserver in 5s.
```

Transfer the backup file to new environment via secure channels (e.g., BitLocker-enabled removable drives).

Restore images and referrer artifacts from a local backup file to a target registry. All tags and their referrers will be included by default.

```console
oras restore --input backup.tar registry-b.k8s.io/kube-apiserver
```

Upon success, the output will be:

```console
## <progress_bar>
Loaded backup archive: backup.tar (58.8 MB)
Found 2 tag(s): v1, v2

✓ Pushed  application/vnd.oci.image.config.v1+json                                                               2.26/2.26 KB  100.00%  447ms
  └─ sha256:45a5868eb9f1dfbce42513000964664014789a43310865b0c8461e773e9972b9
✓ Pushed  application/vnd.oci.image.layer.v1.tar+gzip                                                            25.6/25.6 MB  100.00%     4s
  └─ sha256:149362fdfa6e6a5d9f009b896da3be3172c395ba2287b57d4969f3f46e573055
✓ Pushed  application/vnd.cncf.notary.signature                                                                 1.85/1.85 MB  100.00%  898ms
  └─ sha256:9b666bc868511a0f2d33a738a9ff0bd54eb750a72a832e8b59085d22bbdbaac2
✔ Pushed tag v1 and 1 referrer(s)

✓ Pushed  application/vnd.oci.image.config.v1+json                                                               2.24/2.24 KB  100.00%  353ms
  └─ sha256:f9248aac10f2f82e0970222e36cc7b71215b88e974e001282e5cd89797a82218
✓ Pushed  application/vnd.oci.image.layer.v1.tar+gzip                                                            28.3/28.3 MB  100.00%     3s
  └─ sha256:b08e2ff4391ef70ca747960a731d1f21a75febbd86edc403cd1514a099615808
✓ Pushed  application/vnd.cncf.notary.signature                                                                 1.85/1.85 MB  100.00%  890ms
  └─ sha256:854ad9e87ce93dae54ae1699837b2c812d2f373c3fb62625ea6992efa8f023c4
✔ Pushed tag v2 and 1 referrer(s)

## <status output>
Pushed tag v1 with 1 referrer(s)
Pushed tag v2 with 1 referrer(s)
Successfully restored 2 tag(s) to registry-b.k8s.io/kube-apiserver in 5s.
```

List all tags from the repo `registry-b.k8s.io/kube-apiserver`:

```console
$ oras repo tags registry-b.k8s.io/kube-apiserver
v1
v2
```

#### Backup and Restore an Image with Referrers 

Create a snapshot of a sample image `registry-a.k8s.io/kube-apiserver:v1` and its referrer (e.g. signature) for an air-gapped environment:

```bash
oras backup registry-a.k8s.io/kube-apiserver:v1 --include-referrers --output airgap-snapshot.tar
```

Upon success, the output will be:

```console
## <progress_bar>
Found 1 tag: v1
✓ Pulled  application/vnd.oci.image.config.v1+json                                                               2.26/2.26 KB  100.00%  447ms
  └─ sha256:45a5868eb9f1dfbce42513000964664014789a43310865b0c8461e773e9972b9
✓ Pulled  application/vnd.oci.image.layer.v1.tar+gzip                                                            25.6/25.6 MB  100.00%     4s
  └─ sha256:149362fdfa6e6a5d9f009b896da3be3172c395ba2287b57d4969f3f46e573055
✓ Pulled  application/vnd.cncf.notary.signature                                                                 1.85/1.85 MB  100.00%  898ms
  └─ sha256:9b666bc868511a0f2d33a738a9ff0bd54eb750a72a832e8b59085d22bbdbaac2

## <status output>
Pulled tag v1 and 1 referrer(s)
Exporting to airgap-snapshot.tar
Exported to airgap-snapshot.tar (58.8 MB)
Successfully backed up 1 tag from registry-a.k8s.io/kube-apiserver in 5s.
```

Transfer the `.tar` file to the air-gapped system via a secured channel. Restore the tarball from local to another registry:

```console
oras restore registry-b.k8s.io/kube-apiserver:v1 --input airgap-snapshot.tar
```

Upon success, the output will be:

```console
## <progress_bar>
Loaded backup archive: airgap-snapshot.tar (58.8 MB)
Found 1 tag: v1
✓ Pushed  application/vnd.oci.image.config.v1+json                                                               2.26/2.26 KB  100.00%  412ms
  └─ sha256:45a5868eb9f1dfbce42513000964664014789a43310865b0c8461e773e9972b9
✓ Pushed  application/vnd.oci.image.layer.v1.tar+gzip                                                            25.6/25.6 MB  100.00%  3.9s
  └─ sha256:149362fdfa6e6a5d9f009b896da3be3172c395ba2287b57d4969f3f46e573055
✓ Pushed  application/vnd.cncf.notary.signature                                                                 1.85/1.85 MB  100.00%  876ms
  └─ sha256:9b666bc868511a0f2d33a738a9ff0bd54eb750a72a832e8b59085d22bbdbaac2

## <status output>
Pushed tag v1 with 1 referrer(s)
Successfully restored 1 tag to registry-b.k8s.io/kube-apiserver
```

By default, the image and linked referrers are reliably restored to another registry with minimal steps. Users can use the `--exclude-referrers` flag to exclude linked referrers when using `oras restore`.

```console
$ oras discover registry-b.k8s.io/kube-apiserver:v1
registry-b.k8s.io/kube-apiserver@sha256:9081a6f83f4febf47369fc46b6f0f7683c7db243df5b43fc9defe51b0471a950
└── application/vnd.cncf.notary.signature
    └── sha256:78833f9c870d3b069cdd998cae33b935629399f24743e680ab3bebb90de76589
        └── [annotations]
            ├── org.opencontainers.image.created: "2025-06-10T20:25:53Z"
            └── io.cncf.notary.x509chain.thumbprint#S256: '["xxxxxx"]'
```

## Summary

The `oras backup` and `oras restore` commands introduce a structured, OCI-compliant way to persist and rehydrate artifacts and referrers, bridging a critical gap in the current functionality of the `oras` CLI. This enhancement empowers users with flexible, scriptable, and portable tooling for registry state management.