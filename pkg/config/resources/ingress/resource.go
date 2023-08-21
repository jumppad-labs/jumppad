package ingress

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeIngress is the resource string for the type
const TypeIngress string = "ingress"

// Ingress defines an ingress service mapping ports between local host and resources like containers and kube cluster
type Ingress struct {
	types.ResourceMetadata `hcl:",remain"`

	// local port to expose the service on
	Port int `hcl:"port" json:"port"`

	// Are we exposing a local serve to the target
	// if
	ExposeLocal bool `hcl:"expose_local,optional" json:"expose_local"`

	// details for the destination service
	Target TrafficTarget `hcl:"target,block" json:"target"`

	// path to open in the browser
	OpenInBrowser string `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"`

	// --- Output Params ----

	// IngressId stores the ID of the created connector service
	IngressID string `hcl:"ingress_id,optional" json:"ingress_id,omitempty"`

	// Address is the fully qualified uri for accessing the resource
	Address string `hcl:"address,optional" json:"address,omitempty"`
}

type TargetConfig struct {
	Name          string `hcl:"name,optional" json:"name,omitempty"`
	Type          string `hcl:"type,optional" json:"type,omitempty"`
	ExternalIP    string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
	ConnectorPort int    `hcl:"connector_port,optional" json:"connector_port,omitempty"`
}

// Traffic defines either a source or a destination block for ingress traffic
type TrafficTarget struct {
	Resource TargetConfig `hcl:"resource" json:"resource,omitempty"`

	Port      int    `hcl:"port,optional" json:"port,omitempty"`
	NamedPort string `hcl:"named_port,optional" json:"named_port,omitempty"`

	// Config is an collection which has driver specific content
	Config map[string]string `hcl:"config" json:"config"`
}

func (i *Ingress) Process() error {
	// connector is a reserved name
	if i.Name == "connector" {
		return fmt.Errorf("ingress name 'connector' is a reserved name")
	}

	// validate the remote port, can not be 60000 or 60001 as these
	// ports are used by the connector service
	if i.Port == 60000 || i.Port == 60001 {
		return fmt.Errorf("unable to expose local service using remote port %d,"+
			"ports 60000 and 60001 are reserved for internal use", i.Port)
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(i.ID)
		if r != nil {
			kstate := r.(*Ingress)
			i.IngressID = kstate.IngressID
			i.Address = kstate.Address
		}
	}

	return nil
}
