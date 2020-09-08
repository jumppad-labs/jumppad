module github.com/shipyard-run/shipyard

go 1.14

require (
	github.com/MichaelMure/go-term-markdown v0.1.3
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/apex/log v1.9.0
	github.com/aws/aws-sdk-go v1.33.5 // indirect
	github.com/cucumber/godog v0.10.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/gernest/front v0.0.0-20181129160812-ed80ca338b88
	github.com/gosuri/uitable v0.0.4
	github.com/hashicorp/go-getter v1.5.0
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80
	github.com/hashicorp/terraform v0.12.29
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/kr/pretty v0.2.0
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.3.3
	github.com/shipyard-run/version-manager v0.0.5
	github.com/shipyard-run/connector v0.0.9
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/zclconf/go-cty v1.5.1
	golang.org/x/sys v0.0.0-20200806060901-a37d78b92225 // indirect
	golang.org/x/tools v0.0.0-20200806022845-90696ccdc692 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/api v0.30.0 // indirect
	google.golang.org/grpc v1.31.0
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.0
	k8s.io/apimachinery v0.18.0
	k8s.io/client-go v0.18.0
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20180718150940-a3ef7e9a9bda

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20190830141801-acfa387b8d69

//replace github.com/nicholasjackson/version-manager => ../../nicholasjackson/version-manager
