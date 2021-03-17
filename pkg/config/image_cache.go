package config

// TypeContainer is the resource string for a Container resource
const TypeImageCache ResourceType = "image_cache"

// Container defines a structure for creating Docker containers
type ImageCache struct {
	// embedded type holding name, etc
	ResourceInfo `mapstructure:",squash"`

	Networks []string `json:"networks" state:"true"` // Attach to the correct network // only when Image is specified
}

func NewImageCache(name string) *ImageCache {
	return &ImageCache{
		ResourceInfo: ResourceInfo{Name: name, Type: TypeImageCache, Status: PendingCreation},
		Networks:     []string{},
	}
}
