module oras.land/oras

go 1.20

replace oras.land/oras-go/v2 => github.com/qweeah/oras-go/v2 v2.0.0-20230414234922-c83cd27a25ff

require (
	github.com/need-being/go-tree v0.1.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc2
	github.com/oras-project/oras-credentials-go v0.1.0
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/term v0.8.0
	gopkg.in/yaml.v3 v3.0.1
	oras.land/oras-go/v2 v2.1.0
)

require (
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
)
