module github.com/deislabs/oras

go 1.16

replace (
	// WARNING! Do NOT replace these without also replacing their lines in the `require` stanza below.
	// These `replace` stanzas are IGNORED when this is imported as a library
	github.com/containerd/containerd => github.com/notaryproject/containerd v1.5.0-beta.4.0.20210317114357-17e18de6643b
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/opencontainers/artifacts => github.com/aviral26/artifacts v0.0.0-20210301104922-9ca8c5490fcc
)

require (
	github.com/containerd/containerd v1.5.0-beta.3
	github.com/docker/cli v20.10.5+incompatible
	github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/artifacts v0.0.0-20200916021946-9d10a670db1b
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
)
