package clients

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpsertChartRepository(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	hc := NewHelm(NewTestLogger(t))
	err := hc.UpsertChartRepository("hashicorp", "https://helm.releases.hashicorp.com")
	require.NoError(t, err)
}
