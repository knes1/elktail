package main
import (
	"testing"
	"os"
	"github.io/knes1/elktail/testutils"
)

func TestNewSSHTunnelFromHostStrings(t *testing.T) {
	InitLogging(os.Stderr, os.Stderr, os.Stderr, true)
	tunnel := NewSSHTunnelFromHostStrings("knesek@test1.carespeak.com:2222", "9200:localhost:9200")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	testutils.AssertEqualsString(t, tunnel.Remote.Host, "localhost")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 2222)
	testutils.AssertEqualsInt(t, tunnel.Remote.Port, 9200)
	testutils.AssertEqualsInt(t, tunnel.Local.Port, 9200)

	tunnel = NewSSHTunnelFromHostStrings("test1.carespeak.com:2222", "")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 2222)

	tunnel = NewSSHTunnelFromHostStrings("knesek@test1.carespeak.com", "")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 22)

	tunnel = NewSSHTunnelFromHostStrings("test1.carespeak.com", "")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 22)

}

