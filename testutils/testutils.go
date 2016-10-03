/* Copyright (C) 2016 Krešimir Nesek
 *
 * This software may be modified and distributed under the terms
 * of the MIT license. See the LICENSE file for details.
 */
package testutils

import (
	"testing"
)

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

func Fail(t *testing.T, message string) {
	t.Error(message)
	t.Fail()
}