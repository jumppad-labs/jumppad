package config

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestProcessesTypes(t *testing.T) {
	vars := map[string]cty.Value{}
	vars["string"] = cty.StringVal("abc")
	vars["number"] = cty.NumberIntVal(23)
	vars["bool"] = cty.BoolVal(true)
	vars["array"] = cty.ListVal(
		[]cty.Value{
			cty.StringVal("abc"),
			cty.StringVal("123"),
		})

	vars["map"] = cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("abc"),
	})

	output := ParseVars(vars)

	require.Equal(t, "abc", output["string"])

	num, _ := output["number"].(*big.Float).Int64()
	require.Equal(t, int64(23), num)

	require.True(t, output["bool"].(bool))

	require.Equal(t, "abc", output["array"].([]interface{})[0])
	require.Equal(t, "123", output["array"].([]interface{})[1])

	require.Equal(t, "abc", output["map"].(map[string]interface{})["foo"])
}
