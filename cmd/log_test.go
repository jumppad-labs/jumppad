package cmd

import (
	"testing"

	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupLog(t *testing.T) (*cobra.Command, *mocks.MockDocker) {
	// setup the statefile
	t.Cleanup(setupState(logState))

	md := &mocks.MockDocker{}
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	lc := newLogCmd(nil, md)

	return lc, md
}

func TestLogWithAllCallsDockerLog(t *testing.T) {
	lc, md := setupLog(t)

	// call the command
	err := lc.Execute()
	require.NoError(t, err)

	// check that the docker client was called
	md.AssertCalled(t, "ContainerLogs", mock.Anything, "consul.container.shipyard.run", mock.Anything)
	md.AssertCalled(t, "ContainerLogs", mock.Anything, "docker-cache.container.shipyard.run", mock.Anything)
}

var logState = `
{
 "resources": [
    {
      "name": "docker-cache",
      "type": "image_cache",
      "status": "applied",
      "depends_on": [
        "network.onprem"
      ],
      "networks": [
        "network.onprem"
      ]
    },
    {
      "name": "consul_config",
      "type": "template",
      "status": "applied",
      "source": "data_dir = \"#{{ .Vars.data_dir }}\"\nlog_level = \"DEBUG\"\n\ndatacenter = \"dc1\"\nprimary_datacenter = \"dc1\"\n\nserver = true\n\nbootstrap_expect = 1\nui = true\n\nbind_addr = \"0.0.0.0\"\nclient_addr = \"0.0.0.0\"\nadvertise_addr = \"10.6.0.200\"\n\nports {\n  grpc = 8502\n}\n\nconnect {\n  enabled = true\n}\n",
      "destination": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config/consul.hcl",
      "vars": {
        "data_dir": "/tmp"
      }
    },
    {
      "name": "consul_disabled",
      "type": "container",
      "status": "disabled",
      "disabled": true,
      "image": {
        "name": "consul:1.8.1"
      },
      "build": null
    },
    {
      "name": "consul",
      "type": "container",
      "status": "applied",
      "depends_on": [
        "network.onprem",
        "template.consul_config"
      ],
      "depends": [
        "template.consul_config"
      ],
      "networks": [
        {
          "name": "network.onprem",
          "ip_address": "10.6.0.200",
          "aliases": [
            "myalias"
          ]
        }
      ],
      "image": {
        "name": "consul:1.8.1"
      },
      "build": null,
      "command": [
        "consul",
        "agent",
        "-config-file=/config/consul.hcl"
      ],
      "environment": [
        {
          "key": "something",
          "value": "blah blah"
        },
        {
          "key": "foo",
          "value": ""
        },
        {
          "key": "file",
          "value": "this is the contents of a file"
        },
        {
          "key": "abc",
          "value": "123"
        },
        {
          "key": "SHIPYARD_FOLDER",
          "value": "/home/nicj/.shipyard"
        },
        {
          "key": "HOME_FOLDER",
          "value": "/home/nicj"
        }
      ],
      "volumes": [
        {
          "source": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config",
          "destination": "/config"
        }
      ],
      "port_ranges": [
        {
          "local": "8500-8502",
          "enable_host": true
        }
      ],
      "resources": {
        "cpu": 2000,
        "cpu_pin": [
          0,
          1
        ],
        "memory": 1024
      }
    }
	]
}`
