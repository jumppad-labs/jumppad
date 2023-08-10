package images

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupImageLogTests(t *testing.T, data string) string {
	f, err := ioutil.TempFile("", "*.cache")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	f.WriteString(data)

	return f.Name()
}

func TestImageLogRead(t *testing.T) {
	fn := setupImageLogTests(t, "consul:latest,Docker\nvault:latest,Vagrant\nomad:latest,Docker")
	defer os.Remove(fn)

	i := NewImageFileLog(fn)
	list, err := i.Read(ImageTypeDocker)

	assert.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestImageWrite(t *testing.T) {
	fn := setupImageLogTests(t, "")
	defer os.Remove(fn)

	i := NewImageFileLog(fn)
	i.Log("consul:latest", ImageTypeDocker)

	list, err := i.Read(ImageTypeDocker)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestImageWriteOnlyNew(t *testing.T) {
	fn := setupImageLogTests(t, "consul:latest,Docker\n")
	defer os.Remove(fn)

	i := NewImageFileLog(fn)
	err := i.Log("consul:latest", ImageTypeDocker)

	assert.NoError(t, err)

	list, err := i.Read(ImageTypeDocker)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestImageWriteDeletesCache(t *testing.T) {
	fn := setupImageLogTests(t, "consul:latest,Docker\n")

	i := NewImageFileLog(fn)
	err := i.Clear()
	assert.NoError(t, err)

	assert.NoFileExists(t, fn)
}
