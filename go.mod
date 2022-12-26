module oras.land/oras

go 1.19

require (
	github.com/docker/cli v20.10.22+incompatible
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/need-being/go-tree v0.1.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc2
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	oras.land/oras-go/v2 v2.0.0-20221221030937-258bfba75241
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/docker/docker v20.10.17+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)

replace oras.land/oras-go/v2 => github.com/shizhMSFT/oras-go/v2 v2.0.0-20221223085227-7566ecd0c1f1
