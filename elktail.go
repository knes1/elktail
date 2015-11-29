package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/olivere/elastic.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
	"golang.org/x/crypto/ssh/terminal"
)

/*
 * Structure that holds data necessary to perform tailing.
 */
type Tail struct {
	client          *elastic.Client  //elastic search client that we'll use to contact EL
	queryDefinition *QueryDefinition //structure containing query definition and formatting
	index           string           //latest logstash index name, we will tail this index
	lastTimeStamp   string           //timestamp of the last result
}

var formatRegexp = regexp.MustCompile("%[A-Za-z0-9@_-]+")

func NewTail(configuration *Configuration) *Tail {
	tail := new(Tail)

	var client *elastic.Client
	var err error
	defaultOptions := []elastic.ClientOptionFunc{
		elastic.SetURL(configuration.SearchTarget.Url),
		elastic.SetSniff(false)}

	if configuration.User != "" {
		defaultOptions = append(defaultOptions,
			elastic.SetBasicAuth(configuration.User, configuration.Password))
	}

	if configuration.TraceRequests {
		defaultOptions = append(defaultOptions,
			elastic.SetTraceLog(Trace))
	}

	client, err = elastic.NewClient(defaultOptions...)

	if err != nil {
		Error.Fatalln("Error setting up ElasticSearch client.", err)
	}
	tail.client = client

	tail.queryDefinition = &configuration.QueryDefinition

	indices, err := tail.client.IndexNames()
	if err != nil {
		Error.Fatalln("Error fetching available indices.", err)
	}

	tail.index = tail.findLastIndex(indices, configuration.SearchTarget.IndexPattern)
	return tail
}

func (t *Tail) Start(follow bool, initialEntries int) {
	result, err := t.client.Search().
		Index(t.index).
		Sort(t.queryDefinition.TimestampField, false).
		Query(t.buildSearchQuery()).
		From(0).Size(initialEntries).
		Do()
	if err != nil {
		Error.Fatalln("Error in executing search query.", err)
	}
	t.processResults(result)
	delay := 500 * time.Millisecond
	for follow {
		time.Sleep(delay)
		result, err = t.client.Search().
			Index(t.index).
			Sort(t.queryDefinition.TimestampField, false).
			From(0).
			Size(9000). //TODO: needs rewrite this using scrolling, as this implementation will loose entries if there's more than 9K entries per sleep period
			Query(t.buildTimestampFilteredQuery()).
			Do()
		if err != nil {
			Error.Fatalln("Error in executing search query.", err)
		}
		t.processResults(result)
		if result.TotalHits() > 0 && delay > 500*time.Millisecond {
			delay = 500 * time.Millisecond
		} else if delay <= 2000*time.Millisecond {
			delay = delay + 500*time.Millisecond
		}
	}

}

func (t *Tail) processResults(searchResult *elastic.SearchResult) {
	Info.Printf("Fetched page of %d results out of %d total.\n", len(searchResult.Hits.Hits), searchResult.Hits.TotalHits)
	hits := searchResult.Hits.Hits
	for i := len(hits) - 1; i >= 0; i-- {
		hit := hits[i]
		var entry map[string]interface{}
		err := json.Unmarshal(*hit.Source, &entry)
		if err != nil {
			Error.Fatalln("Failed parsing ElasticSearch response.", err)
		}
		t.lastTimeStamp = entry["@timestamp"].(string)
		t.printResult(entry)
	}
}

func (t *Tail) printResult(entry map[string]interface{}) {
	Trace.Println("Result: ", entry)
	fields := formatRegexp.FindAllString(t.queryDefinition.Format, -1)
	Trace.Println("Fields: ", entry)
	result := t.queryDefinition.Format
	for _, f := range fields {
		value, ok := entry[f[1:len(f)]].(string)
		if ok {
			result = strings.Replace(result, f, value, -1)
		}
	}
	fmt.Println(result)
}

func (t *Tail) buildSearchQuery() elastic.Query {
	if len(t.queryDefinition.Terms) > 0 {
		result := strings.Join(t.queryDefinition.Terms, " ")
		Trace.Printf("Running query string query: %s", result)
		return elastic.NewQueryStringQuery(result)
	} else {
		Trace.Printf("Running query match all query.")
		return elastic.NewMatchAllQuery()
	}
}

func (t *Tail) buildTimestampFilteredQuery() elastic.Query {
	query := elastic.NewFilteredQuery(t.buildSearchQuery()).Filter(
		elastic.NewRangeFilter(t.queryDefinition.TimestampField).
			IncludeUpper(false).
			Gt(t.lastTimeStamp))
	return query
}

func (t *Tail) findLastIndex(indices []string, indexPattern string) string {
	var lastIdx string
	for _, idx := range indices {
		matched, _ := regexp.MatchString(indexPattern, idx)
		if matched {
			if &lastIdx == nil {
				lastIdx = idx
			} else if idx > lastIdx {
				lastIdx = idx
			}
		}
	}
	return lastIdx
}

func main() {
	config := setupConfiguration()

	if config.MoreVerbose || config.TraceRequests {
		InitLogging(os.Stderr, os.Stderr, os.Stderr, true)
	} else if config.Verbose {
		InitLogging(ioutil.Discard, os.Stderr, os.Stderr, false)
	} else {
		InitLogging(ioutil.Discard, ioutil.Discard, os.Stderr, false)
	}

	if config.User != "" {
		fmt.Print("Enter password: ")
		config.Password = readPasswd()
	}

	if config.Help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	tail := NewTail(config)
	tail.Start(!config.ListOnly, config.InitialEntries)
}

func readPasswd() string {
	bytePassword, err := terminal.ReadPassword(0)
	if err != nil {
		Error.Fatalln("Failed to read password.")
	}
	return string(bytePassword)
}