package main

import (
	"github.io/knes1/elktail/testutils"
	"testing"
)

func TestResolveField(t *testing.T) {
	model1 := map[string]interface{}{
		"@timestamp": 3711,
		"message":    2138,
		"map": map[string]interface{}{
			"test": "test",
		},
	}
	testutils.AssertEqualsString(t, "2138", eval(model1, "message"))
	testutils.AssertEqualsString(t, "test", eval(model1, "map.test"))
	testutils.AssertEqualsString(t, "", eval(model1, "map.foo"))
	testutils.AssertEqualsString(t, "", eval(model1, "bar"))
}

func eval(model interface{}, expr string) string {
	result, _ := EvaluateExpression(model, expr)
	return result
}
