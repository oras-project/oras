# Proposal: Multi-Repository Backup and Restore

## Overview

Building upon the single-repository backup and restore functionality, this proposal extends `oras backup` and `oras restore` to support multi-repository operations within a single registry. This enhancement enables users to backup artifacts from multiple repositories in the same registry into a single OCI image layout, and restore them to one or more target registries with flexible repository mapping capabilities.

Multi-repository support addresses enterprise scenarios where users need to:
- Create unified backups spanning multiple related repositories (e.g., all microservices for an application in the same registry)
- Migrate selected repositories between environments
- Reorganize repository structures during restoration (e.g., consolidating scattered repos, changing naming conventions)
- Backup specific sets of repositories for disaster recovery

> [!NOTE]
> For backing up artifacts from multiple registries, it is recommended to create separate backups for each registry. This simplifies authentication handling and allows registry-specific configuration flags to be used.

**Out of Scope:** Backing up an entire registry (all repositories) is not supported. For registry-wide backup and replication, use registry-native features (e.g., Harbor replication, ACR geo-replication) or vendor-specific backup tools. Multi-repository backup should be used for intentional, bounded operations where users specify exactly which repositories to backup.

This proposal maintains full backward compatibility with single-repository operations while introducing intuitive, human-centric syntax for multi-repository workflows.

## Design Principles

1. **Backward Compatibility**: All existing single-repository commands continue to work unchanged
2. **Progressive Disclosure**: Simple cases remain simple; complexity is introduced only when needed
3. **Consistency**: Multi-repository syntax follows established patterns from single-repository design
4. **Flexibility**: Support for repository mapping and restructuring during restore
5. **Safety**: Clear validation and dry-run capabilities to prevent accidental operations

## Multi-Repository Backup

### Basic Syntax

Multi-repository backup accepts multiple repository references from the same registry, allowing users to backup artifacts from different repositories into a single OCI layout.

**Syntax:**
```bash
oras backup [flags] --output <path> <registry>/<repo1>[:<tags>] <registry>/<repo2>[:<tags>] ...
```

Where each repository reference follows the format:
```
<registry>/<repository>[:<tag1>[,<tag2>...]]
```

**Important:** All repository references must be from the same registry. For backing up from multiple registries, create separate backups for each registry.

### Behavior

- **Same registry**: All repository references must be from the same registry
- **All tags**: If no tags specified for a repository, all tags in that repository are backed up
- **Selective tags**: Tags can be specified per repository using comma-separated list
- **Deduplication**: Blobs are deduplicated across all repositories in the backup
- **Referrers**: When `--include-referrers` is used, referrers for all specified artifacts are included
- **Registry-specific flags**: All registry-related flags (authentication, TLS, etc.) apply to the single target registry

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

## Multi-Repository Restore

### Basic Syntax

Multi-repository restore supports several modes to accommodate different use cases:

**Mode 1: Direct Restore (Same Structure)**
```bash
oras restore [flags] --input <path> <target-registry>
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

- **All repositories**: If only target registry specified, all repositories from backup are restored to that registry
- **Selective repositories**: Specific repository destinations can be provided
- **Repository mapping**: Complex mappings via rules file
- **Name preservation**: By default, repository and tag names from backup are preserved
- **Flexible targets**: Can restore to different registries than original backup source

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

### Selective Tag Restoration

Restore specific tags from specific repositories:

```bash
oras restore --input base-images.tar \
  registry.mycompany.com/base/alpine:3.19.0 \
  registry.mycompany.com/base/busybox:1.36.1
```

**Output:**
```console
Loaded backup archive: base-images.tar (42.8 MB)
Restoring selected tags from 2 repository(ies)

✓ Restored registry.mycompany.com/base/alpine:3.19.0 with 0 referrer(s)
✓ Restored registry.mycompany.com/base/busybox:1.36.1 with 0 referrer(s)

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
  Repository: library/busybox
    ✓ 1.36.1 (sha256:355b3a1...) + 0 referrer(s)
  
  Repository: library/alpine
    ✓ 3.19.0 (sha256:51b6726...) + 0 referrer(s)
  
  Repository: library/nginx
    ✓ 1.25-alpine (sha256:a5660da...) + 0 referrer(s)

