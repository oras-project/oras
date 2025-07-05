# ORAS Backup and Restore Commands

The backup and restore commands will add the capability to backup a list of artifacts from a registry and restore them to another registry.
Backup and restore will be supported for any OCI compatible artifact (e.g. container images, helm charts, configuration files,...).


## Overview 

This document outlines various scenarios related to backing up and restoring OCI images from registries and local files. It covers a common workflow for downloading and uploading one or more image resources to and from disk, identifies limitations and challenges in existing solutions, and presents proposals for improvements to enhance usability and portability of managing OCI images.


## Problem Statement & Motivation 

Currently, ORAS commands function on one artifact at a time, so copying a large number of artifacts will require scripting.
There is also no ability to copy multiple artifacts to a compressed tar file.

* https://github.com/oras-project/oras/issues/1366
* https://github.com/oras-project/oras/issues/730


## Scenarios 

This feature will be useful for mirroring, air gapped registries, and disaster recovery.

Users often need to mirror registries for performance and reliability reasons.
Pulling images over the Internet can be significantly slower than pulling images from a local registry.
If network connections are unreliable, a local registry will potentially be a lot more reliable.

Air gapped environments normally require users to copy many artifacts to a portable storage medium and sneaker net that storage into the environment.
The backup and restore commands will make writing that portable storage medium easy.

The backup and restore commands will also help users wanting to copy artifacts from a registry for disaster recovery.
It may be significantly easier to restore a registry from a backup rather than recreate the artifacts.


## Existing Solutions or Expectations

This functionality is similar, but more flexible than `docker save` and `docker load`.
The Docker commands only allow the use of a tar file.


## Proposal 

The backup and restore commands will support reading and writing multiple files to and from a registry.
As well as the flags described here, the commands will support the normal set of flags to support TLS and authentication.


### oras backup

The backup command will read a list of artifacts from the command line or from standard input.
It will support writing to a directory or compressed tar file.

For example, backing up artifacts specified on the command line to a directory:

```bash
oras backup --output ./mirror  registry.k8s.io/kube-apiserver-arm64:v1.31.0 registry.k8s.io/kube-controller-manager-arm64:v1.31.0
```

It is mandatory to specify `--output` argument with the destination.
The source artifacts may be read from different registries although the example reads artifacts from one registry.
If no reference tag or digest is specified, the entire repository will be copied.

The output directory structure is a single OCI layout containing all of the artifacts.
Each artifact in the output OCI layout will be tagged with the name of source.
For example, the `registry.k8s.io/kube-apiserver-arm64:v1.31.0` artifact will be tagged `registry.k8s.io/kube-apiserver-arm64:v1.31.0`.

```bash
% oras repo tags --oci-layout ./mirror
registry.k8s.io/kube-apiserver-arm64:v1.31.0
registry.k8s.io/kube-apiserver-arm64:v1.32.0
registry.k8s.io/kube-controller-manager-arm64:v1.31.0
```

The backup command will also have the ability to write output to a new compressed tar file where the contents are in a single oci-layout
format. For example:

```bash
oras backup --output ./mirror.tgz  registry.k8s.io/kube-apiserver-arm64:v1.31.0 registry.k8s.io/kube-controller-manager-arm64:v1.31.0
```

If the output specified is an existing directory, the output will be written in that directory in OCI layout format.
If the output specified is an existing file, it will be overwritten.
If the output specified is neither a file or a directory, file output is assumed.
The file name does NOT need to end in `tgz`, but file output will be compressed tar file.

### oras restore

The restore command will support reading a directory or a compressed tar file and writing the content to a remote registry.

#### Restore from a directory

An example of restoring from a directory:

```bash
oras restore --input ./mirror localhost:15000/my-mirror
```

It is mandatory to specify `--input` argument with the source directory or file.
The destination registry that is being restored to may be different from the source registry.
An option will be provided to map repositories from the backup to different repositories on the destination registry.
For example, a backup of `foo.registry.example/test` can be restored to `bar.registry.example/another-test` where `test` is mapped to `another-test`.
The tags in the input OCI layout will be used to reconstruct the source.

The above restore example would result in:
```console
% oras repo ls localhost:15000/my-mirror
kube-apiserver-arm64
kube-controller-manager-arm64
% oras repo tags localhost:15000/my-mirror/kube-apiserver-arm64
v1.31.0
v1.32.0
```

A namespace in the registry will be optional.
The registry in the above example could be specified as `localhost:15000`.

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
