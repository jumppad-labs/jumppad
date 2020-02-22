# Change Log

## version 0.0.0-beta.9

### Updated exec command
Exec command now uses lighter ingress container instead of tools

### Container resource
Add "entrypoint" configuration to set the containers entrypoint

### Docs
New documentation contianer which proxies terminal websockets using Envoy. This was an issue when the docs site was 
running behind a proxy server such as Instruqt.


## version 0.0.0-beta.8

### Updated status command
The status command now pretty prints the resources

```shell
➜ shipyard status

 [ CREATED ] docs.docs
 [ CREATED ] container.tools
 [ CREATED ] helm.consul
 [ CREATED ] ingress.consul-http
 [ CREATED ] k8s_cluster.k3s
 [ CREATED ] network.cloud
 [ CREATED ] container.vscode

Pending: 0 Created: 7 Failed: 0
```

To view status in json format use the `--json` flag

### Bug fixes
* alpine/linux was pulled every time when importing images regardless of local cache
* fix `push` to use new configuration

## version 0.0.0-beta.7

### Helm provider
* Helm provider now uninstalls the chart when deleting a resource, previousuly it was assumed that a chart and cluster would be deleted together
* Added `exec` command to allow the creation of a shell or execution of a command in a container or pod
```
➜ yard-dev exec k8s_cluster.k3s consul-consul-227vz               
parameters: []string{"k8s_cluster.k3s", "consul-consul-227vz"} - command: []string{}
2020-02-19T11:45:28.523Z [DEBUG] Image exists in local cache: image=shipyardrun/tools:latest
2020-02-19T11:45:28.524Z [INFO]  Creating Container: ref=exec-524329800
2020-02-19T11:45:28.641Z [DEBUG] Attaching container to network: ref=exec-524329800 network=network.cloud
/ # ls -las
total 68
     4 drwxr-xr-x    1 root     root          4096 Feb 19 11:38 .
     4 drwxr-xr-x    1 root     root          4096 Feb 19 11:38 ..
     4 drwxr-xr-x    1 root     root          4096 Sep 13 06:21 bin
```
* Added `version` command to return the current application verion
* When restarting from pause, health check all containers, helm charts and k8s config
* Update status command to pretty print status and add `--json` flag for detail

### Bug fixes
* Improve test quality


## version 0.0.0-beta.6

### Bug fixes
* Alpine container not pulled when copying images to cluster
* Health check for pod was only looking at status not ready checks
* Check Network exists before removing
* Upgrade Helm dependency

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
