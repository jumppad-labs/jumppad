module github.com/shipyard-run/shipyard

go 1.13

require (
	github.com/DATA-DOG/godog v0.7.13
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/bugsnag/bugsnag-go v1.5.0 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-openapi/spec v0.19.4 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/hashicorp/go-getter v1.4.2-0.20200106182914-9813cbd4eb02
	github.com/hashicorp/go-hclog v0.10.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80
	github.com/hashicorp/terraform v0.12.20
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/prometheus/client_golang v1.2.1 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	github.com/xenolf/lego v0.3.2-0.20160613233155-a9d8cec0e656 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/zclconf/go-cty v1.2.1
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271 // indirect
	golang.org/x/sys v0.0.0-20200212091648-12a6c2dcc1e4 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191028173616-919d9bdd9fe6 // indirect
	gopkg.in/square/go-jose.v1 v1.1.2 // indirect
	helm.sh/helm/v3 v3.1.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	rsc.io/letsencrypt v0.0.1 // indirect
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20180718150940-a3ef7e9a9bda
