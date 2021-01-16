package utils

import (
	"fmt"
	"path"

	"github.com/shipyard-run/connector/crypto"
)

// creates a CA and local leaf cert
func GenerateLocalBundle(out string) error {
	// create the CA
	rk, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	ca, err := crypto.GenerateCA(rk.Private)
	if err != nil {
		return err
	}

	err = rk.Private.WriteFile(path.Join(out, "root.key"))
	if err != nil {
		return err
	}

	err = ca.WriteFile(path.Join(out, "root.cert"))
	if err != nil {
		return err
	}

	// generate a local cert
	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	err = k.Private.WriteFile(path.Join(out, "leaf.key"))
	if err != nil {
		return err
	}

	ips := GetLocalIPAddresses()
	host := GetHostname()

	fmt.Println(ips, host)

	lc, err := crypto.GenerateLeaf(
		ips,
		[]string{"localhost", "*.shipyard.run", host},
		ca,
		rk.Private,
		k.Private)
	if err != nil {
		return err
	}

	err = lc.WriteFile(path.Join(out, "leaf.cert"))
	return nil
}