[DRY RUN] Would restore 3 tag(s) from 3 repository(ies)
[DRY RUN] Total artifacts: 3 (including referrers)
[DRY RUN] Estimated upload size: 42.8 MB

No changes were made. Remove --dry-run to perform actual restore.
```

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

**Note:** All ORAS commands supporting `--oci-layout-path` can inspect backup contents with repository paths.

## Dependencies

This proposal depends on the following:

- **[#1770](https://github.com/oras-project/oras/issues/1770)**: Support `oras repo ls --oci-layout-path` for listing repositories in OCI Image Layout backups.

## Backup Format and Annotations

Multi-repository backups are stored in standard OCI Image Layout format with full reference information preserved in annotations.

### Reference Annotation (`org.opencontainers.image.ref.name`)

The `org.opencontainers.image.ref.name` annotation in `index.json` stores the artifact reference:

- **Default behavior (backward compatible)**: Contains just the tag (e.g., `"latest"`)
- **With `--full-reference` flag**: Contains the full reference (e.g., `"docker.io/library/alpine:latest"`)
- **Multi-repository backup**: Always uses full reference regardless of flag

#### The `--full-reference` Flag

The `--full-reference` flag can be used with single-repository backups to store the complete reference path:

```bash
oras backup --output backup.tar --full-reference docker.io/library/alpine:latest
```

This approach:
- Aligns with the [OCI Image Spec](https://github.com/opencontainers/image-spec/blob/v1.1.1/annotations.md#pre-defined-annotation-keys), which allows both tag-only and full reference formats in `org.opencontainers.image.ref.name`
- Enables automation tools (e.g., air-gap packaging tools like [Zarf](https://github.com/zarf-dev/zarf)) to programmatically determine the original location of backed-up artifacts
- Addresses use cases described in [#1893](https://github.com/oras-project/oras/issues/1893)
- **Multi-repository backups always use full references** to disambiguate artifacts from different repositories

> [!WARNING]
> Backups created with `--full-reference` are **not compatible** with ORAS CLI versions prior to v1.4.0 or tools that expect tag-only references in `org.opencontainers.image.ref.name`. Use this flag only when you need full reference information for automation or multi-registry scenarios.

**Example `index.json` formats:**

**Default single-repository backup (tag-only):**
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
        "org.opencontainers.image.ref.name": "latest"
      }
    }
  ]
}
```

**Single-repository backup with `--full-reference`:**
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

**Multi-repository backup (always uses full references):**

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
        "org.opencontainers.image.ref.name": "docker.io/library/busybox:1.36.1"
      }
    }
  ]
}
```

#### Restore Behavior with Full References

The `oras restore` command automatically detects the reference format in `org.opencontainers.image.ref.name`:

- **Tag-only format** (e.g., `"latest"`): Matches against the tag portion of the restore target
- **Full reference format** (e.g., `"docker.io/library/alpine:latest"`): Automatically extracts and matches the tag, then remaps to the restore target

**Example - Single-repo backup with full reference:**
```bash
# Backup with full reference
oras backup --output backup.tar --full-reference docker.io/library/alpine:latest

