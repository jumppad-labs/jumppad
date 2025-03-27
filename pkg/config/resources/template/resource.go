package template

import (
	"os"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/zclconf/go-cty/cty"
)

// TypeTemplate is the resource string for a Template resource
const TypeTemplate string = "template"

/*
The Template resource allows the processing of templates, outputing the result as a file.

Templating uses the Handlebars language which is Mustache template language can be found at the following location:
[Mustache templating language details](https://mustache.github.io/mustache.5.html).

```hcl

	resource "template" "name" {
	  ...
	}

```

## Template Functions

The template resource provides custom functions that can be used inside your templates as shown in the example below.

```
resource "template" "consul_config" {

	  source = <<-EOF

	  file_content = "{{ file "./myfile.txt" }}"
	  quote = {{quote something}}
	  trim = {{quote (trim with_whitespace)}}

	  EOF

	  destination = "./consul_config/consul.hcl"
	}

```

### quote [string]

Returns the original string wrapped in quotations, quote can be used with the Go template pipe modifier.

```
// given the string abc

quote "abc" // would return the value "abc"
```

### trim [string]

Removes whitespace such as carrige returns and spaces from the begining and the end of the string, can be used with the Go template pipe modifier.

```
// given the string abc

trim " abc " // would return the value "abc"
```

@example Template using HereDoc
```
resource "template" "consul_config" {

	  source = <<-EOF
	  data_dir = "{{data_dir}}"
	  log_level = "DEBUG"

	  datacenter = "dc1"
	  primary_datacenter = "dc1"

	  server = true

	  bootstrap_expect = 1
	  ui = true

	  bind_addr = "0.0.0.0"
	  client_addr = "0.0.0.0"
	  advertise_addr = "10.6.0.200"

	  ports {
	    grpc = 8502
	  }

	  connect {
	    enabled = true
	  }
	  EOF

	  destination = "./consul_config/consul.hcl"

	  variables = {
	    data_dir = "/tmp"
	  }
	}

````

@example
```
data_dir = "/tmp"
log_level = "DEBUG"

datacenter = "dc1"
primary_datacenter = "dc1"

server = true

bootstrap_expect = 1
ui = true

bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"
advertise_addr = "10.6.0.200"

	ports {
	  grpc = 8502
	}

	connect {
	  enabled = true
	}

```

@example External Files
```
resource "template "consul_config" {

	  source = file("./mytemplate.hcl")
	  destination = "./consul_config/consul.hcl"

	  variables = {
	    data_dir = "/tmp"
	  }
	}

	container "consul" {
	  depends_on = ["template.consul_config"]

	  image   {
	    name = "consul:${variable.consul_version}"
	  }

	  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

	  volume {
	    source      = resource.template.consul_config.destination
	    destination = "/config/consul.hcl"
	  }
	}

```

@example Inline variables
```
resource "template" "consul_config" {

	source = <<-EOF
	data_dir = "${data("test")}"
	log_level = "DEBUG"

	datacenter = "${variable.datacenter}"

	server = ${variable.server}
	EOF

	destination = "./consul_config/consul.hcl"

}
```

@resource
*/
type Template struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		Local path to the template source file

		```hcl
		source = "myfile.txt"
		```

		```hcl
		source = <<-EOF
		My inline content
		EOF
		```
	*/
	Source string `hcl:"source" json:"source"`
	/*
		The destination to write the processed template to.

		```hcl
		destination = "${data("config")}/config.toml"
		```
	*/
	Destination string `hcl:"destination" json:"destination"`
	/*
		Variables to use with the template, variables are available to be used within the template using the `go template` syntax.

		```hcl
		variables = {
		  data_dir = "/tmp"
		}
		```

		Given the above variables, these could be used within a template with the following convention.

		```hcl
		data_dir = "{{data_dir}}"
		```

		@type map[string]any
	*/
	Variables map[string]cty.Value `hcl:"variables,optional" json:"variables,omitempty"`
	// @ignore
	Checksum string `hcl:"checksum,optional" json:"checksum,omitempty"`
}

func (t *Template) Process() error {
	t.Destination = utils.EnsureAbsolute(t.Destination, t.Meta.File)

	// Source can be a file or a template as a string
	// check to see if a valid file before making absolute
	src := t.Source
	absSrc := utils.EnsureAbsolute(src, t.Meta.File)

	if _, err := os.Stat(absSrc); err == nil {
		// file exists
		t.Source = absSrc
	} else {
		// source is a string, replace line endings
		t.Source = strings.Replace(t.Source, "\r\n", "\n", -1)
	}

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.Meta.ID)
		if r != nil {
			kstate := r.(*Template)
			t.Checksum = kstate.Checksum
		}
	}

	return nil
}
