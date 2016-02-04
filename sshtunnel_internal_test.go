package main
import (
	"testing"
	"os"
)

func TestNewSSHTunnelFromHostStrings(t *testing.T) {
	InitLogging(os.Stderr, os.Stderr, os.Stderr, true)
	tunnel := NewSSHTunnelFromHostStrings("knesek@test1.carespeak.com:2222", "9200:localhost:9200")
	assertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	assertEqualsString(t, tunnel.Remote.Host, "localhost")
	assertEqualsInt(t, tunnel.Server.Port, 2222)
	assertEqualsInt(t, tunnel.Remote.Port, 9200)
	assertEqualsInt(t, tunnel.Local.Port, 9200)

	tunnel = NewSSHTunnelFromHostStrings("test1.carespeak.com:2222", "")
	assertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	assertEqualsInt(t, tunnel.Server.Port, 2222)

	tunnel = NewSSHTunnelFromHostStrings("knesek@test1.carespeak.com", "")
	assertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	assertEqualsInt(t, tunnel.Server.Port, 22)

	tunnel = NewSSHTunnelFromHostStrings("test1.carespeak.com", "")
	assertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	assertEqualsInt(t, tunnel.Server.Port, 22)

}

func assertEqualsString(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected %s but got %s", expected, actual)
	}
}

func assertEqualsInt(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected %d but got %d", expected, actual)
	}
}