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