module github.com/deislabs/oras

go 1.16

replace (
	// WARNING! Do NOT replace these without also replacing their lines in the `require` stanza below.
	// These `replace` stanzas are IGNORED when this is imported as a library
	github.com/containerd/containerd => github.com/oras-project/containerd v1.5.0-beta.4.0.20210914182246-c90d5cff6817
	github.com/containerd/containerd/api => github.com/oras-project/containerd/api v0.0.0-20210914182246-c90d5cff6817
	// This one keeps switching versions
	github.com/oras-project/artifacts-spec => github.com/oras-project/artifacts-spec v0.0.0-20210910233110-813953a626ae
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)

require (
	github.com/containerd/containerd v1.5.1
	github.com/containerd/containerd/api v1.5.0-beta.3 // indirect
	github.com/docker/cli v20.10.5+incompatible
	github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/need-being/go-tree v0.1.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/oras-project/artifacts-spec v0.0.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)
