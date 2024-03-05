#!/bin/bash

plugins="github.com/jumppad-labs/plugin-sdk/example/src"

# generate a plugin init file
cat > plugin.go <<EOF
package jumppad

// add line for each plugin
import a "github.com/jumppad-labs/plugin-sdk/example/src"

func init() {
  var err error

  // register the plugin
  err = config.Register (
		func(name string, r types.Resource, p sdk.Provider) {
			config.RegisterResource(name, r, p)
		},
		func() (sdk.Config, error) {
			return config.LoadState()
		},
	)

  if err != nil {
    panic(err)
  }
}
EOF

# Build
go build -buildmode=plugin -o plugin.so plugin.go