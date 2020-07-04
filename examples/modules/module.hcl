module "k8s" {
	source = "github.com/shipyard-run/shipyard/examples//single_k3s_cluster"
}

module "consul" {
	source = "../container"
}