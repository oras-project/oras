# ORAS Backup and Restore Commands

The backup and restore commands will add the capability to backup a list of artifacts from a registry and restore them to another registry.
Backup and restore will be support for any OCI compatible artifact (e.g. container images, helm charts, configuration files,...).


## Overview 

This document outlines the different ways that the ORAS backup and restore commands can be used.


## Problem Statement & Motivation 

Currently, ORAS commands function on one artifact at a time, so copying a large numbers of artifacts will require scripting.
There is also no ability to copy multiple artifacts to a compressed tar file.

* https://github.com/oras-project/oras/issues/1366
* https://github.com/oras-project/oras/issues/730


## Scenarios 

This feature will be useful for mirroring, air gapped registries, and disaster recovery.

Users often need to mirror registries for performance and reliability reasons.
Pulling images over the Internet can be significantly slower than pulling images from a local registry.
If network connections are unreliable, a local registry will potentially be a lot more reliable.

Air gapped environments normally require users to copy many artifiacts to a portable storage medium and sneaker net that storage into the environment.
The backup and restore commands will make writing that portable storage medium easy.

The backup and restore commands will also help users wanting to copy artifiacts from a registry for disaster recovery.
It may be significantly easier to restore a registy from a backup rather than recreate the artifacts.


## Existing Solutions or Expectations

This functionality is similar, but more flexible than `docker save` and `docker load`.
The Docker commands only allow the use of a tar file.


## Proposal 

The backup and restore commands will support reading and writing multiple files to and from a registry.
As well as the flags described here, the commands will support the normal set of flags to support TLS and authentication.


### oras backup

The backup command will read a list of atifacts from the command line or from standard input.
It will support writing to a directory or compressed tar file.

For example, backing up artifacts specified on the command line to a directory:

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

Each image will be stored in a subdirectory which matches the repository name.

The backup command will also have the ability to write output to a compressed tar file. For example:

```bash
oras backup --output ./mirror.tgz  registry.k8s.io/kube-apiserver-arm64:v1.31.0 registry.k8s.io/kube-controller-manager-arm64:v1.31.0
```

There will be no validation on file name.
The file name does not need to end in `tgz`, but the format will be compressed tar file.
If the file exists, it will be overwritten.

#### Backup file input from a file

The backup command will support an `--input filename` argument which will be a file containing the remote resources to retrieve.
The format of the contents of the file is a list of images names separated by newlines.
If the `--input` argument is specified, no images may be specified on the command line.

#### Backup file input from standard input

The backup command will support the `--input -` argument to read a list of images from standard input.
The format of the input is a list of images names separated by newlines.

#### Optimize blobs

A further enhancement is to create blobs that are duplicated between images as hard links in the directory output and compressed tar files.


### oras restore

The restore command will support reading a directory or a compressed tar file and writing the content to a remote registry.

#### Restore from a directory

An example of restoring from a directory:

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

#### Restore from a compressed tar file

The directory structure in the tar file will be the same as in the directory output.
An example of reading from a compressed tar file:

```bash
oras restore --input ./mirror.tgz localhost:15000/my-mirror
```

If the specified source is a file, the format is assumed to be a compressed tar file.
There will be no validation of file name format.

#### Restore file input from standard input

The restore command will support the `--input -` argument to read a compressed tar input from standard input.
