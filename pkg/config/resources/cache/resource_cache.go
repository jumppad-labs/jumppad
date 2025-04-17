package cache

import (
	ctypes "github.com/instruqt/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/hclconfig/types"
)

// TypeImageCache is the resource string for a ImageCache resource
const TypeImageCache string = "image_cache"

// ImageCache defines a structure for creating ImageCache containers
type ImageCache struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Registries []Registry `hcl:"registry,block" json:"registries,omitempty"`

	Networks ctypes.NetworkAttachments `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified
}
