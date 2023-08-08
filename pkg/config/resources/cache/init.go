package cache

import "github.com/jumppad-labs/jumppad/pkg/config"

// register the types and provider
func init() {
	config.RegisterResource(TypeImageCache, &ImageCache{}, &Provider{})
}
