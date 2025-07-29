# Scenarios: Portable Backup and Restore of OCI Artifacts and Images

## Overview

Organizations rely on container images and other OCI artifacts to build, deploy, and operate their applications. These images and artifacts are built locally and stored in public or private OCI registries. However, as organizations mature their supply chain security, disaster recovery, and regulatory compliance, they face increasing demands to efficiently acquire, migrate, promote, mirror, and backup images and artifacts across registries and local environments, while preserving provenance, integrity, and metadata.

Today, fragmented tooling and manual scripts make these tasks complex, error-prone, and operationally expensive. Common tools like `docker save/load` and `oras pull/push`, `oras copy` only cover parts of the workflow, often lacking support for referrers, deduplication, and structured backups. This results in brittle processes, duplicated blobs, missing attestations, and frustrated developers.

This document describes the user scenarios and challenges faced by users managing images and OCI artifacts across registries and local environment. It proposes a unified, reliable, and portable solution built into the `oras` CLI to address these gaps. In particular, this document motivates the need for structured backup and restore workflows that simplify artifact movement, ensure completeness, and integrate seamlessly with security and compliance practices. The proposals and detailed CLI design are documented in the [Proposal: Portable Backup and Restore of OCI Artifacts, Images, and Repositories](./backup-restore.md).

## Problem Statement & Motivation

As organizations scale their software supply chain, acquiring and managing OCI artifacts is no longer as simple as pulling images from public registries e.g. Docker Hub or pushing them into a private registry. Security-conscious enterprises are imposing strict controls over how container images, Helm charts, AI models, SBOMs, attestations, and other OCI artifacts flow between registries and local environments.

Take for example, a global bank that cannot allow development teams to directly pull from public registries. Instead, they operate an internal acquisition pipeline where artifacts must first pass through vulnerability scans, software license checks, and supply chain attestation validation. Only after passing these gates are images and artifacts published to the bank's trusted registry for internal use. Similarly, security-critical systems maintain air-gapped environments. For them, acquiring artifacts requires carefully controlled offline transfers, with no room for manual errors or missing metadata.

Enterprises often maintain separate registries for development (DEV), quality assurance (QA), and production (PROD) to reduce the risk of untested artifacts reaching production. Promotion workflows rely on moving OCI artifacts across local environment and registries in a traceable, consistent, and secure manner.

Yet today, developers and users resort to fragmented, CLI tools like:

* `docker save/load` for container images.
* `oras pull/push` for OCI artifacts.
* `oras copy` for copying a single image with referrers.
* Ad-hoc scripts to cobble together backups and restore artifacts across different environments.

This approach brings significant limitations and problems:

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

Dave is a security engineer at a FinTech company that operates in a highly regulated environment. As part of their infrastructure security policies, the production environment is fully air-gapped to reduce the attack surface and comply with regulatory standards.

To deploy software to this air-gapped environment, Dave is responsible for preparing offline snapshots of all required container images and associated artifacts, including SBOMs, signatures, and attestations. These artifacts must be reviewed, signed, and bundled in a secure, verifiable manner before being transferred to production.

The process typically involves:

1. Packaging an image and all its linked referrers from a remote registry into a single `.tar` archive or directory using OCI image layout format locally.
2. Transferring the compressed snapshot over secure channels to the air-gapped network.
3. Restoring the image and referrer artifacts into an internal OCI-compliant registry for deployment.

Today, Dave has no native way to do this using `docker save/load`, `oras pull/push`, or other common tools. The lack of a unified snapshot solution means that artifacts must be copied manually or scripted with `oras copy`, often resulting in incomplete transfers (e.g., missing signatures or dependencies), blobs duplicated across files and no assurance of artifact integrity and completeness, difficulty validating artifact integrity upon restore. See an example GitHub issue [#730](https://github.com/oras-project/oras/issues/730) for details.

### Scenario 2: Image and Artifact Portability Across Isolated Environments

Cindy is a DevOps engineer working for a SaaS company that enforces strict network isolation between development, testing, staging, and production environments. Each environment has its own isolated OCI registry with no direct network connectivity between them. Cindy is responsible for promoting container images and artifacts across these isolated environments. For example, after building and testing an application image in the development environment, she needs to transfer it to the test and production environments.

However, direct registry-to-registry transfers are impossible due to network isolation and security policies. Today, Cindy uses `docker save/load`:

```console
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

### Scenario 3: Backup and Restore Repositories for Disaster Recovery and Compliance Audit

Alice is an infrastructure engineer at a multi-cloud SaaS company. She manages critical application repositories across multiple registries. These repositories contain not only images but also supply chain artifacts like SBOMs, signatures, and image lifecycle metadata.

Recently, a misconfigured cleanup job accidentally deleted an entire production repository. Although image builds could be recreated, many old tags still in use by customers were lost, along with their associated attestations. Old tags are particularly important for her team. For example, base images like `ubuntu` has multiple old tags `18.04`, `20.04`, etc, many enterprise customers still run old versions and those tags must remain accessible for hotfix builds, compliance audits, or rollback. The team spent hours manually recovering what they could using `docker` and `oras`, and scattered scripts, but couldn’t fully restore the original state.

To prevent future incidents and meet compliance audit requirements, Alice needs a unifed, portable solution to back up an entire repository including all tags and referrers into a single archive or directory. She also wants to restore it quickly and completely if something goes wrong.

With a holistic backup and restore solution in `oras`, Alice can capture all artifacts of a repository and recover it with confidence, improving operational resilience and audit readiness while eliminating complex manual work.

### Scenario 4: Uploading and Downloading Image With Referrers Using `oras pull/push` 

Bob, a developer maintaining containerized applications. Bob wants to create a backup of OCI images from the registry to local disk for disaster recovery, local modification, or air-gapped use. However, Bob incorrectly uses `oras pull` and `oras push`:

```console  
# Pull an image and save it locally as a tarball (incorrect usage)
oras pull foo.example.com/app/backend:v1.0.0 -o backend.tar

# Extract the tarball and modify it locally
tar -xf backend.tar

# Push the modified image back to the registry
oras push foo.example.com/app/backend:v1.0.1 ./extracted
```

At first glance, this appears to work. The image is pushed back to the registry. But when the image consumers try to pull and run the image, they encounter errors. The image referrers also lost when pulling and running the image. This is because `oras pull/push` only handles raw artifacts, not the full OCI image required for runnable images. Bob should be using `oras copy --recursive` with `--to-oci-layout` and `--from-oci-layout` to properly export and import an image with referrers in OCI image layout format:

```console
oras copy --recursive --to-oci-layout registry.example.com/app/backend:v1.0.0 ./image-backup:v1.0.0 
```

To restore from an OCI image layout to an image:

```console
oras copy --recursive --from-oci-layout ./image-backup:v1.0.0 registry.example.com/app/backend:v1.0.0
```

Lack of clarity and built-in commands for standardized, reliable image backup and restore causes user confusion and broken workflows. OCI image layout is not widely adopted by users. This pattern is reported repeatedly by users tracked in [GitHub Issue #1160](https://github.com/oras-project/oras/issues/1160), [GitHub Issue #1353](https://github.com/oras-project/oras/issues/1353), [GitHub Issue #1366](https://github.com/oras-project/oras/issues/1366).