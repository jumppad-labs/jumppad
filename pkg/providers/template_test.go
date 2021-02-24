package providers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
)

func setupTemplate(t *testing.T) (*config.Template, *Template) {
	temp := t.TempDir()
	tmpl := createTemplate(filepath.Join(temp, "out.hcl"))

	return tmpl, NewTemplate(tmpl, hclog.NewNullLogger())
}

func TestTemplateReturnsErrorWhenEmpty(t *testing.T) {
	tmpl, provider := setupTemplate(t)
	tmpl.Source = ""

	err := provider.Create()
	assert.Error(t, err)
}

func TestTemplateReturnsErrorWhenCantParse(t *testing.T) {
	tmpl, provider := setupTemplate(t)
	tmpl.Source = "template #{{ .Something"

	err := provider.Create()
	assert.Error(t, err)
}

func TestTemplateProcessesCorrectly(t *testing.T) {
	tmpl, provider := setupTemplate(t)

	err := provider.Create()
	assert.NoError(t, err)

	d, err := ioutil.ReadFile(tmpl.Destination)
	assert.NoError(t, err)

	assert.Contains(t, string(d), `data_dir = "something"`)
}

func TestTemplateDestroyRemovesDestination(t *testing.T) {
	tmpl, provider := setupTemplate(t)

	f, err := os.Create(tmpl.Destination)
	assert.NoError(t, err)
	f.WriteString("test")
	f.Close()

	err = provider.Destroy()
	assert.NoError(t, err)

	assert.NoFileExists(t, tmpl.Destination)
}

func createTemplate(outPath string) *config.Template {
	return &config.Template{
		Source: `
data_dir = "#{{ .Vars.data_dir }}"
log_level = "DEBUG"
node_name = "server"

datacenter = "dc1"
primary_datacenter = "dc1"

server = true

bootstrap_expect = 1
ui = true

bind_addr = "{{ GetPrivateInterfaces | attr \"address\" }}"
client_addr = "0.0.0.0"

ports {
  grpc = 8502
}

connect {
  enabled = true
}`,

		Destination: outPath,

		Vars: map[string]string{
			"data_dir": "something",
		},
	}
}
