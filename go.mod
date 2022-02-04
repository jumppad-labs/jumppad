module github.com/shipyard-run/shipyard

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/MichaelMure/go-term-markdown v0.1.3
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/Microsoft/hcsshim v0.8.22 // indirect
	github.com/barkimedes/go-deepcopy v0.0.0-20200817023428-a044a1957ca4
	github.com/creack/pty v1.1.11
	github.com/cucumber/godog v0.10.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/docker/docker v20.10.12+incompatible
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/fatih/color v1.9.0
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gernest/front v0.0.0-20181129160812-ed80ca338b88
	github.com/gofiber/fiber/v2 v2.5.0
	github.com/gofiber/websocket/v2 v2.0.2
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/hashicorp/go-getter v1.5.6
	github.com/hashicorp/go-hclog v0.15.0
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80
	github.com/hashicorp/terraform v0.12.29
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/opencontainers/image-spec v1.0.2-0.20210819154149-5ad6f50d6283
	github.com/shipyard-run/connector v0.1.0
	github.com/shipyard-run/gohup v0.2.2
	github.com/shipyard-run/version-manager v0.0.5
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190809123943-df4f5c81cb3b // indirect
	github.com/zclconf/go-cty v1.5.1
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/net v0.0.0-20211005001312-d4b1ae081e3b // indirect
	golang.org/x/sys v0.0.0-20211004093028-2c5d950f24ef // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/genproto v0.0.0-20211005153810-c76a74d43a8e // indirect
	google.golang.org/grpc v1.41.0
	helm.sh/helm/v3 v3.7.2
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
)

replace github.com/creack/pty => github.com/shipyard-run/pty v1.1.12-0.20210531091229-b834701fbcc6

//replace golang.org/x/sys/windows => golang.org/x/sys/windows v0.0.0-20191005200804-aed5e5c7ecf12

//replace github.com/shipyard-run/connector => ../connector
