# Portable Backup, and Restore of OCI Artifacts and Images

Authors: @TerryHowe @FeynmanZhou 

## Overview

Organizations rely on container images and other OCI artifacts to build, deploy, and operate their applications. These images and artifacts are built locally and stored in public or private OCI registries. However, as organizations mature their supply chain security, they face increasing demands to efficiently acquire, migrate, promote, mirror, and backup images and artifacts across registries and local environments, while preserving provenance, integrity, and metadata.

Today, fragmented tooling and manual scripts make these tasks complex, error-prone, and operationally expensive. Common tools like `docker save/load` and `oras pull/push`, `oras copy` only cover parts of the workflow, often lacking support for referrers, deduplication, and structured backups. This results in brittle processes, duplicated blobs, missing attestations, and frustrated developers.

This document describes the challenges faced by users managing images and OCI artifacts across registries and local environment. It proposes a unified, reliable, and portable solution built into the `oras` CLI to address these gaps. In particular, this document motivates the need for structured backup and restore workflows that simplify artifact movement, ensure completeness, and integrate seamlessly with security and compliance practices.

## Problem Statement & Motivation

As organizations scale their software supply chain, acquiring and managing OCI artifacts is no longer as simple as pulling images from public registries e.g. Docker Hub or pushing them into a private registry. Security-conscious enterprises are imposing strict controls over how container images, Helm charts, AI models, SBOMs, attestations, and other OCI artifacts flow between registries and local environments.

Take for example, a global bank that cannot allow development teams to directly pull from public registries. Instead, they operate an internal acquisition pipeline where artifacts must first pass through vulnerability scans, software license checks, and supply chain attestation validation. Only after passing these gates are images and artifacts published to the bank's trusted registry for internal use. Similarly, security-critical systems maintain air-gapped environments. For them, acquiring artifacts requires carefully controlled offline transfers, with no room for manual errors or missing metadata.

Enterprises often maintain separate registries for development (DEV), quality assurance (QA), and production (PROD) to reduce the risk of untested artifacts reaching production. Promotion workflows rely on moving OCI artifacts across local environment and registries in a traceable, consistent, and secure manner.

Yet today, developers and users resort to fragmented, CLI tools like:

* `docker save/load` for container images.
* `oras pull/push` for OCI artifacts.
* `oras copy` for copying a single image with artifacts
* Ad-hoc scripts to cobble together backups and restores artifacts across different environments.

This patchwork approach brings significant limitations and problems:

* Backups lack structure, making recovery error-prone.
* Artifact referrers, attestations, and SBOMs are often lost in transit.
* Promotion and migration workflows are tedious, inconsistent, and fragile.
* Duplication of blobs wastes storage and network bandwidth.

The result is frustrated DevOps engineers, wasted resources, and security gaps that erode confidence in artifact management processes. There is a clear need for an integrated, reliable, and user-friendly solution to:

* Efficiently acquire and promote artifacts across local environment and registries.
* Create portable, structured backups in standard formats.
* Restore registry state, including all metadata and dependencies.
* Empower teams to meet security, compliance, and operational requirements across air-gapped, multi-cloud, and hybrid environments.

## Scenarios

This document illustrates real-world scenarios highlighting these challenges and how unified, structured backup and restore functionality built into `oras` can significantly improve user experience, operational efficiency, and supply chain security.

### Scenario 1: Creating Offline Snapshots for Air-Gapped Environments

Dave, a security engineer at a FinTech company. To create a snapshot of the image and its referrers in an air-gapped environment, Dave needs to run the following flow:

1. Packages the image and its referrers from an OCI image layout into a `.tar` for portability.
2. Copies compressed files via secured channels to the air-gapped network.
3. Restores all artifacts from a compressed file to an OCI registry in an air-gapped environment.

