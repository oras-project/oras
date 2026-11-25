# Proposal: Multi-Repository Backup and Restore

## Overview

Building upon the single-repository backup and restore functionality, this proposal extends `oras backup` and `oras restore` to support multi-repository operations across one or more registries. This enhancement enables users to backup artifacts from multiple repositories, in the same registry or in different registries, into a single OCI image layout, and restore them to one or more target registries with flexible repository mapping capabilities.

Multi-repository support addresses enterprise scenarios where users need to:
- Create unified backups spanning multiple related repositories (e.g., all microservices for an application)
- Package artifacts from several registries (e.g., Docker Hub, GHCR and a private registry) into one archive for air-gapped transfer ([#1893](https://github.com/oras-project/oras/issues/1893))
- Migrate selected repositories between environments
- Reorganize repository structures during restoration (e.g., consolidating scattered repos, changing naming conventions)
- Backup specific sets of repositories for disaster recovery

Every backup records the full reference (`<registry>/<repository>:<tag>`) of each artifact, so a backup always says where its contents came from. See [Backup Format and Annotations](#backup-format-and-annotations).

> [!NOTE]
> Backing up from multiple registries in a single operation depends on oras-go support for working with several registries in one command. See [Dependencies](#dependencies).

**Out of Scope:** Backing up an entire registry (all repositories) is not supported. For registry-wide backup and replication, use registry-native features (e.g., Harbor replication, ACR geo-replication) or vendor-specific backup tools. Multi-repository backup should be used for intentional, bounded operations where users specify exactly which repositories to backup.

This proposal keeps existing `oras backup` command lines working and keeps earlier backups restorable, while introducing intuitive, human-centric syntax for multi-repository workflows. `oras restore` changes how its target argument is interpreted (see [Backward Compatibility](#backward-compatibility)).

## Design Principles

1. **Backward Compatibility**: Existing `oras backup` command lines continue to work, and backups created by earlier versions can still be restored (see [Backward Compatibility](#backward-compatibility))
2. **Progressive Disclosure**: Simple cases remain simple; complexity is introduced only when needed
3. **Consistency**: Multi-repository syntax follows established patterns from single-repository design
4. **Flexibility**: Support for repository mapping and restructuring during restore
5. **Safety**: Clear validation and dry-run capabilities to prevent accidental operations

## Multi-Repository Backup

### Basic Syntax

Multi-repository backup accepts multiple repository references, from one or more registries, allowing users to backup artifacts from different repositories into a single OCI layout.

**Syntax:**
```bash
oras backup [flags] --output <path> <registry1>/<repo1>[:<tags>] <registry2>/<repo2>[:<tags>] ...
```

Where each repository reference follows the format:
```
<registry>/<repository>[:<tag1>[,<tag2>...]]
```

References may point to different registries. See [Authentication Handling](#authentication-handling) for how credentials and connection flags apply when more than one registry is involved.

### Behavior

- **Multiple registries**: Repository references may come from different registries
- **Full references**: Every tag is recorded in `index.json` as `<registry>/<repository>:<tag>`, so repositories with the same path on different registries do not collide in the backup
- **All tags**: If no tags specified for a repository, all tags in that repository are backed up
- **Selective tags**: Tags can be specified per repository using comma-separated list
- **Deduplication**: Blobs are deduplicated across all repositories and registries in the backup
- **Referrers**: When `--include-referrers` is used, referrers for all specified artifacts are included
- **Registry-specific flags**: Connection flags (TLS, headers, etc.) apply to every source registry. Credential flags are only accepted when all references share one registry (see [Authentication Handling](#authentication-handling))

### Examples

#### Example 1: Backup Multiple Repositories from Same Registry

Backup multiple utility images from Docker Hub:

```bash
oras backup --output base-images.tar \
  --include-referrers \
  docker.io/library/busybox:1.36.1 \
  docker.io/library/alpine:3.19.0 \
  docker.io/library/nginx:1.25-alpine
```

**Output:**
```console
Found 3 repository(ies) with 3 tag(s):
  - docker.io/library/busybox: 1.36.1
  - docker.io/library/alpine: 3.19.0
  - docker.io/library/nginx: 1.25-alpine

Backing up docker.io/library/busybox:1.36.1...
✓ Pulled application/vnd.oci.image.index.v1+json                            9.31/9.31 KB 100.00%  41ms
✓ Pulled application/vnd.oci.image.manifest.v1+json                           610/610  B 100.00%  38ms
✓ Pulled application/vnd.oci.image.config.v1+json                             459/459  B 100.00%   6ms
✓ Pulled application/vnd.oci.image.layer.v1.tar+gzip                        2.11/2.11 MB 100.00%    3s
Pulled tag 1.36.1 with 0 referrer(s)

Backing up docker.io/library/alpine:3.19.0...
✓ Pulled application/vnd.docker.distribution.manifest.list.v2+json            1.6/1.6 KB 100.00%  36ms
✓ Pulled application/vnd.docker.distribution.manifest.v2+json                 528/528  B 100.00%  41ms
✓ Pulled application/vnd.docker.container.image.v1+json                     1.44/1.44 KB 100.00%   6ms
✓ Pulled application/vnd.docker.image.rootfs.diff.tar.gzip                  3.19/3.19 MB 100.00%   11s
Pulled tag 3.19.0 with 0 referrer(s)

Backing up docker.io/library/nginx:1.25-alpine...
✓ Pulled application/vnd.docker.distribution.manifest.list.v2+json            1.8/1.8 KB 100.00%  38ms
✓ Pulled application/vnd.docker.distribution.manifest.v2+json                 1.5/1.5 KB 100.00%  42ms
✓ Pulled application/vnd.docker.container.image.v1+json                     8.14/8.14 KB 100.00%  12ms
✓ Pulled application/vnd.docker.image.rootfs.diff.tar.gzip                  9.56/9.56 MB 100.00%   18s
Pulled tag 1.25-alpine with 0 referrer(s)

Exporting to base-images.tar
Exported to base-images.tar (42.8 MB)
Successfully backed up 3 tag(s) from 3 repository(ies) in 52s.
```

#### Example 2: Backup Specific Tags from Multiple Repositories

Backup specific version tags from different base images:

```bash
oras backup --output versioned-bases.tar \
  --include-referrers \
  docker.io/library/alpine:3.18.0,3.19.0 \
  docker.io/library/busybox:1.35.0,1.36.1 \
  docker.io/library/redis:7.2-alpine
```

**Output:**
```console
Found 3 repository(ies) with 5 tag(s):
  - docker.io/library/alpine: 3.18.0, 3.19.0
  - docker.io/library/busybox: 1.35.0, 1.36.1
  - docker.io/library/redis: 7.2-alpine

Backing up artifacts...
✓ Pulled docker.io/library/alpine:3.18.0 with 0 referrer(s)
✓ Pulled docker.io/library/alpine:3.19.0 with 0 referrer(s)
✓ Pulled docker.io/library/busybox:1.35.0 with 0 referrer(s)
✓ Pulled docker.io/library/busybox:1.36.1 with 0 referrer(s)
✓ Pulled docker.io/library/redis:7.2-alpine with 0 referrer(s)

Exporting to versioned-bases.tar
Exported to versioned-bases.tar (98.5 MB)
Successfully backed up 5 tag(s) from 3 repository(ies) in 1m 15s.
```

#### Example 3: Backup from Multiple Registries

Backup an application's dependencies from Docker Hub, GHCR and a private registry into one archive:

```bash
oras backup --output app-deps.tar \
  docker.io/library/nginx:1.25-alpine \
  docker.io/library/redis:7.2-alpine \
  ghcr.io/oras-project/oras:v1.1.0 \
  registry.mycompany.com/apps/frontend:v2.0
```

**Output:**
```console
Found 4 repository(ies) from 3 registry(ies) with 4 tag(s):
  docker.io
    - library/nginx: 1.25-alpine
    - library/redis: 7.2-alpine
  ghcr.io
    - oras-project/oras: v1.1.0
  registry.mycompany.com
    - apps/frontend: v2.0

Backing up artifacts...
✓ Pulled docker.io/library/nginx:1.25-alpine with 0 referrer(s)
✓ Pulled docker.io/library/redis:7.2-alpine with 0 referrer(s)
✓ Pulled ghcr.io/oras-project/oras:v1.1.0 with 0 referrer(s)
✓ Pulled registry.mycompany.com/apps/frontend:v2.0 with 0 referrer(s)

Exporting to app-deps.tar
Exported to app-deps.tar (87.3 MB)
Successfully backed up 4 tag(s) from 4 repository(ies) across 3 registry(ies) in 48s.
```

## Multi-Repository Restore

### Basic Syntax

Multi-repository restore supports several modes to accommodate different use cases:

**Mode 1: Direct Restore (Same Structure)**
```bash
oras restore [flags] --input <path> <target-registry>[/<namespace>]
```

**Mode 2: Selective Restore**
```bash
oras restore [flags] --input <path> <target-repo1> <target-repo2> [<target-repo3>...]
```

**Mode 3: Restore with Rules**
```bash
oras restore [flags] --input <path> --rules <rules-file>
```

### Behavior

- **All repositories**: If only a target registry (optionally with a namespace) is specified, all repositories from the backup are restored under it
- **Selective repositories**: Specific repository destinations can be provided
- **Repository mapping**: Complex mappings via rules file
- **Name preservation**: By default, repository paths and tags from the backup are preserved; only the source registry host is replaced by the target
- **Flexible targets**: Can restore to different registries than original backup source
- **Consistent targets**: A single positional argument is always `<target-registry>[/<namespace>]`, however many repositories the backup holds. A backup containing only `docker.io/library/redis` restores to `<target>/library/redis`, the same place it would go in a multi-repository backup. To restore under a different repository name, use a rules file (Mode 3); simpler renaming may be added later
- **Collision detection**: Before anything is pushed, restore checks that no two source repositories map to the same target repository (see [Restoring Backups from Multiple Registries](#restoring-backups-from-multiple-registries))

### Restore Modes

#### Mode 1: Direct Restore to Target Registry

Restore all backed-up repositories to a target registry, preserving original repository paths:

```bash
oras restore --input base-images.tar registry.mycompany.com
```

**Behavior:**
- `docker.io/library/busybox` → `registry.mycompany.com/library/busybox`
- `docker.io/library/alpine` → `registry.mycompany.com/library/alpine`
- `docker.io/library/nginx` → `registry.mycompany.com/library/nginx`

**Output:**
```console
Loaded backup archive: base-images.tar (42.8 MB)
Found 3 repository(ies) with 3 tag(s):
  - docker.io/library/busybox: 1.36.1
  - docker.io/library/alpine: 3.19.0
  - docker.io/library/nginx: 1.25-alpine

Restoring to registry.mycompany.com...
✓ Pushed application/vnd.oci.image.layer.v1.tar+gzip                        2.11/2.11 MB 100.00%    4s
✓ Pushed application/vnd.oci.image.manifest.v1+json                           610/610  B 100.00%  52ms
Restored registry.mycompany.com/library/busybox:1.36.1 with 0 referrer(s)

✓ Pushed application/vnd.docker.image.rootfs.diff.tar.gzip                  3.19/3.19 MB 100.00%    6s
✓ Pushed application/vnd.docker.distribution.manifest.v2+json                 528/528  B 100.00%  48ms
Restored registry.mycompany.com/library/alpine:3.19.0 with 0 referrer(s)

✓ Pushed application/vnd.docker.image.rootfs.diff.tar.gzip                  9.56/9.56 MB 100.00%   14s
✓ Pushed application/vnd.docker.distribution.manifest.v2+json                 1.5/1.5 KB 100.00%  51ms
Restored registry.mycompany.com/library/nginx:1.25-alpine with 0 referrer(s)

Successfully restored 3 tag(s) from 3 repository(ies) to registry.mycompany.com in 28s.
```

A namespace can be added to the target, in which case repository paths are placed under it:

```bash
oras restore --input base-images.tar registry.mycompany.com/mirror
```

- `docker.io/library/busybox` → `registry.mycompany.com/mirror/library/busybox`

#### Mode 2: Selective Repository Restore

Restore only specific repositories from backup. Specify target repositories to filter which ones to restore. Repository paths from the backup are preserved.

```bash
oras restore --input base-images.tar \
  registry.mycompany.com/library/alpine \
  registry.mycompany.com/library/busybox
```

**Behavior:**
- Filters repositories from backup based on command-line targets
- Repository structure is **preserved** (e.g., `docker.io/library/alpine` → `registry.mycompany.com/library/alpine`)
- Only the registry hostname changes, paths remain the same
- A target selects the backed-up repository with the same path. If that path exists under more than one source registry in the backup (e.g., `docker.io/library/nginx` and `mirror.gcr.io/library/nginx`), restore fails and asks for a rules file (Mode 3) to pick the source

**Output:**
```console
Loaded backup archive: base-images.tar (42.8 MB)
Found 3 repository(ies) in backup, restoring 2 repository(ies):
  ✓ docker.io/library/alpine → registry.mycompany.com/library/alpine
  ✓ docker.io/library/busybox → registry.mycompany.com/library/busybox
  - docker.io/library/nginx (skipped)

Restoring selected repositories...
✓ Restored registry.mycompany.com/library/alpine:3.19.0 with 0 referrer(s)
✓ Restored registry.mycompany.com/library/busybox:1.36.1 with 0 referrer(s)

Successfully restored 2 tag(s) from 2 repository(ies) in 12s.
Note: 1 repository(ies) in backup were not restored.
```

> [!TIP]
> To rename or restructure repositories during restore, use Mode 3 with a rules file.

#### Mode 3: Restore with Repository Mapping (Advanced)

For **advanced scenarios** requiring repository renaming or restructuring, use a rules file. This mode is mutually exclusive with positional repository arguments.

> [!IMPORTANT]
> The `--rules` flag cannot be used together with positional repository arguments. Use either:
> - Positional args for direct/selective restore (Mode 1 & 2)
> - Rules file for complex remapping scenarios (Mode 3)

**Rules File (`restore-rules.json`):**
```jsonc
{
  "mappings": [
    // Rename and relocate busybox
    {
      "source": "docker.io/library/busybox",
      "target": "registry.mycompany.com/base/busybox-custom"
    },
    // Change path structure for alpine
    {
      "source": "docker.io/library/alpine",
      "target": "registry.mycompany.com/images/alpine"
    },
    // Selective tags with mapping
    {
      "source": "docker.io/library/nginx",
      "target": "registry.mycompany.com/webservers/nginx",
      "tags": ["1.25-alpine"]
    }
  ]
}
```

**Command:**
```bash
oras restore --input base-images.tar --rules restore-rules.json
```

**Output:**
```console
Loaded backup archive: base-images.tar (42.8 MB)
Loaded rules configuration: restore-rules.json
Found 3 repository(ies) with 3 tag(s) in backup
Applying 3 mapping rule(s)

Restoring with mappings...
✓ docker.io/library/busybox → registry.mycompany.com/base/busybox-custom (1 tag)
✓ docker.io/library/alpine → registry.mycompany.com/images/alpine (1 tag)
✓ docker.io/library/nginx → registry.mycompany.com/webservers/nginx (1 tag)

Successfully restored 3 tag(s) from 3 repository(ies) with remapping in 22s.
```

### Restoring Backups from Multiple Registries

When a backup contains repositories from more than one registry, Mode 1 and Mode 2 drop each source registry host and keep the repository path:

```bash
oras restore --input app-deps.tar airgap.internal.company
```

- `docker.io/library/nginx` → `airgap.internal.company/library/nginx`
- `ghcr.io/oras-project/oras` → `airgap.internal.company/oras-project/oras`
- `registry.mycompany.com/apps/frontend` → `airgap.internal.company/apps/frontend`

Docker Hub repositories keep their `library/` prefix; restore does not apply Docker Hub's short-name normalization to target names.

#### Collisions

Dropping the source registry can map two source repositories to the same target, for example `docker.io/library/nginx` and `mirror.gcr.io/library/nginx`. Restore detects this before pushing anything and fails:

```console
Error: 2 source repositories map to the same target repository:
  docker.io/library/nginx     → airgap.internal.company/library/nginx
  mirror.gcr.io/library/nginx → airgap.internal.company/library/nginx

Recommendations:
  1. Use --preserve-source-registry to keep the source registry in the target path
  2. Use --rules to map each source repository to a distinct target
```

#### Keeping the Source Registry in the Target Path

`--preserve-source-registry` keeps the source registry host as the first path component of each target repository, which avoids collisions:

```bash
oras restore --input app-deps.tar --preserve-source-registry airgap.internal.company
```

- `docker.io/library/nginx` → `airgap.internal.company/docker.io/library/nginx`
- `ghcr.io/oras-project/oras` → `airgap.internal.company/ghcr.io/oras-project/oras`

The flag applies to Mode 1 and Mode 2. It cannot be combined with `--rules`, where targets are given in full.

### Selective Tag Restoration

Restore specific tags from specific repositories:

```bash
oras restore --input base-images.tar \
  registry.mycompany.com/library/alpine:3.19.0 \
  registry.mycompany.com/library/busybox:1.36.1
```

**Output:**
```console
Loaded backup archive: base-images.tar (42.8 MB)
Restoring selected tags from 2 repository(ies)

✓ Restored registry.mycompany.com/library/alpine:3.19.0 with 0 referrer(s)
✓ Restored registry.mycompany.com/library/busybox:1.36.1 with 0 referrer(s)

Successfully restored 2 tag(s) from 2 repository(ies) in 10s.
```

### Dry-Run Mode

Preview restore operations without making changes:

```bash
oras restore --input base-images.tar registry.mycompany.com --dry-run
```

**Output:**
```console
Loaded backup archive: base-images.tar (42.8 MB)
Found 3 repository(ies) with 3 tag(s)

[DRY RUN] The following operations would be performed:

Registry: registry.mycompany.com
  Repository: library/busybox (from docker.io/library/busybox)
    ✓ 1.36.1 (sha256:355b3a1...) + 0 referrer(s)
  
  Repository: library/alpine (from docker.io/library/alpine)
    ✓ 3.19.0 (sha256:51b6726...) + 0 referrer(s)
  
  Repository: library/nginx (from docker.io/library/nginx)
    ✓ 1.25-alpine (sha256:a5660da...) + 0 referrer(s)

[DRY RUN] Would restore 3 tag(s) from 3 repository(ies)
[DRY RUN] Total artifacts: 3 (including referrers)
[DRY RUN] Estimated upload size: 42.8 MB

No changes were made. Remove --dry-run to perform actual restore.
```

Dry-run also reports [collisions](#collisions), so a mapping can be checked before any push.

## Inspecting Backup Contents

Backups are saved in standard OCI Image Layout format, allowing inspection using ORAS commands.

> [!NOTE]
> This proposal depends on [#1770](https://github.com/oras-project/oras/issues/1770) which adds `oras repo ls --oci-layout-path` support.

### List Repositories in Backup

```bash
oras repo ls --oci-layout-path base-images.tar
```

**Output:**
```console
docker.io/library/alpine
docker.io/library/busybox
docker.io/library/nginx
```

Or with a namespace filter:

```bash
oras repo ls --oci-layout-path base-images.tar docker.io/library
```

**Output:**
```console
alpine
busybox
nginx
```

### List Tags in a Repository

```bash
oras repo tags --oci-layout-path base-images.tar docker.io/library/busybox
```

**Output:**
```console
1.36.1
```

### Show Manifest Details

```bash
oras manifest fetch --oci-layout-path base-images.tar docker.io/library/busybox:1.36.1
```

### Discover Referrers

```bash
oras discover --oci-layout-path base-images.tar docker.io/library/busybox:1.36.1
```

**Note:** All ORAS commands supporting `--oci-layout-path` can inspect backup contents with repository paths. Repositories from different registries are listed with their own registry host.

> [!IMPORTANT]
> Use `--oci-layout-path <path> <reference>` to address artifacts in a backup. The `--oci-layout <path>:<tag>` form splits on the last `:`, so it cannot address a full reference such as `docker.io/library/alpine:latest`, and `--oci-layout backup:latest` no longer resolves because no entry is stored under the bare tag.

## Dependencies

This proposal depends on the following:

- **[#1770](https://github.com/oras-project/oras/issues/1770)**: Support `oras repo ls --oci-layout-path` for listing repositories in OCI Image Layout backups.
- **oras-go**: Support for working with repositories on several registries within one command, each with its own credentials. Multi-registry backup cannot ship until this is available. (Issue to be linked.)

## Backup Format and Annotations

Backups are stored in standard OCI Image Layout format with full reference information preserved in annotations.

### Reference Annotation (`org.opencontainers.image.ref.name`)

The `org.opencontainers.image.ref.name` annotation in `index.json` always stores the full reference of the artifact, `<registry>/<repository>:<tag>` (e.g., `"docker.io/library/alpine:latest"`). This applies to single-repository, multi-repository and multi-registry backups alike. There is no tag-only mode and no flag to select one.

This approach:
- Aligns with the [OCI Image Spec](https://github.com/opencontainers/image-spec/blob/v1.1.1/annotations.md#pre-defined-annotation-keys), which allows both tag-only and full reference formats in `org.opencontainers.image.ref.name`
- Enables automation tools (e.g., air-gap packaging tools like [Zarf](https://github.com/zarf-dev/zarf)) to programmatically determine the original location of backed-up artifacts
- Addresses use cases described in [#1893](https://github.com/oras-project/oras/issues/1893)
- Disambiguates artifacts from different repositories and registries
- Keeps one backup format, so tools reading backups do not need to handle two layouts for new backups

The value is not limited in length by the OCI Image Spec. In practice it is bounded by the distribution-spec limits on its parts: 255 characters for the registry and repository name and 128 characters for a tag.

> [!WARNING]
> Backups created by this version **cannot be restored** by earlier ORAS CLI versions, which treat the annotation value as a tag. Because `oras backup` and `oras restore` are experimental, this format change is made without an opt-in flag.

**Example `index.json` formats:**

**Single-repository backup:**
```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.index.v1+json",
      "digest": "sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412",
      "size": 9218,
      "annotations": {
        "org.opencontainers.image.ref.name": "docker.io/library/alpine:latest"
      }
    }
  ]
}
```

**Multi-registry backup:**

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.index.v1+json",
      "digest": "sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412",
      "size": 9218,
      "annotations": {
        "org.opencontainers.image.ref.name": "docker.io/library/alpine:latest"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.index.v1+json",
      "digest": "sha256:355b3a1c6b8f2c7f0a1b5e8c3d9f1e2a4b6c8d0e2f4a6b8c0e2f4a6b8c0e2f4a",
      "size": 7892,
      "annotations": {
        "org.opencontainers.image.ref.name": "ghcr.io/oras-project/oras:v1.1.0"
      }
    }
  ]
}
```

#### Restore Behavior

`oras restore` parses each `org.opencontainers.image.ref.name` value with the ORAS reference parser:

- **Full reference** (all backups created by this version, e.g., `"docker.io/library/alpine:latest"`): The registry, repository and tag are extracted. The registry and repository identify the source for mapping; the tag is used as the destination tag.
- **Tag-only** (backups created by earlier versions, e.g., `"latest"`): Treated as a tag in a single repository with no recorded source. There is no repository path to keep, so the positional argument is the target repository, as today.

**Example:**
```bash
oras backup --output backup.tar docker.io/library/alpine:latest

# Restore to a different registry; the repository path is kept
oras restore --input backup.tar registry.mycompany.com
# Result: docker.io/library/alpine:latest → registry.mycompany.com/library/alpine:latest

# Restore under a namespace
oras restore --input backup.tar registry.mycompany.com/mirror
# Result: docker.io/library/alpine:latest → registry.mycompany.com/mirror/library/alpine:latest
```

> [!NOTE]
> The full reference format in `org.opencontainers.image.ref.name` conforms to the OCI spec's reference grammar: `component ("/" component)*` where each component matches `alphanum (separator alphanum)*` with character set `A-Za-z0-9` and separators `-._:@/+`.

## Rules File Format

The rules file uses JSON format for defining complex restore scenarios:

```jsonc
{
  // Global settings (optional)
  "settings": {
    "exclude-referrers": false,  // Exclude referrers during restore
    "fail-fast": true            // Fail on first error vs continue
  },
  "mappings": [
    // Example 1: Simple repository rename
    {
      "source": "registry-a.mycompany.com/old/name",
      "target": "registry-b.mycompany.com/new/name"
    },
    // Example 2: Selective tag restore
    {
      "source": "registry-a.mycompany.com/app",
      "target": "registry-b.mycompany.com/app",
      "tags": ["v1.0", "v1.1", "latest"]
    },
    // Example 3: Exclude referrers for specific repo
    {
      "source": "registry-a.com/untrusted",
      "target": "registry-b.com/untrusted",
      "exclude-referrers": true
    },
    // Example 4: Repository consolidation
    {
      "source": "registry-a.com/microservices/service-1",
      "target": "registry-b.com/unified/services"
    },
    {
      "source": "registry-a.com/microservices/service-2",
      "target": "registry-b.com/unified/services"
    },
    {
      "source": "registry-a.com/microservices/service-3",
      "target": "registry-b.com/unified/services"
    },
    // Example 5: Tag renaming during restore
    {
      "source": "registry-a.com/app",
      "target": "registry-b.com/app",
      "tag-mappings": {
        "latest": "stable",
        "v2.0-rc1": "v2.0"
      }
    }
  ]
}
```

Each `source` is a full repository reference including its registry, so rules can tell apart repositories with the same path on different registries.

When several sources map to the same target repository (as in Example 4), their tags must not overlap. If two sources would write the same tag to the same target, restore fails before pushing anything and names the conflicting tag.

## User Experience Enhancements

### Progress Reporting

For multi-repository operations, progress is organized by repository. Each repository is shown with its full name, including the registry, so multi-registry backups read the same way:

```console
Backing up 3 repositories...

[1/3] docker.io/library/busybox
  ✓ 1.36.1 (3/3 artifacts, 4.2 MB)
  Repository complete: 3 artifacts, 4.2 MB

[2/3] docker.io/library/alpine
  ✓ 3.19.0 (3/3 artifacts, 7.8 MB)
  Repository complete: 3 artifacts, 7.8 MB

[3/3] docker.io/library/nginx
  ✓ 1.25-alpine (4/4 artifacts, 30.8 MB)
  Repository complete: 4 artifacts, 30.8 MB

Exported to base-images.tar (42.8 MB)
Successfully backed up 3 repositories with 3 tags in 52s
```

### Error Handling

Clear error messages with recovery suggestions:

```console
Error: Failed to backup ghcr.io/myorg/private-tool
  Reason: Authentication failed
  
Partial backup created: 2 of 3 repositories backed up successfully
  ✓ docker.io/library/busybox (1 tag)
  ✓ docker.io/library/alpine (1 tag)
  ✗ ghcr.io/myorg/private-tool (authentication required)

Recommendations:
  1. Check credentials for ghcr.io
  2. Run "oras login ghcr.io" to store credentials for that registry
```

## Validation and Safety

### Command Validation

The restore command validates arguments to prevent conflicting options:

```bash
# ERROR: Cannot use --rules with positional repository args
oras restore --input backup.tar --rules rules.json registry.io/repo
```

**Output:**
```console
Error: --rules cannot be used with positional repository arguments.

Use either:
  - Positional args for direct/selective restore:
      oras restore --input backup.tar registry.io/repo1 registry.io/repo2
  
  - Rules file for repository mapping:
      oras restore --input backup.tar --rules rules.json
```

Other validation errors, all raised before any artifact is copied:

- **Credential flags with multiple registries**: `--username`, `--password`, `--password-stdin` or `--identity-token` used with references from more than one registry (see [Authentication Handling](#authentication-handling))
- **Target collisions**: two source repositories map to the same target repository (see [Collisions](#collisions))
- **Tag conflicts**: two sources write the same tag to the same target repository
- **Conflicting options**: `--preserve-source-registry` used together with `--rules`

## Authentication Handling

Multi-repository operations use the same authentication mechanisms as single-repository commands.

**Single registry:** When all references come from one registry, all registry-specific flags (`--username`, `--password`, `--plain-http`, `--insecure`, `--ca-file`, etc.) work as they do today.

**Multiple registries:** Credentials are looked up per registry from the credential store, the same way other ORAS commands do. Log in to each registry beforehand with `oras login`, or point to a config file with `--registry-config`.

- **Credential flags** (`--username`, `--password`, `--password-stdin`, `--identity-token`): rejected when the references span more than one registry, because it would be unclear which registry they belong to.
- **Connection flags** (`--plain-http`, `--insecure`, `--ca-file`, `--cert-file`, `--key-file`, `-H`/`--header`, `--resolve`): apply to every source registry. Per-registry connection settings are out of scope for this proposal; split the backup if registries need different settings.
- **Restore** pushes to a single target registry in Modes 1 and 2, so all flags behave as they do today. When a rules file maps to more than one target registry, the same rules as for multi-registry backup apply.

**Example:**
```bash
# Store credentials for each source registry
oras login ghcr.io
oras login registry.mycompany.com

# Backup from Docker Hub, GHCR and a private registry in one operation
oras backup --output app-deps.tar \
  docker.io/library/nginx:1.25-alpine \
  ghcr.io/myorg/private-tool:v1.0 \
  registry.mycompany.com/app/frontend:v2.0
```

## Advanced Use Cases

### Use Case 1: Registry Migration

Migrate specific repositories between registries:

```bash
# Step 1: Backup from Docker Hub
oras backup --output migration.tar \
  --include-referrers \
  docker.io/library/busybox:1.36.1 \
  docker.io/library/alpine:3.19.0

# Step 2: Restore to private registry
oras restore --input migration.tar \
  registry.mycompany.com/base-images
```

### Use Case 2: Multi-Environment Promotion

Promote tested artifacts from staging to production:

```bash
# Backup specific versions from staging
oras backup --output staging-promotion.tar \
  --include-referrers \
  staging.mycompany.com/apps/frontend:v2.0 \
  staging.mycompany.com/apps/backend:v2.0 \
  staging.mycompany.com/apps/cache:v2.0

# Restore to production with renaming
oras restore --input staging-promotion.tar --rules promotion-rules.json
```

**promotion-rules.json:**
```jsonc
{
  "mappings": [
    {
      "source": "staging.mycompany.com/apps/frontend",
      "target": "production.mycompany.com/apps/frontend",
      "tag-mappings": {
        "v2.0": "latest"
      }
    },
    {
      "source": "staging.mycompany.com/apps/backend",
      "target": "production.mycompany.com/apps/backend",
      "tag-mappings": {
        "v2.0": "latest"
      }
    },
    {
      "source": "staging.mycompany.com/apps/cache",
      "target": "production.mycompany.com/apps/cache",
      "tag-mappings": {
        "v2.0": "stable"
      }
    }
  ]
}
```

### Use Case 3: Disaster Recovery

Create comprehensive backups with scheduled automation:

```bash
# Daily backup script for critical repositories
DATE=$(date +%Y%m%d)
oras backup --output /backups/critical-apps-$DATE.tar \
  --include-referrers \
  registry.mycompany.com/production/frontend \
  registry.mycompany.com/production/backend \
  registry.mycompany.com/production/database

# Retention: Keep last 7 days
find /backups -name "critical-apps-*.tar" -mtime +7 -delete

# After incident, restore specific version
# Target the registry: repository paths such as production/frontend are kept
oras restore --input /backups/critical-apps-20251120.tar \
  registry.mycompany.com \
  --dry-run  # Verify first

# Then actual restore
oras restore --input /backups/critical-apps-20251120.tar \
  registry.mycompany.com
```

### Use Case 4: Repository Consolidation

Consolidate multiple legacy repositories:

```bash
# Backup scattered repositories
oras backup --output legacy-apps.tar \
  legacy-registry.mycompany.com/team-a/app-1 \
  legacy-registry.mycompany.com/team-b/app-2 \
  legacy-registry.mycompany.com/team-c/app-3 \
  legacy-registry.mycompany.com/shared/utils

# Restore to unified structure
oras restore --input legacy-apps.tar --rules consolidation-rules.json
```

**consolidation-rules.json:**
```jsonc
{
  "mappings": [
    {
      "source": "legacy-registry.mycompany.com/team-a/app-1",
      "target": "registry.mycompany.com/services/app-1"
    },
    {
      "source": "legacy-registry.mycompany.com/team-b/app-2",
      "target": "registry.mycompany.com/services/app-2"
    },
    {
      "source": "legacy-registry.mycompany.com/team-c/app-3",
      "target": "registry.mycompany.com/services/app-3"
    },
    {
      "source": "legacy-registry.mycompany.com/shared/utils",
      "target": "registry.mycompany.com/libs/common"
    }
  ]
}
```

### Use Case 5: Air-Gapped Deployment

Transfer application stack to air-gapped environment:

```bash
# On connected system: Backup from company registry
oras backup --output application-stack.tar \
  --include-referrers \
  registry.mycompany.com/apps/frontend:v2.0 \
  registry.mycompany.com/apps/backend:v2.0 \
  registry.mycompany.com/apps/database:v2.0

# Transfer application-stack.tar via secure channel

# On air-gapped system: Restore to internal registry
oras restore --input application-stack.tar \
  airgap.internal.company/production
```

**For dependencies from multiple registries:**

Back up everything into one archive, then restore it to a single internal registry:

```bash
# On connected system: one backup from Docker Hub, GHCR and the company registry
oras backup --output application-stack.tar \
  --include-referrers \
  docker.io/library/nginx:1.25-alpine \
  docker.io/library/redis:7.2-alpine \
  ghcr.io/oras-project/oras:v1.1.0 \
  registry.mycompany.com/apps/frontend:v2.0

# On air-gapped system: restore all repositories, keeping the source registry in the path
oras restore --input application-stack.tar \
  --preserve-source-registry \
  airgap.internal.company
# docker.io/library/nginx              → airgap.internal.company/docker.io/library/nginx
# ghcr.io/oras-project/oras            → airgap.internal.company/ghcr.io/oras-project/oras
# registry.mycompany.com/apps/frontend → airgap.internal.company/registry.mycompany.com/apps/frontend
```

Tools that package images for air-gapped environments can read the full reference of each artifact from `index.json` to find where it came from.

## Flag Reference

### Backup Flags (New)

- `--dry-run`: Preview backup operation without downloading artifacts

### Restore Flags (New)

- `--rules <file>`: Rules configuration file for repository mapping
- `--preserve-source-registry`: Keep the source registry host as the first path component of each target repository (Modes 1 and 2). Avoids collisions when restoring a multi-registry backup to one registry. Cannot be used with `--rules`.

### Common Flags (Enhanced)

All existing authentication, TLS, and connection flags continue to work with multi-repository operations. With references from more than one registry, credential flags are rejected and connection flags apply to every registry (see [Authentication Handling](#authentication-handling)).

## Backward Compatibility

Existing single-repository backup command lines continue to work:

```bash
oras backup --output backup.tar registry.com/repo:tag
oras backup --output backup.tar registry.com/repo
```

**Restore target:** `oras restore` always treats a single positional argument as `<target-registry>[/<namespace>]` and keeps the source repository path under it. This changes existing restore command lines:

```bash
oras backup --output backup.tar registry.com/team/repo:tag
oras restore --input backup.tar target.com/team/repo
# Before: registry.com/team/repo:tag → target.com/team/repo:tag
# Now:    registry.com/team/repo:tag → target.com/team/repo/team/repo:tag

oras restore --input backup.tar target.com
# Now:    registry.com/team/repo:tag → target.com/team/repo:tag
```

Tag-only backups created by earlier versions are the exception: they record no repository path, so the argument remains the target repository.

**Backup format:** `oras backup` and `oras restore` are experimental, and this proposal changes the backup format without an opt-in flag:

- **New backups, new client**: Supported.
- **Old (tag-only) backups, new client**: Supported. Restore treats a tag-only `ref.name` as a tag and requires an explicit target repository, as today.
- **New backups, old client**: Not supported. Earlier versions use the `ref.name` value directly as the destination tag, which fails for a full reference. Users must upgrade the client that restores.
- **Other ORAS commands on new backups**: Address artifacts with `--oci-layout-path <path> <full-reference>`; `--oci-layout <path>:<tag>` no longer resolves (see [Inspecting Backup Contents](#inspecting-backup-contents)).

The release notes for the version that ships this change must call out the format change and the new interpretation of the restore target.

## Summary

This proposal extends `oras backup` and `oras restore` with multi-repository support across one or more registries. The design focuses on intentional, bounded backup operations where users explicitly specify which repositories to include.

Key features:
- **Multiple repositories in single backup**: Backup from multiple explicitly-specified repos, in one registry or several
- **Full references**: Every backup records `<registry>/<repository>:<tag>` for each artifact
- **Flexible restore options**: Direct, selective, or rules-based restoration
- **Repository mapping**: Powerful remapping capabilities for restructuring during restore, with collision detection
- **Safety features**: Dry-run preview and clear validation
- **Enterprise-ready**: Credential store support per registry, error recovery, and progress reporting
- **Consistent restore targets**: Repository paths are always kept under the restore target; existing backup command lines are unchanged and existing backups remain restorable

**Out of Scope:**
- **Entire registry backup**: Not supported. Use registry-native replication/backup features for registry-wide operations.
- **Pattern-based selection**: Not included in this proposal. May be considered in future iterations if customer demand exists.
- **Per-registry connection settings**: Connection flags apply to all source registries. Split the backup if registries need different TLS or header settings.

