module oras.land/oras

go 1.21

require (
	github.com/containerd/console v1.0.3
	github.com/morikuni/aec v1.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc5
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/sync v0.6.0
	golang.org/x/term v0.16.0
	gopkg.in/yaml.v3 v3.0.1
	oras.land/oras-go/v2 v2.3.1-0.20231227022511-1d9ad6c409b3
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
)

replace oras.land/oras-go/v2 => github.com/qweeah/oras-go/v2 v2.3.6
