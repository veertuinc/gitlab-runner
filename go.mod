module gitlab.com/gitlab-org/gitlab-runner

go 1.13

require (
	cloud.google.com/go v0.49.0 // indirect
	cloud.google.com/go/storage v1.0.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/ayufan/golang-kardianos-service v0.0.0-20160429143213-0c8eb6d8fff2
	github.com/containerd/continuity v0.0.0-20181203112020-004b46473808 // indirect
	github.com/docker/cli v0.0.0-20181219132003-336b2a5cac7f
	github.com/docker/distribution v2.7.0+incompatible
	github.com/docker/docker v1.4.2-0.20190822180741-9552f2b2fdde
	github.com/docker/docker-credential-helpers v0.4.1 // indirect
	github.com/docker/go-connections v0.3.0
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/go-units v0.3.2-0.20160802145505-eb879ae3e2b8 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/machine v0.7.1-0.20170120224952-7b7a141da844
	github.com/fullsailor/pkcs7 v0.0.0-20190404230743-d7302db945fa
	github.com/getsentry/raven-go v0.0.0-20160518204710-dffeb57df75d
	github.com/golang/mock v1.3.1
	github.com/gorhill/cronexpr v0.0.0-20160318121724-f0984319b442
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.3.1-0.20170228224354-599cba5e7b61
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/jpillora/backoff v0.0.0-20170222002228-06c7a16c845d
	github.com/kardianos/osext v0.0.0-20160811001526-c2c54e542fb7 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/markelog/trie v0.0.0-20171230083431-098fa99650c0
	github.com/minio/minio-go/v6 v6.0.49
	github.com/mitchellh/gox v1.0.1
	github.com/onsi/ginkgo v1.10.3 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc6.0.20190115182101-c1e454b2a1bf // indirect
	github.com/pkg/errors v0.8.0
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.6.0
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/sanity-io/litter v1.2.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tdewolff/parse/v2 v2.4.2 // indirect
	github.com/tevino/abool v0.0.0-20160628101133-3c25f2fe7cd0
	github.com/urfave/cli v1.20.0
	github.com/vektra/mockery v1.1.2
	gitlab.com/ayufan/golang-cli-helpers v0.0.0-20171103152739-a7cf72d604cd
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae
	gopkg.in/ini.v1 v1.52.0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689
)

replace github.com/docker/docker v1.4.2-0.20190822180741-9552f2b2fdde => github.com/docker/engine v1.4.2-0.20190822180741-9552f2b2fdde

replace github.com/minio/go-homedir v0.0.0-20190425115525-017018655514 => gitlab.com/steveazz/go-homedir v0.0.0-20190425115525-017018655514
