package cmd

import (
	"fmt"
	"path"

	"github.com/jumppad-labs/connector/crypto"
	"github.com/spf13/cobra"
)

func newConnectorCertCmd() *cobra.Command {
	var generateCA bool
	var generateLeaf bool
	var rootKey string
	var rootCA string
	var ipAddresses []string
	var dnsNames []string

	connectorCertCmd := &cobra.Command{
		Use:   "generate-certs [output location]",
		Short: "Generate TLS certificates for the server to the specified output location",
		Long:  `Allows you to generate a TLS root and leaf certificates for securing connector communication`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			if generateCA {
				k, err := crypto.GenerateKeyPair()
				if err != nil {
					return err
				}

				c, err := crypto.GenerateCA("Connector CA", k.Private)
				if err != nil {
					return err
				}

				err = k.Private.WriteFile(path.Join(args[0], "root.key"))
				if err != nil {
					return err
				}

				err = c.WriteFile(path.Join(args[0], "root.cert"))
				if err != nil {
					return err
				}

				return nil
			}

			if generateLeaf {
				// load the root key
				ca := &crypto.X509{}
				err := ca.ReadFile(rootCA)
				if err != nil {
					return fmt.Errorf("Unable to read root certificate: %s", rootCA)
				}

				rk := crypto.NewKeyPair()
				err = rk.Private.ReadFile(rootKey)
				if err != nil {
					return fmt.Errorf("Unable to read root certificate: %s", rootKey)
				}

				k, err := crypto.GenerateKeyPair()
				if err != nil {
					return err
				}

				err = k.Private.WriteFile(path.Join(args[0], "leaf.key"))
				if err != nil {
					return err
				}

				lc, err := crypto.GenerateLeaf("Connector Leaf", ipAddresses, dnsNames, ca, rk.Private, k.Private)
				if err != nil {
					return err
				}

				err = lc.WriteFile(path.Join(args[0], "leaf.cert"))
				return nil
			}

			return nil
		},
	}

	connectorCertCmd.Flags().BoolVarP(&generateCA, "ca", "", false, "Generate a CA x509 certificate and private key")
	connectorCertCmd.Flags().BoolVarP(&generateLeaf, "leaf", "", false, "Generate a leaf c509 certificate and private key")
	connectorCertCmd.Flags().StringVarP(&rootKey, "root-key", "", "", "Root key to use for generating the leaf certificate")
	connectorCertCmd.Flags().StringVarP(&rootCA, "root-ca", "", "", "CA cert to use for generating the leaf certificate")
	connectorCertCmd.Flags().StringSliceVarP(&ipAddresses, "ip-address", "", []string{}, "IP address to add to the leaf certificate")
	connectorCertCmd.Flags().StringSliceVarP(&dnsNames, "dns-name", "", []string{}, "DNS name to add to leaf certificate")

	return connectorCertCmd
}
