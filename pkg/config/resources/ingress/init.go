package ingress

import "github.com/jumppad-labs/jumppad/pkg/config"

func init() {
	config.RegisterResource(TypeIngress, &Ingress{}, &Provider{})
}
