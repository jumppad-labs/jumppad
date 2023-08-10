package cache

import "github.com/jumppad-labs/hclconfig/types"

// TypeContainer is the resource string for a Container resource
const TypeImageCache string = "image_cache"

// Container defines a structure for creating Docker containers
type ImageCache struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks []string `json:"networks" state:"true"` // Attach to the correct network // only when Image is specified
}
