package main

import (
	"fmt"
	"gopkg.in/olivere/elastic.v2"
	"regexp"
	"encoding/json"
	"time"
	"flag"
	"os"
	"io/ioutil"
	"strings"
)

/*
 * Structure that holds data necessary to perform tailing.
 */
type Tail struct {
	client		 	*elastic.Client	  //elastic search client that we'll use to contact EL
	queryDefinition *QueryDefinition  //structure containing query definition and formatting
	index 			string			  //latest logstash index name, we will tail this index
	lastTimeStamp 	string			  //timestamp of the last result
	lastIds			map[string]bool	  //IDs that were returned in the last search (tail) query
}

var FormatRegexp  = regexp.MustCompile("%[A-Za-z0-9@_-]+")

func NewTail(searchTarget *SearchTarget, queryDef *QueryDefinition) *Tail {
	tail := new(Tail);

	client, err := elastic.NewClient(
		elastic.SetURL(searchTarget.url),
		elastic.SetSniff(false))
	if (err != nil) {
		Error.Fatalln("Error setting up ElasticSearch client.", err)
	}
	tail.client = client

	tail.queryDefinition = queryDef

	indices, err := tail.client.IndexNames()
	if (err != nil) {
		Error.Fatalln("Error fetching available indices.", err)
	}

	tail.index = tail.findLastIndex(indices, searchTarget.indexPattern)
	return tail
}

func (t *Tail) Start(follow bool, initialEntries int) {
	t.lastIds = make(map[string]bool)
	result, err := t.client.Search().
		Index(t.index).
		Sort(t.queryDefinition.timestampField, false).
		Query(t.buildSearchQuery()).
		From(0).Size(initialEntries).
		Do()
	if (err != nil) {
		Error.Fatalln("Error in executing search query.", err)
	}
	t.processResults(result)
	delay := 500 * time.Millisecond
	for ;follow; {
		time.Sleep(delay)
		result, err = t.client.Search().
			Index(t.index).
			Sort(t.queryDefinition.timestampField, false).
			From(0).
			Query(t.buildTimestampFilteredQuery()).
			Do()
		if (err != nil) {
			Error.Fatalln("Error in executing search query.", err)
		}
		t.processResults(result)
		if (result.TotalHits() > 0 && delay > 500 * time.Millisecond) {
			delay = 500 * time.Millisecond
		}  else if (delay <= 2000 * time.Millisecond) {
			delay = delay + 500 * time.Millisecond
		}
	}

}

func (t *Tail) processResults(searchResult *elastic.SearchResult) {
	Trace.Printf("Fetched page of %d results out of %d total.\n", len(searchResult.Hits.Hits), searchResult.Hits.TotalHits)
	hits := searchResult.Hits.Hits
	for i := len(hits) - 1; i >= 0; i-- {
		hit := hits[i]
		var entry map[string]interface{}
		err := json.Unmarshal(*hit.Source, &entry)
		if err != nil {
			Error.Fatalln("Failed parsing ElasticSearch response.", err)
		}
		//fmt.Printf("[%s] %s\n", entry["@timestamp"], entry["message"])
		t.lastTimeStamp = entry["@timestamp"].(string)
		t.printResult(entry)
		/*
		var out bytes.Buffer
		json.Indent(&out, *hit.Source, "", "\t")
		out.WriteTo(os.Stdout)
		*/
		//t.lastIds[entry["@timestamp"].(string)] = true
	}
}

func (t *Tail) printResult(entry map[string]interface{}) {
	Trace.Println("Result: ", entry)
	fields := FormatRegexp.FindAllString(t.queryDefinition.format, -1)
	Trace.Println("Fields: ", entry)
	result := t.queryDefinition.format
	for _, f := range fields {
		value, ok := entry[f[1:len(f)]].(string)
		if (ok) {
			result = strings.Replace(result, f,value, -1)
		}
	}
	fmt.Println(result)
}

func (t *Tail) buildSearchQuery() elastic.Query {
	if (len(t.queryDefinition.terms) > 0) {
		result := strings.Join(t.queryDefinition.terms, " ")
		Info.Printf("Running query string query: %s", result)
		return elastic.NewQueryStringQuery(result)
	} else {
		Info.Printf("Running query match all query.")
		return elastic.NewMatchAllQuery()
	}
}

func (t *Tail) buildTimestampFilteredQuery() elastic.Query {
	query := elastic.NewFilteredQuery(t.buildSearchQuery()).Filter(
		elastic.NewRangeFilter(t.queryDefinition.timestampField).
				IncludeUpper(false).
				Gt(t.lastTimeStamp))
	return query
}

func (t *Tail) findLastIndex(indices []string, indexPattern string) string {
	var lastIdx string
	for _, idx := range indices {
		matched, _ := regexp.MatchString(indexPattern, idx)
		if (matched) {
			if (&lastIdx == nil) {
				lastIdx = idx
			} else if (idx > lastIdx) {
				lastIdx = idx
			}
		}
	}
	return lastIdx
}


func main() {
	config := setupConfiguration()

	if (config.moreVerbose) {
		InitLogging(os.Stderr, os.Stderr, os.Stderr)
	} else if (config.verbose) {
		InitLogging(ioutil.Discard, os.Stderr, os.Stderr)
	} else {
		InitLogging(ioutil.Discard, ioutil.Discard, os.Stderr)
	}

	if (config.help) {
		flag.PrintDefaults()
		os.Exit(0)
	}

	tail := NewTail(&config.searchTarget, &config.queryDefinition)
	tail.Start(!config.listOnly, config.initialEntries)
}

