package clients

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ImageTypeDocker defines a type for a Docker image
const ImageTypeDocker string = "Docker"

// ImageLog logs machine images to make cleanup possible
type ImageLog interface {
	Log(string, string) error
	Read(string) ([]string, error)
	Clear() error
}

type ImageFileLog struct {
	f string
}

// NewImageFileLog creates an ImageLog which uses a file as the underlying
// Datastore
func NewImageFileLog(file string) *ImageFileLog {
	return &ImageFileLog{file}
}

// Log an image has been downloaded by Shypyard
func (i *ImageFileLog) Log(name, t string) error {
	// check the existing entries do not add if allready in there
	entries, err := i.Read(t)
	if err != nil {
		return err
	}

	for _, v := range entries {
		// found just exit
		if v == name {
			return nil
		}
	}

	f, err := os.OpenFile(i.f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.WriteString(fmt.Sprintf("%s,%s\n", name, t))
	fmt.Println(n)
	return err
}

// Read a list of images which have been downloaded by Shipyard
func (i *ImageFileLog) Read(t string) ([]string, error) {
	f, _ := os.Open(i.f)
	defer f.Close()

	output := []string{}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if parts[1] == t {
			output = append(output, parts[0])
		}
	}

	return output, nil
}

// Clear the list of images
func (i *ImageFileLog) Clear() error {
	return os.Remove(i.f)
}
