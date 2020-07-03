module "k8s" {
	source = "github.com/shipyard-run/shipyard/functional_tests/test_fixtures//single_k3s_cluster"
}

module "consul" {
	source = "../container"
}