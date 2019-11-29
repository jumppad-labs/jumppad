package shipyard

func setup() {
	//md := &clients.MockDocker{}
}

/*
func TestCorrectlyOrdersElements(t *testing.T) {
	n1 := &Network{Name: "network1"}
	c1 := &Container{Name: "container1", networkRef: n1}
	cl1 := &Cluster{Name: "cluster1", networkRef: n1}
	h1 := &Helm{Name: "helm1", clusterRef: cl1}
	i1 := &Ingress{Name: "ingress1", targetRef: cl1}

	c,_ := New()
	c.Containers = []*Container{c1}
	c.Clusters = []*Cluster{cl1}
	c.Networks = []*Network{n1}
	c.Ingresses = []*Ingress{i1}
	c.HelmCharts = []*Helm{h1}

	// process the config
	oc := generateOrder(c)

	// first element should be a network
	assert.Len(t, oc, 5)

	el1, ok := oc[0].(*Network)
	assert.True(t, ok)
	assert.Equal(t, "network1", el1.Name)

	co1, ok := oc[1].(*Container)
	assert.True(t, ok)
	assert.Equal(t, "container1", co1.Name)

	cll1, ok := oc[2].(*Cluster)
	assert.True(t, ok)
	assert.Equal(t, "cluster1", cll1.Name)

	hl1, ok := oc[3].(*Helm)
	assert.True(t, ok)
	assert.Equal(t, "helm1", hl1.Name)

	in1, ok := oc[4].(*Ingress)
	assert.True(t, ok)
	assert.Equal(t, "ingress1", in1.Name)
}
*/
