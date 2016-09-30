package main

import (
	"os"
	"testing"
)

func TestNewSSHTunnelFromHostStrings(t *testing.T) {
	InitLogging(os.Stderr, os.Stderr, os.Stderr, true)
	tunnel := NewSSHTunnelFromHostStrings("knesek@test1.carespeak.com:2222", "9200:localhost:9200")
	AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	AssertEqualsString(t, tunnel.Remote.Host, "localhost")
	AssertEqualsInt(t, tunnel.Server.Port, 2222)
	AssertEqualsInt(t, tunnel.Remote.Port, 9200)
	AssertEqualsInt(t, tunnel.Local.Port, 9200)

	tunnel = NewSSHTunnelFromHostStrings("test1.carespeak.com:2222", "")
	AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	AssertEqualsInt(t, tunnel.Server.Port, 2222)

	tunnel = NewSSHTunnelFromHostStrings("knesek@test1.carespeak.com", "")
	AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	AssertEqualsInt(t, tunnel.Server.Port, 22)

	tunnel = NewSSHTunnelFromHostStrings("test1.carespeak.com", "")
	AssertEqualsString(t, tunnel.Server.Host, "test1.carespeak.com")
	AssertEqualsInt(t, tunnel.Server.Port, 22)

}
