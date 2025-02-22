package config

import (
	"path"

	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

// registeredTypes is a static list of types that can be used by the parser
// it is the responsibility of the type to register itself with the parser
var registeredTypes map[string]types.Resource

// registeredProvider is a static list of providers that can be used by the parser
// it is the responsibility of the type to register itself with the parser
var registeredProviders map[string]Provider

func init() {
	registeredTypes = map[string]types.Resource{}
	registeredProviders = map[string]Provider{}
}

// RegisterResource allows a resource to register itself with the parser
func RegisterResource(name string, r types.Resource, p sdk.Provider) {
	if r != nil {
		registeredTypes[name] = r
	}

	if p != nil {
		registeredProviders[name] = p
	}
}

// setupHCLConfig configures the HCLConfig package and registers the custom types
func NewParser(callback hclconfig.WalkCallback, variables map[string]string, variablesFiles []string) *hclconfig.Parser {
	cfg := hclconfig.DefaultOptions()

	cfg.Callback = callback
	cfg.VariableEnvPrefix = "JUMPPAD_VAR_"
	cfg.Variables = variables
	cfg.VariablesFiles = variablesFiles
	cfg.ModuleCache = path.Join(utils.JumppadHome(), "modules")

	p := hclconfig.NewParser(cfg)

	// Register the types
	for k, v := range registeredTypes {
		p.RegisterType(k, v)
	}

	// Register the custom functions
	p.RegisterFunction("jumppad", customHCLFuncJumppad)
	p.RegisterFunction("docker_ip", customHCLFuncDockerIP)
	p.RegisterFunction("docker_host", customHCLFuncDockerHost)
	p.RegisterFunction("data", customHCLFuncDataFolder)
	p.RegisterFunction("data_with_permissions", customHCLFuncDataFolderWithPermissions)
	p.RegisterFunction("system", customHCLFuncSystem)
	p.RegisterFunction("exists", customHCLFuncExists)

	return p
}
