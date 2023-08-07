package jumppad

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/blueprint"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/build"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/providers"
)

type Providers interface {
	GetProvider(c types.Resource) providers.Provider
}

type ProvidersImpl struct {
	clients *Clients
}

func NewProviders(c *Clients) Providers {
	return &ProvidersImpl{c}
}

// generateProviderImpl returns providers grouped together in order of execution
func (p *ProvidersImpl) GetProvider(c types.Resource) providers.Provider {
	switch c.Metadata().Type {
	case blueprint.TypeBlueprint:
		return providers.NewNull(c.Metadata(), p.clients.Logger)
	case resources.TypeBook:
		return providers.NewBook(c.(*resources.Book), p.clients.Logger)
	case build.TypeBuild:
		return build.NewProvider(c.(*build.Build), p.clients.ContainerTasks, p.clients.Logger)
	case resources.TypeCertificateCA:
		return providers.NewCertificateCA(c.(*resources.CertificateCA), p.clients.Logger)
	case resources.TypeCertificateLeaf:
		return providers.NewCertificateLeaf(c.(*resources.CertificateLeaf), p.clients.Logger)
	case resources.TypeChapter:
		return providers.NewChapter(c.(*resources.Chapter), p.clients.Logger)
	case container.TypeContainer:
		return container.NewContainerProvider(c.(*container.Container), p.clients.ContainerTasks, p.clients.HTTP, p.clients.Logger)
	case resources.TypeCopy:
		return providers.NewCopy(c.(*resources.Copy), p.clients.Logger)
	case resources.TypeDocs:
		return providers.NewDocs(c.(*resources.Docs), p.clients.ContainerTasks, p.clients.Logger)
	case resources.TypeHelm:
		return providers.NewHelm(c.(*resources.Helm), p.clients.Kubernetes, p.clients.Helm, p.clients.Getter, p.clients.Logger)
	case resources.TypeIngress:
		return providers.NewIngress(c.(*resources.Ingress), p.clients.ContainerTasks, p.clients.Connector, p.clients.Logger)
	case resources.TypeImageCache:
		return providers.NewImageCache(c.(*resources.ImageCache), p.clients.ContainerTasks, p.clients.HTTP, p.clients.Logger)
	case resources.TypeK8sCluster:
		return providers.NewK8sCluster(c.(*resources.K8sCluster), p.clients.ContainerTasks, p.clients.Kubernetes, p.clients.HTTP, p.clients.Connector, p.clients.Logger)
	case resources.TypeK8sConfig:
		return providers.NewK8sConfig(c.(*resources.K8sConfig), p.clients.Kubernetes, p.clients.Logger)
	case resources.TypeLocalExec:
		return providers.NewLocalExec(c.(*resources.LocalExec), p.clients.Command, p.clients.Logger)
	case resources.TypeNomadCluster:
		return providers.NewNomadCluster(c.(*resources.NomadCluster), p.clients.ContainerTasks, p.clients.Nomad, p.clients.Connector, p.clients.Logger)
	case resources.TypeNomadJob:
		return providers.NewNomadJob(c.(*resources.NomadJob), p.clients.Nomad, p.clients.Logger)
	case resources.TypeNetwork:
		return providers.NewNetwork(c.(*resources.Network), p.clients.Docker, p.clients.Logger)
	case types.TypeOutput:
		return providers.NewNull(c.Metadata(), p.clients.Logger)
	case types.TypeModule:
		return providers.NewNull(c.Metadata(), p.clients.Logger)
	case resources.TypeRemoteExec:
		return providers.NewRemoteExec(c.(*resources.RemoteExec), p.clients.ContainerTasks, p.clients.Logger)
	case resources.TypeRandomNumber:
		return providers.NewRandomNumber(c.(*resources.RandomNumber), p.clients.Logger)
	case resources.TypeRandomID:
		return providers.NewRandomID(c.(*resources.RandomID), p.clients.Logger)
	case resources.TypeRandomPassword:
		return providers.NewRandomPassword(c.(*resources.RandomPassword), p.clients.Logger)
	case resources.TypeRandomUUID:
		return providers.NewRandomUUID(c.(*resources.RandomUUID), p.clients.Logger)
	case resources.TypeRandomCreature:
		return providers.NewRandomCreature(c.(*resources.RandomCreature), p.clients.Logger)
	case container.TypeSidecar:
		return container.NewSidecarProvider(c.(*container.Sidecar), p.clients.ContainerTasks, p.clients.HTTP, p.clients.Logger)
	case resources.TypeTemplate:
		return providers.NewTemplate(c.(*resources.Template), p.clients.Logger)
	case resources.TypeTask:
		return providers.NewTask(c.(*resources.Task), p.clients.Logger)
	case types.TypeVariable:
		return providers.NewNull(c.Metadata(), p.clients.Logger)
	}

	return nil
}
