package clients

type Clients struct {
	Docker         Docker
	ContainerTasks ContainerTasks
	Kubernetes     Kubernetes
	Helm           Helm
	HTTP           HTTP
	Nomad          Nomad
	Command        Command
	Logger         Logger
	Getter         Getter
	Browser        System
	ImageLog       ImageLog
	Connector      Connector
	TarGz          *TarGz
}
