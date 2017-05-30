/* Copyright (C) 2016 Krešimir Nesek
 *
 * This software may be modified and distributed under the terms
 * of the MIT license. See the LICENSE file for details.
 */
package main

import (
	"github.com/knes1/elktail/testutils"
	"os"
	"testing"
)

func TestNewSSHTunnelFromHostStrings(t *testing.T) {
	InitLogging(os.Stderr, os.Stderr, os.Stderr, true)
	tunnel := NewSSHTunnelFromHostStrings("knesek@test1.example.com:2222", "9200:localhost:9200")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.example.com")
	testutils.AssertEqualsString(t, tunnel.Remote.Host, "localhost")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 2222)
	testutils.AssertEqualsInt(t, tunnel.Remote.Port, 9200)
	testutils.AssertEqualsInt(t, tunnel.Local.Port, 9200)

	tunnel = NewSSHTunnelFromHostStrings("test1.example.com:2222", "")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.example.com")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 2222)

	tunnel = NewSSHTunnelFromHostStrings("knesek@test1.example.com", "")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.example.com")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 22)

	tunnel = NewSSHTunnelFromHostStrings("test1.example.com", "")
	testutils.AssertEqualsString(t, tunnel.Server.Host, "test1.example.com")
	testutils.AssertEqualsInt(t, tunnel.Server.Port, 22)

}
