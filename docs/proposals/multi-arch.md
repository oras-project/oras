# Multi-arch image support for ORAS

## Command Design

### Overview

- `oras manifest index create`: Create an image index from source manifests. 
- `oras manifest index update`: Add/remove manifests to/from an image index.

### Create an image index

#### Definition

Create an image index from source manifests. The command auto-detects platform information for each source manifests.

> [!IMPORTANT]
> All source manifests referenced are required to exist in the same repository as the target index.

Usage:
```
oras manifest index create [flags] <name>[:<tag[,<tag>][...]] [{<tag>|<digest>}...]
```

Flags:

- `--subject`: Add a subject manifest for the to-be-created index.
- `--oci-layout`: Set the given repository as an OCI layout.
- `--annotation`: Add annotations for the to-be-created index.
- `--annotation-file`: Add annotations for the to-be-created index and the individual source manifests.
- `--artifact-type`: Add artifact type information for the to-be-created index.
- `--output`: Output the updated manifest to a location. Auto push is disabled.

Aliases: `pack`

#### Examples

Create an index from source manifests tagged `amd64`, `darwin`, `armv7` in the repository `localhost:5000/hello`, and push the index without tagging it.

```sh
oras manifest index create localhost:5000/hello amd64 darwin armv7
```

Create an index from source manifests tagged amd64, darwin, armv7 in the repository localhost:5000/hello, and push the index with tag `latest`:

```sh
oras manifest index create localhost:5000/hello:latest amd64 darwin armv7
```

Create an index from source manifests using both tags and digests, and push the index with tag `latest`:

```sh
oras manifest index create localhost:5000/hello:latest amd64 sha256:xxx armv7
```

Create an index and push it with multiple tags:

```sh
oras manifest index create localhost:5000/tag1, tag2, tag3 amd64 arm64 sha256:xxx
```

### Update an image index

#### Definition

Add/Remove a manifest from an image index. The updated index will be created as a new index and the old index will not be deleted. 
If the user specify the index with tags, the corresponding tags will be updated to the new index. If the old index has other tags, the remaining tags will not be updated to the new index.

Usage:

```
oras manifest index update <name>{:<tag>|@<digest>} {--add/--remove/--annotation/--annotation-file} {...}
```

Flags:

- `--add`: Add a manifest to the index. The manifest will be added as the last element of the index.
- `--remove`: Remove a manifest from the index.
- `--annotation`: Update annotations for the index.
- `--annotation-file`: Update annotations for the index and the individual manifests.
- `--oci-layout`: Set the target as an oci image layout.
- `--tag`: Tag the updated index. Multiple tags can be provided.
- `--output`: Output the updated manifest to a location. Auto push is disabled.

> [!NOTE]
> One of `--add`/`--remove`/`--annotation`/`--annotation-file` should be used, as there has to be something to update. Otherwise the command does nothing.

#### Examples

Add one manifest and remove two manifests from an index.

```sh
oras manifest index update localhost:5000/hello:latest --add win64 --remove sha256:xxx --remove arm64
```

Update the index referenced by tag1 and tag3, and make tag1 and tag3 point to the
updated index. If the old index has other tags, they remain pointing to the old index.

```sh
oras manifest index update localhost:5000/hello:tag1,tag3 --remove sha256:xxx --remove sha256:xxx --add s390x
```

## Design Considerations

### Command design: Making a subcommand group `index` under `oras manifest`

`oras manifest index create` is chosen instead of `oras manifest create-index` with the following reasons:
* The structure of the `oras manifest index` sub command group aligns well with the existing sub command groups `oras manifest/blob/repo`.
* If in the future more index commands are needed, grouping them under the `index` group makes the manifest commands neater. Operations for other manifest types may be needed in the future, and creating new sub groups parallel to `index` looks feasible (i.e. `oras manifest image create`).

### Combining _add manifest_ and _remove manifest_ operations as one `index update` operation

Combining the add and remove manifest operations as one `update` command has several benefits, such as it makes less garbage and fewer request calls when doing multiple adds and removes.

### `oras manifest index create / update` will auto detect platform information for each source manifest.

Platform information is automatically detected from the manifest config. Specifying
platform by the user might be supported in the future.

## FAQ

### Should we require all the source manifests and the to-be-created index to be in the same repository?

Yes, as allowing multiple repositories will introduce a lot of copying of missing blobs and manifests.

### Should we automatically push the created/updated index?

Yes, the created/updated index is automatically pushed for better user experience.
