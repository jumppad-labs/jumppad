# Change Log

## Version 0.0.0-beta.5

### Introduce taint command and the ability to re-create resources.

Resources can now be tainted using the command `shipyard taint [type] [name]`

When a resource is marked as tained the next run of `shipyard run` will destroy the resource and re-create it.
This feature is especailly useful when building blueprints, often you require a change to a particular container you run `shipyard destroy`
to destroy the stack and then `shipyard run` to re-create. Now it is possible to destroy only the affected resource with `shipyard taint`.

### Change behaviour when processing folders

Previously `shipyard run` would recurse into folders, this behaviour causes problems when the sub-folders contain `*.hcl` files which are not
Shipyard resources. `shipyard run` now only process the top level folder. Sub folder support will be added when we add the `module` feature.

### Improve handling for failed resources

Resources which fail to create can now be retired by re-running `shipyard run`, any depended resources which were not created due to the failure
will also be created when the command is run.