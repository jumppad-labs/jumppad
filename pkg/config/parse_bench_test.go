package config

import "testing"

func BenchmarkConfig(b *testing.B) {
	clear := setup()
	defer clear()

	for n := 0; n < b.N; n++ {
		c, _ := New()
		err := ParseFolder("./examples/single-cluster-k8s", c)
		if err != nil {
			b.Fatal(err)
		}
	}
}
