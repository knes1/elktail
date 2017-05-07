/* Copyright (C) 2016 Krešimir Nesek
 *
 * This software may be modified and distributed under the terms
 * of the MIT license. See the LICENSE file for details.
 */
package main
import (
	"testing"
	"github.io/knes1/elktail/testutils"
)


func TestResolveField(t *testing.T) {
	model1 := map[string]interface{}{
		"@timestamp": 3711,
		"message":   2138,
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

func TestFormatRegexp(t *testing.T) {
	formatString := "%timestamp %message[25] %trace[10] %error"

	match := FormatRegexp.FindAllStringSubmatch(formatString, -1)

	testutils.AssertEqualsString(t, "%timestamp", match[0][0])
	testutils.AssertEqualsString(t, "%timestamp", match[0][1])
	testutils.AssertEqualsString(t, "", match[0][2])

	testutils.AssertEqualsString(t, "%message[25]", match[1][0])
	testutils.AssertEqualsString(t, "%message", match[1][1])
	testutils.AssertEqualsString(t, "25", match[1][2])

	testutils.AssertEqualsString(t, "%trace[10]", match[2][0])
	testutils.AssertEqualsString(t, "%trace", match[2][1])
	testutils.AssertEqualsString(t, "10", match[2][2])

	testutils.AssertEqualsString(t, "%error", match[3][0])
	testutils.AssertEqualsString(t, "%error", match[3][1])
	testutils.AssertEqualsString(t, "", match[3][2])
}
