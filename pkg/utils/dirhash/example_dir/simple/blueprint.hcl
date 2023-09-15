resource "blueprint" "container" {
  title       = "Simple container example"
  author      = "Nic Jackson<jackson.nic@gmail.com>"
  slug        = "container"
  description = <<-EOF
    This is the description for the blueprint, it can contain
    markdown such as:

    ### Column headings
    And things like bulleted lists

    * one
    * two
    * three

    #### Example Usage
    It is also possible to incude code blocks

    ```hcl
    module "one" {
      source = "./subfolder"

      variables = {
        nodes = 1
      }
    }
    ```
  EOF
}