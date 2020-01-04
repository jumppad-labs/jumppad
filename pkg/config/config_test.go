package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceCount(t *testing.T) {
	c, _ := New()
	c.Docs = &Docs{}
	c.Clusters = []*Cluster{&Cluster{}}
	c.Containers = []*Container{&Container{}}
	c.Networks = []*Network{&Network{}}
	c.HelmCharts = []*Helm{&Helm{}}
	c.K8sConfig = []*K8sConfig{&K8sConfig{}}
	c.Ingresses = []*Ingress{&Ingress{}}
	c.Execs = []*Exec{&Exec{}}

	assert.Equal(t, 9, c.ResourceCount())
}
