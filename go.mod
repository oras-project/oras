module oras.land/oras

go 1.16

replace (
	// WARNING! Do NOT replace these without also replacing their lines in the `require` stanza below.
	// These `replace` stanzas are IGNORED when this is imported as a library
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/containerd v1.5.9
	github.com/docker/docker v17.12.1-ce+incompatible
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	oras.land/oras-go v0.4.0
)
