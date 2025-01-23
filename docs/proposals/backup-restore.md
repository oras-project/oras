# Backup and Restore Commands

This document describes how the backup and restore commands will work and how the feature will be rolled out.
The proposal is to create the feature as a minimum viable product and follow that up with non breaking features to the commands.
The initial implementation will be the easiest implementation that is useful.

## Minium Viable Product

The backup command will initially just support writing to a directory and the restore command will only support reading from a directory.

### oras backup

The backup command will initially only support reading a list of files from the command line and writing to a directory:

```bash
oras backup --output ./mirror  registry.k8s.io/kube-apiserver-arm64:v1.31.0 registry.k8s.io/kube-controller-manager-arm64:v1.31.0
```

The generated directory structure is `<specified-directory>/<repository>`.
The command above puts the OCI layout for the Kubernetes API server in `mirror/kube-api-server-arm64`.
The directory structure with intermediate blobs removed:

```bash
$ find mirror
mirror
mirror/kube-apiserver-arm64
mirror/kube-apiserver-arm64
mirror/kube-apiserver-arm64/ingest
mirror/kube-apiserver-arm64/oci-layout
mirror/kube-apiserver-arm64/blobs
mirror/kube-apiserver-arm64/blobs/sha256
mirror/kube-apiserver-arm64/blobs/sha256/3f4e2c5863480125882d92060440a5250766bce764fee10acdbac18c872e4dc7
...
mirror/kube-apiserver-arm64/blobs/sha256/4f80fb2b9442dbecd41e68b598533dcaaf58f9d45cce2e03a715499aa9f6b676
mirror/kube-apiserver-arm64/index.json
mirror/kube-controller-manager-arm64
mirror/kube-controller-manager-arm64
mirror/kube-controller-manager-arm64/ingest
mirror/kube-controller-manager-arm64/oci-layout
mirror/kube-controller-manager-arm64/blobs
mirror/kube-controller-manager-arm64/blobs/sha256
mirror/kube-controller-manager-arm64/blobs/sha256/3f4e2c5863480125882d92060440a5250766bce764fee10acdbac18c872e4dc7
...
mirror/kube-controller-manager-arm64/blobs/sha256/4f80fb2b9442dbecd41e68b598533dcaaf58f9d45cce2e03a715499aa9f6b676
mirror/kube-controller-manager-arm64/index.json
$
```

The directory `mirror` must exist for the initial feature.
Each image will be stored in a subdirectory which matches the repository name.

### oras restore

Initially, the restore command will only support reading a directory and writing to a registry:

```bash
oras restore --input ./mirror localhost:15000/my-mirror
```

The above backup example would result in:
```bash
localhost:15000/my-mirror/kube-apiserver-arm64:v1.31.0
localhost:15000/my-mirror/kube-controller-manager-arm64:v1.31.0
```

A namespace in the registry will be optional.
The registry in the above example could be specified as `localhost:15000`.


Any directory in the input that does not contain an `index.json` shall be silently ignored.

## Compressed Tar File Support

After the command supports directory write and read, support will be added for write and read to a compressed tar file.
The directory structure in the tar file would be the same as in the directory output.

```bash
oras backup --output ./mirror.tgz  registry.k8s.io/kube-apiserver-arm64:v1.31.0 registry.k8s.io/kube-controller-manager-arm64:v1.31.0
```

There will be no validation on file name.
The file name does not need to end in `tgz`, but the format will be compressed tar file.
If the file exists, it will be overwritten.

```bash
oras restore --input ./mirror.tgz localhost:15000/my-mirror
```
If the specified source is a file, the format is assumed to be a compressed tar file.

## Backup file input from a file

The backup command will support an `--input filename` argument which will be a file.
The format of the contents of the file is a list of images names separated by newlines.
If the `--input` argument is specified, no images may be specified on the command line.

## Backup file input from standard input

The backup command will support the `--input -` argument to read a list of images from standard input.
The format of the input is a list of images names separated by newlines.

## Restore file input from standard input

The restore command will support the `--input -` argument to read a compressed tar input from standard input.

## Optimize blobs

Add the feature to create blobs that are duplicated between images as hard links in the directory output and compressed tar files.
