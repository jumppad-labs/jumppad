package config

type Blueprint struct {
	Title       string   `hcl:"Title,optional"`
	Author      string   `hcl:"Author,optional"`
	Slug        string   `hcl:"Slug,optional"`
	Intro       string   `hcl:"Intro,optional"`
	BrowserTabs []string `hcl:"BrowserTabs,optional"`
}
