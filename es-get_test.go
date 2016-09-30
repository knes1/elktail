package main

import (
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
	AssertEqualsString(t, "2138", eval(model1, "message"))
	AssertEqualsString(t, "test", eval(model1, "map.test"))
	AssertEqualsString(t, "", eval(model1, "map.foo"))
	AssertEqualsString(t, "", eval(model1, "bar"))
}

func eval(model interface{}, expr string) string {
	result, _ := EvaluateExpression(model, expr)
	return result
}

func AssertEqualsString(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected %s but got %s", expected, actual)
	}
}

func AssertEqualsInt(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected %d but got %d", expected, actual)
	}
}
