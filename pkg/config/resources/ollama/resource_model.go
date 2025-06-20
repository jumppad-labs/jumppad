package ollama

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

const TypeOllamaModel = "ollama_model"

type OllamaModel struct {
	types.ResourceBase
	Model    string `json:"model" hcl:"model"`
	Insecure bool   `json:"insecure" hcl:"insecure"`

	// output fields
	Digest string `json:"digest" hcl:"digest"`
	Size   int64  `json:"size" hcl:"size"`
}

func (m *OllamaModel) Process() error {
	cfg, err := config.LoadState()
	if err != nil {
		r, _ := cfg.FindResource(m.Meta.ID)
		if r != nil {
			kstate := r.(*OllamaModel)
			m.Digest = kstate.Digest
			m.Size = kstate.Size
		}
	}

	return nil
}
