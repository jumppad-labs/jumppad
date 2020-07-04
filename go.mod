module github.com/shipyard-run/shipyard

go 1.13

require (
	github.com/DATA-DOG/godog v0.7.13
	github.com/MichaelMure/go-term-markdown v0.1.3
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/cucumber/godog v0.10.0
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go v1.5.1-1 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/gernest/front v0.0.0-20181129160812-ed80ca338b88
	github.com/go-noisegate/noisegate v0.0.0-20200426084925-117e8e7980ca // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/hashicorp/go-getter v1.4.2-0.20200106182914-9813cbd4eb02
	github.com/hashicorp/go-hclog v0.10.1
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80
	github.com/hashicorp/terraform v0.12.20
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/prometheus/common v0.7.0 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.6.1
	github.com/theupdateframework/notary v0.6.1 // indirect
	github.com/zclconf/go-cty v1.2.1
	golang.org/x/sys v0.0.0-20200212091648-12a6c2dcc1e4 // indirect
	golang.org/x/tools v0.0.0-20200426102838-f3a5411a4c3b // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	helm.sh/helm/v3 v3.1.1
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20180718150940-a3ef7e9a9bda
