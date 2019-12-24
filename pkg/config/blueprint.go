package config

import (
	"fmt"
	"net/url"
)

// Blueprint defines a stack blueprint for defining yard configs
type Blueprint struct {
	Title          string   `hcl:"title,optional"`
	Author         string   `hcl:"author,optional"`
	Slug           string   `hcl:"slug,optional"`
	Intro          string   `hcl:"intro,optional"`
	BrowserWindows []string `hcl:"browser_windows,optional"`
	Environment    []KV     `hcl:"env,block"`
}

// Validate the Blueprint and return errors
func (b *Blueprint) Validate() []error {
	errors := make([]error, 0)
	// ensure BrowserWindows are valid URIs
	for _, i := range b.BrowserWindows {
		uri, err := url.Parse(i)
		if err != nil {
			errors = append(
				errors,
				fmt.Errorf("invalid BrowserWindow URI: %s, %s", i, err),
			)
		}

		if uri.String() == "" {
			errors = append(
				errors,
				fmt.Errorf("invalid BrowserWindow URI, uri is empty: %s", i),
			)
		}
	}

	return errors
}
