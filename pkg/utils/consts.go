package utils

import "fmt"

var InvalidBlueprintURIError = fmt.Errorf("Inavlid blueprint URI")
var NameExceedsMaxLengthError = fmt.Errorf("Name exceeds the max length of 128 characters")
var NameContainsInvalidCharactersError = fmt.Errorf("Name contains invalid characters characters must be either a-z, A-Z, 0-9, -, _")

// ImageVolumeName is the name of the volume which stores the images for clusters
const ImageVolumeName string = "images"

// Name of the Cache resource
const CacheResourceName string = "docker-cache"
