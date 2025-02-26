module "test" {
  source = "./module"
}

# output "works" {
#   value = module.test.output.works.output.*.key
# }

output "broken" {
  value = module.test.output.broken
}