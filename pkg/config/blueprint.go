package config

// Blueprint defines a stack blueprint for defining yard configs
type Blueprint struct {
	Title          string   `hcl:"title,optional"`
	Author         string   `hcl:"author,optional"`
	Slug           string   `hcl:"slug,optional"`
	Intro          string   `hcl:"intro,optional"`
	BrowserWindows []string `hcl:"browser_windows,optional"`
	Environment    []KV     `hcl:"env,block"`
}
