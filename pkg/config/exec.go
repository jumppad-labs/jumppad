package config

type Exec struct {
	Name        string
	Script      string   `hcl:"script,optional"`
	Command     string   `hcl:"cmd,optional"`
	Arguments   []string `hcl:args,optional`
	Environment []KV     `hcl:"env,block"`
}
