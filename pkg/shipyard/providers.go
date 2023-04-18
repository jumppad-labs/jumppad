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
	case resources.TypeCertificateCA:
		return providers.NewCertificateCA(c.(*resources.CertificateCA), cc.Logger)
	case resources.TypeCertificateLeaf:
		return providers.NewCertificateLeaf(c.(*resources.CertificateLeaf), cc.Logger)
	case resources.TypeContainer:
		return providers.NewContainer(c.(*resources.Container), cc.ContainerTasks, cc.HTTP, cc.Logger)
	case resources.TypeCopy:
		return providers.NewCopy(c.(*resources.Copy), cc.Logger)
	case resources.TypeSidecar:
		return providers.NewContainerSidecar(c.(*resources.Sidecar), cc.ContainerTasks, cc.HTTP, cc.Logger)
	case resources.TypeDocs:
		return providers.NewDocs(c.(*resources.Docs), cc.ContainerTasks, cc.Logger)
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
	case resources.TypeLocalExec:
		return providers.NewLocalExec(c.(*resources.LocalExec), cc.Command, cc.Logger)
	//case config.TypeNomadCluster:
	//	return providers.NewNomadCluster(c.(*config.NomadCluster), cc.ContainerTasks, cc.Nomad, cc.Logger)
	//case config.TypeNomadJob:
	//	return providers.NewNomadJob(c.(*config.NomadJob), cc.Nomad, cc.Logger)
	case resources.TypeNetwork:
		return providers.NewNetwork(c.(*resources.Network), cc.Docker, cc.Logger)
	case types.TypeOutput:
		return providers.NewNull(c.Metadata(), cc.Logger)
	case types.TypeModule:
		return providers.NewNull(c.Metadata(), cc.Logger)
	case resources.TypeRemoteExec:
		return providers.NewRemoteExec(c.(*resources.RemoteExec), cc.ContainerTasks, cc.Logger)
	case resources.TypeTemplate:
		return providers.NewTemplate(c.(*resources.Template), cc.Logger)
	}

	return nil
}
