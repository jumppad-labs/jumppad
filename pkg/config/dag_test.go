package config

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestDoYaLikeDAGs(t *testing.T) {
	l := hclog.New(&hclog.LoggerOptions{Level: hclog.Error})
	log.SetOutput(l.StandardWriter(&hclog.StandardLoggerOptions{}))

	d := dag.AcyclicGraph{}

	cl := NewCluster("cl")
	cl2 := NewCluster("cl2")
	n := NewNetwork("net")

	d.Add(&cl)
	d.Add(&cl2)
	d.Add(&n)

	d.Connect(dag.BasicEdge(&n, &cl))
	d.Connect(dag.BasicEdge(&n, &cl2))

	w := &dag.Walker{
		Callback: func(v dag.Vertex) tfdiags.Diagnostics {
			switch val := v.(type) {
			case *Network:
				fmt.Printf("node %#v\n", val.Name)
			case *Cluster:
				fmt.Printf("node %#v\n", val.Name)
			}

			return nil
		}}

	w.Update(&d)
	w.Wait()

	t.Fail()
}
