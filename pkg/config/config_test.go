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
	c.LocalExecs = []*LocalExec{&LocalExec{}}
	c.RemoteExecs = []*RemoteExec{&RemoteExec{}}

	assert.Equal(t, 10, c.ResourceCount())
}
