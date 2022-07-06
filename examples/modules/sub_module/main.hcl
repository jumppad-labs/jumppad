module "consul" {
  source = "../../single_file"

  variables = {
    version = "consul:from-mod"
  }
}
