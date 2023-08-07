package util

import (
	"os"

	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/blueprint"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// setupHCLConfig configures the HCLConfig package and registers the custom types
func SetupHCLConfig(callback hclconfig.ProcessCallback, variables map[string]string, variablesFiles []string) *hclconfig.Parser {
	cfg := hclconfig.DefaultOptions()
	cfg.ParseCallback = callback
	cfg.Variables = variables
	cfg.VariablesFiles = variablesFiles

	p := hclconfig.NewParser(cfg)

	// Register the types
	p.RegisterType(blueprint.TypeBlueprint, &blueprint.Blueprint{})
	p.RegisterType(resources.TypeBook, &resources.Book{})
	p.RegisterType(resources.TypeBuild, &Build{})
	p.RegisterType(resources.TypeCertificateCA, &CertificateCA{})
	p.RegisterType(resources.TypeCertificateLeaf, &CertificateLeaf{})
	p.RegisterType(resources.TypeChapter, &Chapter{})
	p.RegisterType(resources.TypeContainer, &Container{})
	p.RegisterType(resources.TypeCopy, &Copy{})
	p.RegisterType(resources.TypeDocs, &Docs{})
	p.RegisterType(resources.TypeRemoteExec, &RemoteExec{})
	p.RegisterType(resources.TypeHelm, &Helm{})
	p.RegisterType(resources.TypeImageCache, &ImageCache{})
	p.RegisterType(resources.TypeIngress, &Ingress{})
	p.RegisterType(resources.TypeK8sCluster, &K8sCluster{})
	p.RegisterType(resources.TypeK8sConfig, &K8sConfig{})
	p.RegisterType(resources.TypeLocalExec, &LocalExec{})
	p.RegisterType(resources.TypeNetwork, &Network{})
	p.RegisterType(resources.TypeNomadCluster, &NomadCluster{})
	p.RegisterType(resources.TypeNomadJob, &NomadJob{})
	p.RegisterType(resources.TypeRandomNumber, &RandomNumber{})
	p.RegisterType(resources.TypeRandomID, &RandomID{})
	p.RegisterType(resources.TypeRandomPassword, &RandomPassword{})
	p.RegisterType(resources.TypeRandomUUID, &RandomUUID{})
	p.RegisterType(resources.TypeRandomCreature, &RandomCreature{})
	p.RegisterType(resources.TypeSidecar, &Sidecar{})
	p.RegisterType(resources.TypeTask, &Task{})
	p.RegisterType(resources.TypeTemplate, &Template{})

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
