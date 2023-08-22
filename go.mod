module oras.land/oras

go 1.20

replace oras.land/oras-go/v2 => github.com/Wwwsylvia/oras-go/v2 v2.0.0-20230821122310-d774911dfd59

require (
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc4
	github.com/oras-project/oras-credentials-go v0.2.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/term v0.11.0
	gopkg.in/yaml.v3 v3.0.1
	oras.land/oras-go/v2 v2.2.1-0.20230807082644-bbe92af00542
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
)
