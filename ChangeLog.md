# Change Log

## version 0.0.4

### Container
Allow HTTP health checks to be added to containers


## version 0.0.2

## Docs
Improve UX with documentation, Shipyard now autogenerates the JSON files required for Docusarus, the user
only needs to author the markdown

## Nomad
Implmeneted ability to push a local container to the Nomad cluster
Allow mounting of custom volumes for Nomad clusters

## Build process
Added Chocolatey and Brew, Deb and Rpm instalation sources

## Yard files
Yard files are to be depricated in favor of Markdown files for blueprints.
The information which was previously added to the .yard file can now be added as frontdown in a `README.md` file which resides in the root of your blueprint.

````
---
title: Single Container Example
author: Nic Jackson
slug: single_container
browser_windows: http://localhost:8080
---

# Single Container

This blueprint shows how you can create a single container with Shipyard

```shell
curl localhost:8080
```
````

When the user runs `shipyard run`, this renders to the terminal as:

```
########################################################

Title Single Container Example
Author Nic Jackson

########################################################


1 Single Container
────────────────────────────────────────────────────────────────────────────────

This blueprint shows how you can create a single container with Shipyard

┃ curl localhost:8080
```

### Bugfixes
* Move create shipyard home directory to run or get, this was generating with invalid permissions when using the quick install

## version 0.0.0-beta.12

### Bugfixes
* Ensure downloaded blueprints are stored with their full path
* Increase test coverage

## version 0.0.0-beta.11

### Bugfixes
* Correct log line in Kubernetes controller

## version 0.0.0-beta.11

### Bugfixes
* Fix bug where Kubernetes Config was not returning an error when applying bad config
* Add log output for Helm charts and Kubernetes config

## version 0.0.0-beta.10

### Added no-browser to `run` command
The `run` command now has a `no-browser` flag which supresses any browser windows from opening if they are defined in the stack


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
