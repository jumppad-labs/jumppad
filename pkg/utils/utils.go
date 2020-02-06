package utils

import (
	"fmt"
)

// FQDN generates the full qualified name for a container
func FQDN(name string, networkName string) string {
	if networkName == "" {
		return fmt.Sprintf("%s.shipyard", name)
	}

	return fmt.Sprintf("%s.%s.shipyard", name, networkName)
}
