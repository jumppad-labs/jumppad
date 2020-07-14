module "k8s" {
	source = "github.com/shipyard-run/shipyard//examples/single_k3s_cluster?ref=testing"

}

module "consul" {
  depends_on = ["module.k8s"]
	source = "../container"
}