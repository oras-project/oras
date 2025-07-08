# Proposal: Portable Backup and Restore of OCI Artifacts and Images

Authors: @TerryHowe @FeynmanZhou 

## Overview

As the adoption of referrers and OCI artifacts expands beyond container images to include signatures, SBOMs, Helm charts, and other supply chain metadata, users face increasing challenges in managing and preserving complete repository states across environments. Existing tooling lacks a consistent and efficient way to perform portable, repository-level backups and restores that include all images and associated referrers.

This proposal introduces a holistic solution with two new commands `oras backup` and `oras restore` to the ORAS CLI to address these gaps. The proposed solution enables users to archive entire repositories or specific tagged artifacts from an OCI registry into a portable, structured format (directory or archive), and to restore them reliably back into any registry. This supports critical scenarios such as disaster recovery, migration between environments, air-gapped deployments, and supply chain integrity validation.

By providing native support for comprehensive backup and restore workflows, this enhancement improves user experience, simplifies operational tooling, and ensures that all artifacts including linked referrers are preserved with integrity and fidelity according to OCI specifications.

## Problem Statement & Motivation

Problem statement and motivation are documented in the [scenario doc](https://github.com/oras-project/oras/pull/1778).

## Scenarios

The [scenario doc](https://github.com/oras-project/oras/pull/1778) illustrates real-world scenarios highlighting these challenges and how unified, structured backup and restore functionality built into `oras` can significantly improve user experience, operational efficiency, and supply chain security.

## Existing Solutions

* `docker save/load` supports exporting and importing images but not referrers or OCI artifacts.
* `oras pull/push` handles single artifacts, but not repository-level operations.
* Users can write scripts to persist multiple artifacts and repositories in local OCI layout and distribute to registries via `oras copy`, but it's error-prone for users to do so.

## Proposal

This document proposes two new command sets, `oras backup` and `oras restore`, to address the identified problems and support the scenarios outlined above. It also describes the desired user experience for backing up and restoring artifacts, images, and repositories between a registry and the local environment. This proposal meets user expectations of portability, structure, and artifact completeness using OCI specifications.

### New Command/Parameters in the CLI

#### Command: `oras backup`

**Short summary:**
Backup OCI artifacts and repositories from a registry into a structured, portable OCI image layout or archive tarball file locally.

**Syntax:**
```console
oras backup --flags <registry>/<repository>[:<ref1>[,<ref2>...]] [...]
```

**Output:**
An OCI image layout directory or `.tar` archive containing the images, artifacts, their metadata, and optional referrers.

**New Flags:**

* `--output <path>`: Required. Target directory path or tar file path to write in local filesystem.
* `--include-referrers`: Back up the image and its linked referrers (e.g., attestations, SBOMs).

> [!NOTE] 
> The file extension determines the output format. If the output path does not include a file extension, it is assumed that the output should be a directory. When an unsupported extension such as `.zip` or `.tar.gz` is specified, `oras` should display a warning indicating that the format is not supported. In such cases, it will proceed to create a directory at the specified path instead.

**Common flags:**

* `--concurrency <int>`: Number of parallel fetch operations. Default: `3`.
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
```console
oras restore --flags <registry>/<repository>[:<ref1>[,<ref2>...]] [...]
```

**Output:**
Artifacts are uploaded to the target registry/registries as specified.

**New flags:**

- `--input <path>`: Required. Restore from a folder or archive file to registry.
- `--exclude-referrers`: Restore the image from backup without the referrers

**Common flags:**

* `--concurrency <int>`: Number of parallel upload operations. Default: `3`.
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
* `--distribution-spec string`: [Preview] set OCI distribution spec version and API option for target. Options: v1.1-referrers-tag, v1.1-referrers-api
* `--no-tty`: Disable progress bars.

### User Experience in the CLI

**Offline Snapshot for Air-Gapped Environments**

Create a snapshot of a sample image `registry-a.k8s.io/kube-apiserver:v1` and its referrer (e.g. signature) for an air-gapped environment:

```console
oras backup registry-a.k8s.io/kube-apiserver:v1 --include-referrers --output airgap-snapshot.tar
```

Transfer the `.tar` file to the air-gapped system via a secured channel. Restore the tarball from local to another registry:

```console
oras restore registry-b.k8s.io/kube-apiserver:v1 --input airgap-snapshot.tar
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

**Backup and Restore an Entire Repository and Tagged Artifacts**

Assume two tags `v1` and `v2` are stored in a repository `registry.k8s.io/kube-apiserver`. Backup the entire repo to a tarball and restore it to another registry:

```console
# Backup a repository from a registry to a local compressed tarball. All tags and their referrers will be included.
oras backup --output backup.tar --include-referrers registry-a.k8s.io/kube-apiserver
```

Transfer the backup file to new environment via secure channels (e.g., BitLocker-enabled removable drives)

Restore images and referrer artifacts from a local backup file to a target registry. All tags and their referrers will be included by default.

```console
oras restore --input backup.tar registry-b.k8s.io/kube-apiserver
```

List all tags from the repo `registry-b.k8s.io/kube-apiserver`

```console
$ oras repo tags registry-b.k8s.io/kube-apiserver
v1
v2
```

**Backup and Restore Multiple Repositories**

Backup multiple repositories from a registry to a local OCI image layout

```console
$ oras backup registry.k8s.io/kube-apiserver registry.k8s.io/kube-controller-manager --output ./k8s-control-plane
```

List the backup repositories in the OCI image layout. 

```console
$ oras repo list --oci-layout k8s-control-plane 
registry.k8s.io/kube-apiserver
registry.k8s.io/kube-controller-manager
```

Restore them from local OCI image layout to two repositories respectively in a registry

```console
oras restore localhost:5000/kube-apiserver localhost:5000/kube-controller-manager --input ./k8s-control-plane
```

## Summary

The `oras backup` and `oras restore` commands introduce a structured, OCI-compliant way to persist and rehydrate artifacts and referrers, bridging a critical gap in the current functionality of the `oras` CLI. This enhancement empowers users with flexible, scriptable, and portable tooling for registry state management.