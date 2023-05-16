package resources

import (
	"os"

	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig"
)

// setupHCLConfig configures the HCLConfig package and registers the custom types
func SetupHCLConfig(callback hclconfig.ProcessCallback, variables map[string]string, variablesFiles []string) *hclconfig.Parser {
	cfg := hclconfig.DefaultOptions()
	cfg.ParseCallback = callback
	cfg.Variables = variables
	cfg.VariablesFiles = variablesFiles

	p := hclconfig.NewParser(cfg)

	// Register the types
	p.RegisterType(TypeCertificateCA, &CertificateCA{})
	p.RegisterType(TypeCertificateLeaf, &CertificateLeaf{})
	p.RegisterType(TypeContainer, &Container{})
	p.RegisterType(TypeCopy, &Copy{})
	p.RegisterType(TypeDocs, &Docs{})
	p.RegisterType(TypeRemoteExec, &RemoteExec{})
	p.RegisterType(TypeHelm, &Helm{})
	p.RegisterType(TypeImageCache, &ImageCache{})
	p.RegisterType(TypeIngress, &Ingress{})
	p.RegisterType(TypeK8sCluster, &K8sCluster{})
	p.RegisterType(TypeK8sConfig, &K8sConfig{})
	p.RegisterType(TypeLocalExec, &LocalExec{})
	p.RegisterType(TypeNetwork, &Network{})
	p.RegisterType(TypeNomadCluster, &NomadCluster{})
	p.RegisterType(TypeNomadJob, &NomadJob{})
	p.RegisterType(TypeRandomNumber, &RandomNumber{})
	p.RegisterType(TypeRandomID, &RandomID{})
	p.RegisterType(TypeRandomPassword, &RandomPassword{})
	p.RegisterType(TypeRandomUUID, &RandomUUID{})
	p.RegisterType(TypeRandomCreature, &RandomCreature{})
	p.RegisterType(TypeSidecar, &Sidecar{})
	p.RegisterType(TypeTemplate, &Template{})

	// Register the custom functions
	p.RegisterFunction("jumppad", customHCLFuncJumppad)
	p.RegisterFunction("docker_ip", customHCLFuncDockerIP)
	p.RegisterFunction("docker_host", customHCLFuncDockerHost)
	p.RegisterFunction("data", customHCLFuncDataFolder)
	p.RegisterFunction("data_with_permissions", customHCLFuncDataFolderWithPermissions)

	return p
}

func customHCLFuncJumppad() (string, error) {
	return utils.JumppadHome(), nil
}

// returns the docker host ip address
func customHCLFuncDockerIP() (string, error) {
	return utils.GetDockerIP(), nil
}

func customHCLFuncDockerHost() (string, error) {
	return utils.GetDockerHost(), nil
}

func customHCLFuncDataFolderWithPermissions(name string, permissions int) (string, error) {
	perms := os.FileMode(permissions)
	return utils.GetDataFolder(name, perms), nil
}

func customHCLFuncDataFolder(name string) (string, error) {
	perms := os.FileMode(0775)
	return utils.GetDataFolder(name, perms), nil
}
