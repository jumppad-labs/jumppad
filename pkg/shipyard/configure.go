package shipyard

import (
	"os"

	"github.com/shipyard-run/hclconfig"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// setupHCLConfig configures the HCLConfig package and registers the custom types
func setupHCLConfig(callback hclconfig.ProcessCallback, variables map[string]string, variablesFiles []string) *hclconfig.Parser {
	cfg := hclconfig.DefaultOptions()
	cfg.ParseCallback = callback
	cfg.Variables = variables
	cfg.VariablesFiles = variablesFiles

	p := hclconfig.NewParser(cfg)

	// Register the types
	p.RegisterType(resources.TypeCertificateCA, &resources.CertificateCA{})
	p.RegisterType(resources.TypeCertificateLeaf, &resources.CertificateLeaf{})
	p.RegisterType(resources.TypeContainerIngress, &resources.ContainerIngress{})
	p.RegisterType(resources.TypeContainer, &resources.Container{})
	p.RegisterType(resources.TypeDocs, &resources.Docs{})
	p.RegisterType(resources.TypeHelm, &resources.Helm{})
	p.RegisterType(resources.TypeIngress, &resources.Ingress{})
	p.RegisterType(resources.TypeK8sCluster, &resources.K8sCluster{})
	p.RegisterType(resources.TypeNetwork, &resources.Network{})
	p.RegisterType(resources.TypeNomadCluster, &resources.NomadCluster{})
	p.RegisterType(resources.TypeNomadIngress, &resources.NomadIngress{})
	p.RegisterType(resources.TypeSidecar, &resources.Sidecar{})
	p.RegisterType(resources.TypeTemplate, &resources.Template{})
	p.RegisterType(resources.TypeImageCache, &resources.ImageCache{})

	// Register the custom functions
	p.RegisterFunction("docker_ip", customHCLFuncDockerIP)
	p.RegisterFunction("docker_host", customHCLFuncDockerHost)
	p.RegisterFunction("data", customHCLFuncDataFolder)
	p.RegisterFunction("data_with_permissions", customHCLFuncDataFolderWithPermissions)

	return p
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

// generateProviderImpl returns providers grouped together in order of execution
func generateProviderImpl(c types.Resource, cc *Clients) providers.Provider {
	switch c.Metadata().Type {
	case resources.TypeContainer:
		return providers.NewContainer(c.(*resources.Container), cc.ContainerTasks, cc.HTTP, cc.Logger)
	//case resources.TypeContainerIngress:
	//	return providers.NewContainerIngress(c.(*resources.ContainerIngress), cc.ContainerTasks, cc.Logger)
	//case config.TypeSidecar:
	//	return providers.NewContainerSidecar(c.(*config.Sidecar), cc.ContainerTasks, cc.HTTP, cc.Logger)
	//case config.TypeDocs:
	//	return providers.NewDocs(c.(*config.Docs), cc.ContainerTasks, cc.Logger)
	//case config.TypeExecRemote:
	//	return providers.NewRemoteExec(c.(*config.ExecRemote), cc.ContainerTasks, cc.Logger)
	//case config.TypeExecLocal:
	//	return providers.NewExecLocal(c.(*config.ExecLocal), cc.Command, cc.Logger)
	//case config.TypeHelm:
	//	return providers.NewHelm(c.(*config.Helm), cc.Kubernetes, cc.Helm, cc.Getter, cc.Logger)
	//case config.TypeIngress:
	//	return providers.NewIngress(c.(*config.Ingress), cc.ContainerTasks, cc.Connector, cc.Logger)
	case resources.TypeImageCache:
		return providers.NewImageCache(c.(*resources.ImageCache), cc.ContainerTasks, cc.HTTP, cc.Logger)
	//case config.TypeK8sCluster:
	//	return providers.NewK8sCluster(c.(*config.K8sCluster), cc.ContainerTasks, cc.Kubernetes, cc.HTTP, cc.Connector, cc.Logger)
	//case config.TypeK8sConfig:
	//	return providers.NewK8sConfig(c.(*config.K8sConfig), cc.Kubernetes, cc.Logger)
	//case config.TypeK8sIngress:
	//	return providers.NewK8sIngress(c.(*config.K8sIngress), cc.ContainerTasks, cc.Logger)
	//case config.TypeNomadCluster:
	//	return providers.NewNomadCluster(c.(*config.NomadCluster), cc.ContainerTasks, cc.Nomad, cc.Logger)
	//case config.TypeNomadIngress:
	//	return providers.NewNomadIngress(c.(*config.NomadIngress), cc.ContainerTasks, cc.Logger)
	//case config.TypeNomadJob:
	//	return providers.NewNomadJob(c.(*config.NomadJob), cc.Nomad, cc.Logger)
	case resources.TypeNetwork:
		return providers.NewNetwork(c.(*resources.Network), cc.Docker, cc.Logger)
	//case config.TypeOutput:
	//	return providers.NewNull(c.Info(), cc.Logger)
	case resources.TypeTemplate:
		return providers.NewTemplate(c.(*resources.Template), cc.Logger)
		//case config.TypeCertificateCA:
		//	return providers.NewCertificateCA(c.(*config.CertificateCA), cc.Logger)
		//case config.TypeCertificateLeaf:
		//	return providers.NewCertificateLeaf(c.(*config.CertificateLeaf), cc.Logger)
		//case config.TypeCopy:
		//	return providers.NewCopy(c.(*config.Copy), cc.Logger)
	}

	return nil
}