# Restore to different registry (auto-remaps)
oras restore --input backup.tar registry.mycompany.com/alpine:latest
# Result: docker.io/library/alpine:latest → registry.mycompany.com/alpine:latest
```

This auto-detection maintains backward compatibility while enabling new capabilities.

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

## User Experience Enhancements

### Progress Reporting

For multi-repository operations, progress is organized by repository:

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
Error: Failed to backup docker.io/library/postgres
  Reason: Authentication failed
  
Partial backup created: 2 of 3 repositories backed up successfully
  ✓ docker.io/library/busybox (1 tag)
  ✓ docker.io/library/alpine (1 tag)
  ✗ docker.io/library/postgres (authentication required)

Recommendations:
  1. Check credentials for docker.io
  2. Use --username and --password to provide authentication
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

## Authentication Handling

Multi-repository operations leverage the same authentication mechanisms as single-repository commands. Since all repositories in a backup operation must be from the same registry, all registry-specific flags (`--username`, `--password`, `--plain-http`, `--insecure`, `--ca-file`, etc.) work as expected.

**For multiple registries:** Create separate backups for each registry. This approach:
- Allows using registry-specific flags for each backup
- Simplifies authentication management
- Provides better organization (one backup per registry)
- Enables parallel backup operations

**Example:**
```bash
# Backup from Docker Hub
oras backup --output docker-hub-backup.tar \
  docker.io/library/nginx:1.25-alpine \
  docker.io/library/redis:7.2-alpine

# Backup from private registry with authentication
oras backup --output private-backup.tar \
  --username admin --password-stdin \
  registry.mycompany.com/app/frontend \
  registry.mycompany.com/app/backend
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
oras restore --input /backups/critical-apps-20251120.tar \
  registry.mycompany.com/production \
  --dry-run  # Verify first

# Then actual restore
oras restore --input /backups/critical-apps-20251120.tar \
  registry.mycompany.com/production
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

**For dependencies from multiple public registries:**

Create separate backups for each registry, then transfer and restore individually:

```bash
# Backup from Docker Hub
oras backup --output docker-images.tar \
  docker.io/library/nginx:1.25-alpine \
  docker.io/library/redis:7.2-alpine

# Backup from GHCR
oras backup --output ghcr-images.tar \
  ghcr.io/oras-project/oras:v1.1.0

# Transfer both tarballs, then restore separately on air-gapped system
oras restore --input docker-images.tar airgap.internal.company/docker
oras restore --input ghcr-images.tar airgap.internal.company/ghcr
```

## Flag Reference

### Backup Flags (New)

- `--full-reference`: Store complete reference path (registry/repository:tag) in `org.opencontainers.image.ref.name` annotation. Enables automation tools to determine original artifact location. Multi-repository backups always use full references. **Warning**: Backups created with this flag are not compatible with ORAS CLI versions prior to v1.4.0.
- `--dry-run`: Preview backup operation without downloading artifacts

### Restore Flags (New)

- `--rules <file>`: Rules configuration file for repository mapping

### Common Flags (Enhanced)

All existing authentication, TLS, and connection flags continue to work with multi-repository operations.

## Backward Compatibility

All existing single-repository commands continue to work unchanged:

```bash
# These continue to work exactly as before
oras backup --output backup.tar registry.com/repo:tag
oras restore --input backup.tar registry.com/repo:tag
oras backup --output backup.tar registry.com/repo
oras restore --input backup.tar registry.com/repo
```

Multi-repository support is activated only when:
- Multiple repository arguments are provided
- A rules file is specified

## Summary

This proposal extends `oras backup` and `oras restore` with intuitive multi-repository support within a single registry, while maintaining full backward compatibility. The design focuses on intentional, bounded backup operations where users explicitly specify which repositories to include.

Key features:
- **Multiple repositories in single backup**: Backup from multiple explicitly-specified repos within the same registry
- **Flexible restore options**: Direct, selective, or rules-based restoration
- **Repository mapping**: Powerful remapping capabilities for restructuring during restore
- **Safety features**: Dry-run preview and clear validation
- **Enterprise-ready**: Full support for registry-specific flags, error recovery, and progress reporting
- **Backward compatible**: Single-repository operations unchanged
- **Multi-registry approach**: For multiple registries, create separate backups for better organization and simpler authentication

**Out of Scope:**
- **Entire registry backup**: Not supported. Use registry-native replication/backup features for registry-wide operations.
- **Pattern-based selection**: Not included in this proposal. May be considered in future iterations if customer demand exists.
- **Cross-registry backup**: Requires separate backup operations per registry.

