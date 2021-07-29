module github.com/shipyard-run/shipyard

go 1.16

require (
	github.com/Masterminds/semver v1.5.0
	github.com/MichaelMure/go-term-markdown v0.1.3
	github.com/Microsoft/go-winio v0.4.17 // indirect
	github.com/aws/aws-sdk-go v1.33.5 // indirect
	github.com/containerd/cgroups v1.0.1 // indirect
	github.com/containerd/continuity v0.1.0 // indirect
	github.com/creack/pty v1.1.11
	github.com/cucumber/godog v0.10.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/docker/docker v17.12.0-ce-rc1.0.20210212105829-69f9c8c90662+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/gernest/front v0.0.0-20181129160812-ed80ca338b88
	github.com/gofiber/fiber/v2 v2.5.0
	github.com/gofiber/websocket/v2 v2.0.2
	github.com/google/uuid v1.2.0 // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/hashicorp/go-getter v1.5.1
	github.com/hashicorp/go-hclog v0.15.0
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80
	github.com/hashicorp/terraform v0.12.29
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.4.0
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/opencontainers/runc v1.0.0-rc93 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/shipyard-run/connector v0.0.18
	github.com/shipyard-run/gohup v0.2.2
	github.com/shipyard-run/version-manager v0.0.5
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/zclconf/go-cty v1.5.1
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/api v0.30.0 // indirect
	google.golang.org/grpc v1.33.2
	helm.sh/helm/v3 v3.6.3
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	rsc.io/letsencrypt v0.0.3 // indirect
)

//replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20180718150940-a3ef7e9a9bda

//replace golang.org/x/sys => golang.org/x/sys v0.0.0-20190830141801-acfa387b8d69

replace github.com/creack/pty => github.com/jeffreystoke/pty v1.1.12-0.20201126201855-c1c1e24408f9

//replace github.com/shipyard-run/connector => ../connector
