/* Copyright (C) 2016 Krešimir Nesek
 *
 * This software may be modified and distributed under the terms
 * of the MIT license. See the LICENSE file for details.
 */
package main

import (
	tu "github.com/knes1/elktail/testutils"
	"testing"
)

func TestExtractDate(t *testing.T) {
	tu.AssertEqualsString(t, "2016-06-17", extractYMDDate("2016-06-17T04:06", "-").Format("2006-01-02"))
	tu.AssertEqualsString(t, "2016-06-17", extractYMDDate("logstash-2016.06.17", ".").Format("2006-01-02"))
}

func TestFindIndicesForDateRange(t *testing.T) {
	indices := [...]string{
		"logstash-2016.06.15",
		"logstash-2016.06.16",
		"logstash-2016.06.17",
		"logstash-2016.06.18",
		"logstash-2016.06.19",
		"logstash-2016.06.20",
	}
	x := findIndicesForDateRange(indices[0:], "logstash.*", "2016-06-16", "2016-06-18")
	t.Log(x)
	tu.AssertEqualsInt(t, 3, len(x))

}

func TestDrainOldEntries(t *testing.T) {
	arr := []displayedEntry{
		{timeStamp: "2016-01-01", id: "1"},
		{timeStamp: "2016-01-02", id: "2"},
		{timeStamp: "2016-01-03", id: "3"},
	}

	drainOldEntries(&arr, "2016-01-02")
	tu.AssertEqualsInt(t, 2, len(arr))

}
