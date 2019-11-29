package providers

import "fmt"

// FQDN generate the name of a docker container
func FQDN(name, networkName string) string {
	return fmt.Sprintf("%s.%s.shipyard", name, networkName)
}
