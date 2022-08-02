package providers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
	assert "github.com/stretchr/testify/require"
)

func setupTemplate(t *testing.T, filename string) (*config.Template, *Template) {
	tmpl := createTemplate(t, filename)

	return tmpl, NewTemplate(tmpl, hclog.NewNullLogger())
}

func TestTemplateReturnsErrorWhenEmpty(t *testing.T) {
	tmpl, provider := setupTemplate(t, "")
	tmpl.Source = ""

	err := provider.Create()
	assert.Error(t, err)
}

func TestTemplateReturnsErrorWhenCantParse(t *testing.T) {
	tmpl, provider := setupTemplate(t, "")
	tmpl.Source = "template #{{ .Something"

	err := provider.Create()
	assert.Error(t, err)
}

func TestTemplateProcessesCorrectly(t *testing.T) {
	dir := t.TempDir()
	filename := path.Join(dir, "temp.txt")
	ioutil.WriteFile(filename, []byte("mycontent"), os.ModePerm)

	tmpl, provider := setupTemplate(t, filename)

	err := provider.Create()
	assert.NoError(t, err)

	d, err := ioutil.ReadFile(tmpl.Destination)
	assert.NoError(t, err)

	assert.Contains(t, string(d), `data_dir = "something"`)
	assert.Contains(t, string(d), `grpc = 8502`)
	assert.Contains(t, string(d), `grpc = 8500`)
	assert.Contains(t, string(d), `grpc_other = 2000`)
	assert.Contains(t, string(d), `grpc_other = 2001`)
	assert.Contains(t, string(d), `enabled = true`)
	assert.Contains(t, string(d), `not_enabled = false`)
	assert.Contains(t, string(d), `bool_var = true`)
	assert.Contains(t, string(d), `num_var = 13`)
	assert.Contains(t, string(d), `string_var = "Abc"`)

	// check template functions

	assert.Contains(t, string(d), `file_content = "mycontent"`)
	assert.Contains(t, string(d), `quote_string = "Abc"`)
	assert.Contains(t, string(d), `trim_spaces = "with spaces"`)
}

func TestTemplateWriteSourceWhenNoVars(t *testing.T) {
	tmpl, provider := setupTemplate(t, "")
	provider.config.Vars = nil

	err := provider.Create()
	assert.NoError(t, err)

	d, err := ioutil.ReadFile(tmpl.Destination)
	assert.NoError(t, err)

	assert.Contains(t, string(d), `data_dir = "#{{ .Vars.data_dir }}"`)
}

func TestTemplateOverwritesExistingFile(t *testing.T) {
	tmpl, provider := setupTemplate(t, "")

	f, err := os.Create(tmpl.Destination)
	assert.NoError(t, err)
	f.WriteString("Some text in the file")
	f.Close()

	err = provider.Create()
	assert.NoError(t, err)

	d, err := ioutil.ReadFile(tmpl.Destination)
	assert.NoError(t, err)

	assert.Contains(t, string(d), `data_dir = "something"`)
}

func TestTemplateDestroyRemovesDestination(t *testing.T) {
	tmpl, provider := setupTemplate(t, "")

	f, err := os.Create(tmpl.Destination)
	assert.NoError(t, err)
	f.WriteString("test")
	f.Close()

	err = provider.Destroy()
	assert.NoError(t, err)

	assert.NoFileExists(t, tmpl.Destination)
}

func createTemplate(t *testing.T, file string) *config.Template {
	tmpl := fmt.Sprintf(`
variable "bool_var" {
	default = true
}

variable "num_var" {
	default = 13
}

variable "str_var" {
	default = "Abc"
}

variable "str_var_whitespace" {
	default = " with spaces "
}

variable "other_ports" {
	default = [2000,2001]
}

template "fetch_consul_resources" {
  source = <<EOF

	data_dir = "#{{ .Vars.data_dir }}"
	log_level = "DEBUG"
	node_name = "server"

	datacenter = "dc1"
	primary_datacenter = "dc1"

	server = true

	bootstrap_expect = 1
	ui = true
	enabled = #{{ .Vars.enabled }}
	not_enabled = #{{ .Vars.not_enabled }}

	bool_var = #{{ .Vars.bool_var }}
	num_var = #{{ .Vars.num_var }}
	string_var = "#{{ .Vars.string_var }}"

	bind_addr = "{{ GetPrivateInterfaces | attr \"address\" }}"
	client_addr = "0.0.0.0"

	ports {
		#{{ range .Vars.ports }}
	  grpc = #{{ . }}
		#{{ end }}
	}
	
	other_ports {
		#{{ range .Vars.other_ports }}
	  grpc_other = #{{ . }}
		#{{ end }}
	}

	functions {
		file_content = "#{{ file "%s" }}"
		quote_string = #{{ .Vars.string_var | quote }}
		trim_spaces = "#{{ .Vars.string_var_whitespace | trim }}"
	}

	EOF

	destination = "./out.txt"

  vars = {
		other_ports = var.other_ports
		bool_var = var.bool_var
		string_var = var.str_var
		num_var = var.num_var
		data_dir = "something"
		enabled = true
		not_enabled = false
		ports = [8502, 8500]
		port = 8342
		config = {
			a = 1
			ports = ["sfsf","sfsf"]
		}
		string_var_whitespace = var.str_var_whitespace
  }
}`, file)

	conf, _ := config.CreateConfigFromStrings(t, tmpl)
	cr, err := conf.FindResource("template.fetch_consul_resources")
	assert.NoError(t, err)

	return cr.(*config.Template)
}
