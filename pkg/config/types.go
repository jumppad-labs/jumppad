package config

var RegisteredTypes = []ResourceType{
	TypeCertificateCA,
	TypeCertificateLeaf,
	TypeContainer,
	TypeContainerIngress,
	TypeCopy,
	TypeDocs,
	TypeExecLocal,
	TypeExecRemote,
	TypeHelm,
	TypeIngress,
	TypeImageCache,
	TypeK8sCluster,
	TypeK8sConfig,
	TypeK8sIngress,
	TypeLegacyIngress,
	TypeModule,
	TypeNetwork,
	TypeNomadCluster,
	TypeNomadIngress,
	TypeNomadJob,
	TypeOutput,
	TypeSidecar,
	TypeTemplate,
	TypeVariable,
}

func isRegisteredType(t ResourceType) bool {
	for _, rt := range RegisteredTypes {
		if t == rt {
			return true
		}
	}

	return false
}