No unified snapshot solution available in `docker` or `oras`. Blobs duplicated across files and no assurance of artifact integrity and completeness causes problems to users. See [GitHub Issue #730](https://github.com/oras-project/oras/issues/730).

### Scenario 2: Image and Artifact Portability Across Isolated Environments

Cindy is a DevOps engineer working for a SaaS company that enforces strict network isolation between development, testing, staging, and production environments. Each environment has its own isolated OCI registry with no direct network connectivity between them. Cindy is responsible for promoting container images and artifacts across these isolated environments. For example, after building and testing an application image in the development environment, she needs to transfer it to the test and production environments.

However, direct registry-to-registry transfers are impossible due to network isolation and security policies. Today, Cindy uses `docker save/load`:

```bash
# Build image in development
docker build -t myapp:v1 .

# Export to a tar file
docker save myapp:v1 -o myapp.tar

# Manually transfer the tar file to test or production environment (e.g., via secure file transfer)

# Load image in the target environment
docker load -i myapp.tar

# Tag and push to the target environment registry
docker tag myapp:v1 registry.test.example.com/myapp:v1
docker push registry.test.example.com/myapp:v1
```

This process becomes even more complex when promoting artifacts beyond images, such as Helm charts, SBOMs, or signed attestations. Cindy must maintain separate scripts, manual tag tracking, and artifact validation. This process increased the risk of human error, artifact loss, and security gaps.

Manual tag mapping and artifact tracking introduce errors. The existing `docker`-based workflow cannot easily handle artifacts beyond images. Referrers and metadata are often lost during promotion, and in environments where `docker` is unavailable, image portability becomes impossible.

With a unified backup and restore solution, Cindy can efficiently move images and artifacts between isolated environments using a consistent, reliable workflow. The process preserves all referrers, tags, and metadata end-to-end, and works even in restricted environments without relying on `docker`. This ensures artifact integrity and completeness across the entire software delivery pipeline while significantly reducing operational complexity and human error.

### Scenario 3: Backup and Restore Repositories

Alice is an infrastructure engineer at a multi-cloud SaaS company responsible for maintaining container images and artifacts across multiple registries. These registries, hosted on different cloud providers, store critical application components required for her company’s services to run reliably across regions.

One of Alice's primary concerns is disaster recovery. If the repository is accidentally deleted, corrupted, or compromised, she needs a reliable way to restore it quickly to minimize downtime and operational impact. However, existing tools fall short. For example, docker save/load works only for images and requires the image to be pulled into Docker's internal storage (containerd image store) first, which is inefficient and limited. Worse, it doesn’t preserve referrer artifacts, such as SBOMs, signatures, or attestations, nor does it handle repository-level metadata.

Alice wants a simple, portable solution that allows her to archive an entire repository or even multiple repositories for an application stack, including all artifacts, tags, referrers, and metadata, into a single, compressed file that can be stored in durable blob storage. This archive acts as a disaster recovery backup, ready to be restored at any time to any registry, whether on-premises or in the cloud. Alternatively, Alice also wants to backup a repository to local system as an OCI image layout for local modification. 

With the backup in place, Alice can confidently proceed with registry maintenance tasks or operational changes, knowing that if something goes wrong—such as an accidental repository deletion—she can quickly restore the entire repository from her backup archive. This streamlined backup and restore process eliminates the need for manual scripting, reduces human error, and ensures artifact integrity and completeness. Alice can now maintain disaster recovery readiness across all her registries, improving operational resilience and reducing business risk.

### Scenario 4: Uploading and Downloading Image With Referrers Using `oras pull/push` 

Bob, a developer maintaining containerized applications. Bob wants to create a backup of OCI images from the registry to local disk for disaster recovery, local modification, or air-gapped use. However, Bob incorrectly uses `oras pull` and `oras push`:

```bash  
# Pull an image to local saved as a tarball  (incorrect usage)
oras pull foo.example.com/app/backend:v1.0.0 -o backend.tar
# Extract the tarball and modify it locally
tar -xf backend.tar
# Push the modified image back to the registry
oras push foo.example.com/app/backend:v1.0.1 ./extracted
```

At first glance, this appears to work. The image is pushed back to the registry. But when the image consumers try to pull and run the image, they encounter errors. The image referrers also lost when pulling and running the image. This is because `oras pull/push` only handles raw artifacts, not the full OCI image required for runnable images. Bob should be using `oras copy --recursive` with `--to-oci-layout` and `--from-oci-layout` to properly export and import an image with referrers in OCI image layout format:

```bash
oras copy --recursive --to-oci-layout registry.example.com/app/backend:v1.0.0 ./image-backup:v1.0.0 
```

To restore from an OCI image layout to an image:

```bash
oras copy --recursive --from-oci-layout ./image-backup:v1.0.0 registry.example.com/app/backend:v1.0.0
```

Lack of clarity and built-in commands for standardized, reliable image backup and restore causes user confusion and broken workflows. OCI image layout is not widely adopted by users. This pattern is reported repeatedly by users tracked in [GitHub Issue #1160](https://github.com/oras-project/oras/issues/1160), [GitHub Issue #1353](https://github.com/oras-project/oras/issues/1353), [GitHub Issue #1366](https://github.com/oras-project/oras/issues/1366).

## Existing Solutions or Expectations

* `docker save/load` supports exporting and importing images but not referrers or OCI artifacts.
* `oras pull/push` handles single artifacts, but not repository-level operations.
* It's inefficient to persist multiple artifacts in OCI layout format via `oras copy`.

This proposal meets user expectations of portability, structure, and artifact completeness using OCI specifications.

## Proposal

This document proposes two new command sets, `oras backup` and `oras restore`, to address the identified problems and support the scenarios outlined above. It also describes the desired user experience for backing up and restoring artifacts, images, and repositories between a registry and the local environment.

### New Command/Parameters in the CLI

#### Command: `oras backup`

**Short summary:**
Backup OCI artifacts and repositories from a registry into a structured, portable OCI image layout or archive tarball file locally.

**Syntax:**
```bash
oras backup [flags] <registry>/<repository>[:<ref1>[,<ref2>...]] [...]
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
```bash
oras restore [flags] <registry>/<repository>[:<ref1>[,<ref2>...]] [...]
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

```bash
oras backup registry-a.k8s.io/kube-apiserver:v1 --include-referrers --output airgap-snapshot.tar
```

Transfer the `.tar` file to the air-gapped system via a secured channel. Restore the tarball from local to another registry:

```bash
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

```bash
# Backup a repository from a registry to a local compressed tarball. All tags and their referrers will be included.
oras backup --output backup.tar --include-referrers registry-a.k8s.io/kube-apiserver
```

Transfer the backup file to new environment via secure channels (e.g., BitLocker-enabled removable drives)

Restore images and referrer artifacts from a local backup file to a target registry. All tags and their referrers will be included be default.

```bash
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

```bash
$ oras backup registry.k8s.io/kube-apiserver registry.k8s.io/kube-controller-manager --output k8s-control-plane
```

List the repositories in the OCI image layout

```console
$ oras repo list --oci-layout k8s-control-plane 
registry.k8s.io/kube-apiserver
registry.k8s.io/kube-controller-manager
```

Restore them to two repositories in a registry

```bash
$ oras restore localhost:5000/kube-apiserver localhost:5000/kube-controller --input k8s-control-plane
```

## Summary

The `oras backup` and `oras restore` commands introduce a structured, OCI-compliant way to persist and rehydrate artifacts and referrers, bridging a critical gap in the current functionality of the `oras` CLI. This enhancement empowers users with flexible, scriptable, and portable tooling for registry state management.