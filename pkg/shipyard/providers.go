package shipyard

import (
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/providers"
)

// generateProviderImpl returns providers grouped together in order of execution
func generateProviderImpl(c types.Resource, cc *clients.Clients) providers.Provider {
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
	case resources.TypeHelm:
		return providers.NewHelm(c.(*resources.Helm), cc.Kubernetes, cc.Helm, cc.Getter, cc.Logger)
	case resources.TypeIngress:
		return providers.NewIngress(c.(*resources.Ingress), cc.ContainerTasks, cc.Connector, cc.Logger)
	case resources.TypeImageCache:
		return providers.NewImageCache(c.(*resources.ImageCache), cc.ContainerTasks, cc.HTTP, cc.Logger)
	case resources.TypeK8sCluster:
		return providers.NewK8sCluster(c.(*resources.K8sCluster), cc.ContainerTasks, cc.Kubernetes, cc.HTTP, cc.Connector, cc.Logger)
	case resources.TypeK8sConfig:
		return providers.NewK8sConfig(c.(*resources.K8sConfig), cc.Kubernetes, cc.Logger)
	//case config.TypeNomadCluster:
	//	return providers.NewNomadCluster(c.(*config.NomadCluster), cc.ContainerTasks, cc.Nomad, cc.Logger)
	//case config.TypeNomadIngress:
	//	return providers.NewNomadIngress(c.(*config.NomadIngress), cc.ContainerTasks, cc.Logger)
	//case config.TypeNomadJob:
	//	return providers.NewNomadJob(c.(*config.NomadJob), cc.Nomad, cc.Logger)
	case resources.TypeNetwork:
		return providers.NewNetwork(c.(*resources.Network), cc.Docker, cc.Logger)
	case types.TypeOutput:
		return providers.NewNull(c.Metadata(), cc.Logger)
	case types.TypeModule:
		return providers.NewNull(c.Metadata(), cc.Logger)
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
