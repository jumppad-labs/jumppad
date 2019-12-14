package config

type Blueprint struct {
	Title          string   `hcl:"title,optional"`
	Author         string   `hcl:"author,optional"`
	Slug           string   `hcl:"slug,optional"`
	Intro          string   `hcl:"intro,optional"`
	BrowserWindows []string `hcl:"browser_windows,optional"`
}
