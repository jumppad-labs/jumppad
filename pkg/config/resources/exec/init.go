package exec

import "github.com/jumppad-labs/jumppad/pkg/config"

// register the types and provider
func init() {
	config.RegisterResource(TypeLocalExec, &LocalExec{}, &LocalProvider{})
	config.RegisterResource(TypeRemoteExec, &RemoteExec{}, &RemoteProvider{})
}
