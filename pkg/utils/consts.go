package utils

import "fmt"

var InvalidBlueprintURIError = fmt.Errorf("inavlid blueprint URI")
var NameExceedsMaxLengthError = fmt.Errorf("name exceeds the max length of 128 characters")
var NameContainsInvalidCharactersError = fmt.Errorf("name contains invalid characters characters must be either a-z, A-Z, 0-9, -, _")

// ImageVolumeName is the name of the volume which stores the images for clusters
const ImageVolumeName string = "images"

// BuildImagePrefix is the default prefix added to any image built by jumppad
const BuildImagePrefix = "jumppad.dev/localcache"

// Name of the Cache resource
const CacheName string = "docker-cache"

// Address of the proxy used for caching docker images
const jumppadProxyAddress string = "http://default.image-cache.local.jmpd.in:3128"

// Addresses to bypass when using a HTTP Proxy
const ProxyBypass string = "localhost,127.0.0.1,cluster.local,jumppad.dev,jumpd.in,svc,consul"

const LocalTLD = "jmpd.in"

const MaxRandomPort = 32767
const MinRandomPort = 30000
