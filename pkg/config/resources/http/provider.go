package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type Provider struct {
	config *HTTP
	log    logger.Logger
	client http.Client
}

func (p *Provider) Init(cfg types.Resource, l logger.Logger) error {
	c, ok := cfg.(*HTTP)
	if !ok {
		return fmt.Errorf("unable to initialize provider, resource is not of type HTTP")
	}

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	p.client = client
	p.config = c
	p.log = l

	return nil
}

func (p *Provider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Metadata().Type), "ref", p.config.Metadata().ID)

	// If a timeout was specified, set it
	if p.config.Timeout != "" {
		timeout, err := time.ParseDuration(p.config.Timeout)
		if err != nil {
			return err
		}

		p.client.Timeout = timeout
	}

	var payload io.Reader
	if p.config.Method == "POST" {
		payload = bytes.NewBuffer([]byte(p.config.Payload))
	}

	// create a http request
	request, err := http.NewRequest(p.config.Method, p.config.URL, payload)
	if err != nil {
		return err
	}

	// add headers
	for k, v := range p.config.Headers {
		request.Header.Add(k, v)
	}

	// make the request
	response, err := p.client.Do(request)
	if err != nil {
		return err
	}

	// read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// set the outputs
	p.config.Status = response.StatusCode
	p.config.Body = string(body)

	return nil
}

func (p *Provider) Destroy() error {
	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *Provider) Refresh() error {
	return nil
}

func (p *Provider) Changed() (bool, error) {
	return false, nil
}
