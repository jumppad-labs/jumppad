package healthcheck

/*
A `health_check` stanza allows the definition of a health check which must pass before the container is marked as successfully created.
There are three different types of healthcheck `http`, `tcp`, and `exec`, these are not mutually exclusive, it is possible to define more than one health check.

Health checks are executed sequentially, if one health check fails, the following checks are not executed. The execution order is `http`, `tcp`, `exec`.

@example
```

	health_check {
	    timeout = "30s"
	    http {
	      address = "http://localhost:8500/v1/status/leader"
	      success_codes = [200]
	    }

	    http {
	      address = "http://localhost:8500/v1"
	      success_codes = [200]
	    }

	    tcp {
	      address = "localhost:8500"
	    }

	   exec {
	      script = <<-EOF
	        #!/bin/bash

	        curl "http://localhost:9090"
	      EOF
	    }
	  }

```

@type HealthCheck
*/
type HealthCheckContainer struct {
	/*
		The maximum duration to wait before marking the health check as failed. Expressed as a Go duration, e.g. `1s` = 1 second,
		`100ms` = 100 milliseconds.
	*/
	Timeout string `hcl:"timeout" json:"timeout"`
	/*
		HTTP Health Check block defining the address to check and expected status codes.

		Can be specified more than once.
	*/
	HTTP []HealthCheckHTTP `hcl:"http,block" json:"http,omitempty"`
	/*
		TCP Health Check block defining the address to test.

		Can be specified more than once.
	*/
	TCP []HealthCheckTCP `hcl:"tcp,block" json:"tcp,omitempty"`
	/*
		Exec Health Check block defining either a command to run in the current container, or a script to execute.

		Can be specified more than once.
	*/
	Exec []HealthCheckExec `hcl:"exec,block" json:"exec,omitempty"`
}

/*
A HTTP health check executes an HTTP request for the given address and evaluates the response against the expected `success_codes`.
If the reponse matches any of the given codes the check passes.

@example
```

	http {
	  address = "http://localhost:8500/v1/status/leader"
	  method  = "GET"
	  body    = <<-EOF
	    {"test": "123"}
	  EOF
	  headers = {
	    "X-Auth-Token": ["abc123"]
	  }
	  success_codes = [200]
	}

```
*/
type HealthCheckHTTP struct {
	// The URL to check, health check expects a HTTP status code to be returned by the URL in order to pass the health check.
	Address string `hcl:"address" json:"address,omitempty"`
	// HTTP method to use when executing the check
	Method string `hcl:"method,optional" json:"method,omitempty"`
	// HTTP body to send with the request
	Body string `hcl:"body,optional" json:"body,omitempty"`
	// HTTP headers to send with the check
	Headers map[string][]string `hcl:"headers,optional" json:"headers,omitempty"`
	/*
		HTTP status codes returned from the endpoint when called.
		If the returned status code matches any in the array then the health check will pass.
	*/
	SuccessCodes []int `hcl:"success_codes,optional" json:"success_codes,omitempty"`
}

/*
A TCP health check attempts to open a connection to the given address.
If a connection can be opened then the check passes.

@example
```

	tcp {
	  address = "localhost:8500"
	}

```
*/
type HealthCheckTCP struct {
	// The adress to check.
	Address string `hcl:"address" json:"address,omitempty"`
}

/*
Exec health checks allow you to execute a command or script in the current container.
If the command or script receives an exit code 0 the check passes.
*/
type HealthCheckExec struct {
	/*
		The command to execute, the command is run in the target container.

		@example
		```
		exec {
			command = ["pg_isready"]
		}
		```
	*/
	Command []string `hcl:"command,optional" json:"command,omitempty"`
	/*
		A script to execute in the target container, the script is coppied to the container into the /tmp directory and is then executed.

		@example
		```
		exec {
		  script = <<-EOF
		    #!/bin/bash

		    FILE=/etc/resolv.conf
		    if [ -f "$FILE" ]; then
		        echo "$FILE exists."
		    fi
		  EOF
		}
		```
	*/
	Script string `hcl:"script,optional" json:"script,omitempty"`
	// ExitCode to mark a successful check, default 0
	ExitCode int `hcl:"exit_code,optional" json:"exit_code,omitempty"`
}

/*
A `health_check` stanza allows the definition of a health check which must pass before the resource is marked as successfully created.

@example
```

	health_check {
		timeout = "60s"
		pods = [
			"component=server,app=consul",
			"component=client,app=consul"
		]
	}

```
*/
type HealthCheckKubernetes struct {
	/*
		The maximum duration to wait before marking the health check as failed.
		Expressed as a Go duration, e.g. `1s` = 1 second, `100ms` = 100 milliseconds.
	*/
	Timeout string `hcl:"timeout" json:"timeout"`
	/*
		An array of kubernetes selector syntax.
		The healthcheck ensures that all containers defined by the selector are running before passing the healthcheck.
	*/
	Pods []string `hcl:"pods" json:"pods,omitempty"`
}

type HealthCheckNomad struct {
	// Timeout expressed as a go duration i.e 10s
	Timeout string `hcl:"timeout" json:"timeout"`
	//	jobs = ["redis"] // are the Nomad jobs running and healthy
	Jobs []string `hcl:"jobs" json:"jobs,omitempty"`
}
