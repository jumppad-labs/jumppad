package cache

import (
	"github.com/jumppad-labs/hclconfig/types"
	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
)

// TypeImageCache is the resource string for a ImageCache resource
const TypeImageCache string = "image_cache"

/*
ImageCache defines a structure for creating ImageCache containers

```hcl

	resource "image_cache" "name" {
	  ...
	}

```

@include cache.Registry
@include container.NetworkAttachment

@resource
*/
type ImageCache struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Registries []Registry `hcl:"registry,block" json:"registries,omitempty"`

	/*
	 Attach to the correct network,
	 only when Image is specified.

	 @type []container.NetworkAttachment
	*/
	Networks ctypes.NetworkAttachments `hcl:"network,block" json:"networks,omitempty"`
}
