package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	httpclient "github.com/jumppad-labs/jumppad/pkg/clients/http"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

// ModelProvider handles the lifecycle of Ollama models
type ModelProvider struct {
	config     *OllamaModel
	log        sdk.Logger
	httpClient httpclient.HTTP
}

// Init initializes the provider with the given configuration
func (p *ModelProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*OllamaModel)
	if !ok {
		return fmt.Errorf("unable to cast resource to OllamaModel")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.log = l
	p.httpClient = cli.HTTP

	return nil
}

// Create pulls the specified Ollama model
func (p *ModelProvider) Create(ctx context.Context) error {
	p.log.Info("Pulling Ollama model", "model", p.config.Model)

	// Check if model already exists
	exists, digest, _, err := p.checkModelExists()
	if err != nil {
		return fmt.Errorf("failed to check if model exists: %w", err)
	}

	if exists {
		p.log.Debug("Model already exists", "model", p.config.Model, "digest", digest)
		return nil
	}

	// Pull the model
	pullReq := map[string]any{
		"name":     p.config.Model,
		"insecure": p.config.Insecure,
	}

	reqBody, err := json.Marshal(pullReq)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/pull", ollamaHost()), bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to pull model, status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Read the streaming response
	decoder := json.NewDecoder(resp.Body)
	var lastStatus string
	for {
		var pullResp map[string]any
		if err := decoder.Decode(&pullResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull response: %w", err)
		}

		// Log progress
		if status, ok := pullResp["status"].(string); ok && status != lastStatus {
			lastStatus = status
			p.log.Debug("Pull progress", "status", status)

			// Check for download progress
			if completed, ok := pullResp["completed"].(float64); ok {
				if total, ok := pullResp["total"].(float64); ok {
					percentage := (completed / total) * 100
					p.log.Debug("Download progress", "percentage", fmt.Sprintf("%.1f%%", percentage))
				}
			}
		}

		// Check for errors
		if errMsg, ok := pullResp["error"].(string); ok && errMsg != "" {
			return fmt.Errorf("pull error: %s", errMsg)
		}

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("pull cancelled: %w", ctx.Err())
		default:
		}
	}

	// Get the model info after pull
	_, digest, size, err := p.checkModelExists()
	if err != nil {
		return fmt.Errorf("failed to get model info after pull: %w", err)
	}

	p.config.Digest = digest
	p.config.Size = size

	return nil
}

// Do not remove the model on destroy
func (p *ModelProvider) Destroy(ctx context.Context, force bool) error {
	return nil
}

// Lookup returns the model name for identification
func (p *ModelProvider) Lookup() ([]string, error) {
	return []string{p.config.Model}, nil
}

// Refresh updates the model state from Ollama
func (p *ModelProvider) Refresh(ctx context.Context) error {
	exists, digest, size, err := p.checkModelExists()
	if err != nil {
		return fmt.Errorf("failed to refresh model state: %w", err)
	}

	if !exists {
		return fmt.Errorf("model %s no longer exists", p.config.Model)
	}

	p.config.Digest = digest
	p.config.Size = size

	return nil
}

// Changed checks if the model has changed since creation
func (p *ModelProvider) Changed() (bool, error) {
	// For now, models don't change once pulled
	// In the future, we could check if a newer version is available
	return false, nil
}

// checkModelExists checks if the model already exists in Ollama
func (p *ModelProvider) checkModelExists() (bool, string, int64, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tags", ollamaHost()), nil)
	if err != nil {
		return false, "", 0, fmt.Errorf("failed to create tags request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, "", 0, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, "", 0, fmt.Errorf("failed to list models, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tagsResp struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return false, "", 0, fmt.Errorf("failed to decode tags response: %w", err)
	}

	// Check if our model exists
	for _, model := range tagsResp.Models {
		if model.Name == p.config.Model {
			return true, model.Digest, model.Size, nil
		}
	}

	return false, "", 0, nil
}

func ollamaHost() string {
	// Default to localhost if not set
	host := "http://localhost:11434"
	if h := os.Getenv("OLLAMA_HOST"); h != "" {
		host = h
	}
	return host
}
