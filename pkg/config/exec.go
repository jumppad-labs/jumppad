package config

type Exec struct {
	Name        string
	Command     string   `hcl:"cmd"`
	Arguments   []string `hcl:args,optional`
	Environment []KV     `hcl:"env,block"`
}
