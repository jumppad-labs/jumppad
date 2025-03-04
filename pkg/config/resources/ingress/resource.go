package ingress

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeIngress is the resource string for the type
const TypeIngress string = "ingress"

/*
The ingress resource allows you to expose services in Kubernetes and Nomad tasks to the local machine.

It also allows you to expose applications that are running to the local machine to a Kubernetes or Nomad cluster.

@resource
*/
type Ingress struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		If the application to be exposed exists on the target then this is the port that will be opened on the local machine that will direct traffic to the remote service.

		If the application exists on the local machine then this is the port where the application is running.
	*/
	Port int `hcl:"port" json:"port"`
	/*
		If set to `true` a service running on the local machine will be exposed to the target cluster.
		If `false` then a service running on the target cluster will be exposed to the local machine.
	*/
	ExposeLocal bool `hcl:"expose_local,optional" json:"expose_local"`
	// The target for the ingress.
	Target TrafficTarget `hcl:"target,block" json:"target"`

	// path to open in the browser
	OpenInBrowser string `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"`
	/*
		The unique identifier for the created ingress.

		@computed
	*/
	IngressID string `hcl:"ingress_id,optional" json:"ingress_id,omitempty"`
	/*
		The full address where the exposed application can be reached from the local network.

		Generally this is the local ip address of the machine running Jumppad and the port where the application is exposed.

		@computed
	*/
	LocalAddress string `hcl:"local_address,optional" json:"local_address,omitempty"`
	/*
		The address of the exposed service as it would be rechable from the target cluster.

		This is generally a kubernetes service reference and port or for Nomad a rechable IP address and port.

		@computed
	*/
	RemoteAddress string `hcl:"remote_address,optional" json:"remote_address,omitempty"`
}

type TargetConfig struct {
	Meta          types.Meta `hcl:"meta" json:"meta"`
	ExternalIP    string     `hcl:"external_ip,optional" json:"external_ip,omitempty"`
	ConnectorPort int        `hcl:"connector_port,optional" json:"connector_port,omitempty"`
}

// Traffic defines either a source or a destination block for ingress traffic
type TrafficTarget struct {
	/*
		A reference to the `nomad_cluster` or `kubernetes_cluster` resource.

		@example
		```
		resource "k8s_cluster" "dev" {
		}

		resource "ingress" "consul_http" {
		  port = 18500

		  target {
		    resource = resource.k8s_cluster.dev
		    port     = 8500

		    config = {
		      service   = "consul-consul-server"
		      namespace = "default"
		    }
		  }
		}
		```
	*/
	Resource TargetConfig `hcl:"resource" json:"resource,omitempty"`
	/*
		The numerical reference for the target service port.

		Either `port` or `named_port` must be specified.
	*/
	Port int `hcl:"port,optional" json:"port,omitempty"`
	/*
		The string reference for the target service port.

		Either `port` or `named_port` must be specified.
	*/
	NamedPort string `hcl:"named_port,optional" json:"named_port,omitempty"`
	/*
		The configuration parameters for the ingress, configuration parameters differ depending on the target type.

		@example Kubernetes target config
		```
			service   = "Kubernetes service name"
			namespace = "Kubernetes namespace where the service is deployed"
		```

		@example Nomad target config
		```
			job   = "Name of the Nomad job"
			group = "Group in the job"
			task  = "Name of the task in the group"
		```
	*/
	Config map[string]string `hcl:"config" json:"config"`
}

func (i *Ingress) Process() error {
	// connector is a reserved name
	if i.Meta.Name == "connector" {
		return fmt.Errorf("ingress name 'connector' is a reserved name")
	}

	// validate the remote port, can not be 60000 or 60001 as these
	// ports are used by the connector service
	if i.Port == 60000 || i.Port == 60001 {
		return fmt.Errorf("unable to expose local service using remote port %d,"+
			"ports 60000 and 60001 are reserved for internal use", i.Port)
	}

	if i.Target.Config == nil {
		i.Target.Config = make(map[string]string)
	}

	sn, _ := utils.ReplaceNonURIChars(i.Target.Config["service"])
	// if service is not set, use the name of the ingress
	if i.Target.Config["service"] == "" {
		sn, _ = utils.ReplaceNonURIChars(i.Meta.Name)
	}

	i.Target.Config["service"] = sn

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(i.Meta.ID)
		if r != nil {
			kstate := r.(*Ingress)
			i.IngressID = kstate.IngressID
			i.LocalAddress = kstate.LocalAddress
			i.RemoteAddress = kstate.RemoteAddress
		}
	}

	return nil
}
