module github.com/deislabs/oras

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/Microsoft/hcsshim v0.8.6 // indirect
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v0.4.1 // indirect
	github.com/bugsnag/bugsnag-go v1.5.0 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.3.0-beta.2.0.20190823190603-4a2f61c4f2b4
	github.com/containerd/continuity v0.0.0-20181203112020-004b46473808 // indirect
	github.com/docker/cli v0.0.0-20190506213505-d88565df0c2d
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
	github.com/docker/docker v0.0.0-00010101000000-000000000000
	github.com/docker/docker-credential-helpers v0.6.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gorilla/mux v1.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/miekg/dns v0.0.0-20181005163659-0d29b283ac0f // indirect
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/phayes/freeport v0.0.0-20171002181615-b8543db493a5
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90 // indirect
	github.com/prometheus/common v0.2.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190129233650-316cf8ccfec5 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/xenolf/lego v0.0.0-20160613233155-a9d8cec0e656 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20140908184405-b21fdbd4370f // indirect
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	google.golang.org/genproto v0.0.0-20190128161407-8ac453e89fca // indirect
	google.golang.org/grpc v1.20.1 // indirect
	gopkg.in/square/go-jose.v1 v1.1.2 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	rsc.io/letsencrypt v0.0.1 // indirect
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	rsc.io/letsencrypt => github.com/dmcgowan/letsencrypt v0.0.0-20160928181947-1847a81d2087
)
