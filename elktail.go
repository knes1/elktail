/*
Copyright (C) 2016 Krešimir Nesek
This software may be modified and distributed under the terms
of the MIT license. See the LICENSE file for details.
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/context"
	"gopkg.in/olivere/elastic.v5"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

//
// Tail is a structure that holds data necessary to perform tailing.
//
type Tail struct {
	client          *elastic.Client  //elastic search client that we'll use to contact EL
	queryDefinition *QueryDefinition //structure containing query definition and formatting
	indices         []string         //indices to search through
	lastTimeStamp   string           //timestamp of the last result
	lastIDs         []displayedEntry //result IDs that we fetched in the last query, used to avoid duplicates when using tailing query time window
	order           bool             //search order - true = ascending (may be reversed in case date-after filtering)
}

type displayedEntry struct {
	timeStamp string
	id        string
}

func (entry *displayedEntry) isBefore(timeStamp string) bool {
	return entry.timeStamp < timeStamp
}

// Regexp for parsing out format fields
var formatRegexp = regexp.MustCompile("%[A-Za-z0-9@_.-]+")

const dateFormatDMY = "2006-01-02"
const dateFormatFull = "2006-01-02T15:04:05.999Z07:00"
const tailingTimeWindow = 500

// NewTail creates a new Tailer using configuration
func NewTail(configuration *Configuration) *Tail {
	tail := new(Tail)

	var client *elastic.Client
	var err error
	var url = configuration.SearchTarget.Url
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
		Trace.Printf("Adding http:// prefix to given url. Url: " + url)
	}

	if !Must(regexp.MatchString(".*:\\d+", url)) && Must(regexp.MatchString("http://[^/]+$", url)) {
		url += ":9200"
		Trace.Printf("No port was specified, adding default port 9200 to given url. Url: " + url)
	}

	//if a tunnel is successfully created, we need to connect to tunnel url (which is localhost on tunnel port)
	if configuration.SearchTarget.TunnelUrl != "" {
		url = configuration.SearchTarget.TunnelUrl
	}

	defaultOptions := []elastic.ClientOptionFunc{
		elastic.SetURL(url),
		elastic.SetSniff(false),
		elastic.SetHealthcheckTimeoutStartup(10 * time.Second),
		elastic.SetHealthcheckTimeout(2 * time.Second),
	}

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
		Error.Fatalf("Could not connect Elasticsearch client to %s: %s.", url, err)
	}
	tail.client = client

	tail.queryDefinition = &configuration.QueryDefinition

	tail.selectIndices(configuration)

	//If we're date filtering on start date, then the sort needs to be ascending
	if configuration.QueryDefinition.AfterDateTime != "" {
		tail.order = true //ascending
	} else {
		tail.order = false //descending
	}
	return tail
}

// Selects appropriate indices in EL based on configuration. This basically means that if query is date filtered,
// then it attempts to select indices in the filtered date range, otherwise it selects the last index.
func (tail *Tail) selectIndices(configuration *Configuration) {
	indices, err := tail.client.IndexNames()
	if err != nil {
		Error.Fatalln("Could not fetch available indices.", err)
	}

	if configuration.QueryDefinition.IsDateTimeFiltered() {
		startDate := configuration.QueryDefinition.AfterDateTime
		endDate := configuration.QueryDefinition.BeforeDateTime
		if startDate == "" && endDate != "" {
			lastIndex := findLastIndex(indices, configuration.SearchTarget.IndexPattern)
			lastIndexDate := extractYMDDate(lastIndex, ".")
			if lastIndexDate.Before(extractYMDDate(endDate, "-")) {
				startDate = lastIndexDate.Format(dateFormatDMY)
			} else {
				startDate = endDate
			}
		}
		if endDate == "" {
			endDate = time.Now().Format(dateFormatDMY)
		}
		tail.indices = findIndicesForDateRange(indices, configuration.SearchTarget.IndexPattern, startDate, endDate)

	} else {
		index := findLastIndex(indices, configuration.SearchTarget.IndexPattern)
		result := [...]string{index}
		tail.indices = result[:]
	}
	Info.Printf("Using indices: %s", tail.indices)
}

// Start the tailer
func (tail *Tail) Start(follow bool, initialEntries int) {
	result, err := tail.initialSearch(initialEntries)
	if err != nil {
		Error.Fatalln("Error in executing search query.", err)
	}
	tail.processResults(result)
	delay := 500 * time.Millisecond
	for follow {
		time.Sleep(delay)
		if tail.lastTimeStamp != "" {
			//we can execute follow up timestamp filtered query only if we fetched at least 1 result in initial query
			result, err = tail.client.Search().
				Index(tail.indices...).
				Sort(tail.queryDefinition.TimestampField, false).
				From(0).
				Size(9000). //TODO: needs rewrite this using scrolling, as this implementation may loose entries if there's more than 9K entries per sleep period
				Query(tail.buildTimestampFilteredQuery()).
				Do(context.Background())
		} else {
			//if lastTimeStamp is not defined we have to repeat the initial search until we get at least 1 result
			result, err = tail.initialSearch(initialEntries)
		}
		if err != nil {
			Error.Fatalln("Error in executing search query.", err)
		}
		tail.processResults(result)

		//Dynamic delay calculation for determining delay between search requests
		if result.TotalHits() > 0 && delay > 500*time.Millisecond {
			delay = 500 * time.Millisecond
		} else if delay <= 2000*time.Millisecond {
			delay = delay + 500*time.Millisecond
		}
	}
}

// Initial search needs to be run until we get at least one result
// in order to fetch the timestamp which we will use in subsequent follow searches
func (tail *Tail) initialSearch(initialEntries int) (*elastic.SearchResult, error) {
	return tail.client.Search().
		Index(tail.indices...).
		Sort(tail.queryDefinition.TimestampField, tail.order).
		Query(tail.buildSearchQuery()).
		From(0).Size(initialEntries).
		Do(context.Background())
}

// Process the results (e.g. prints them out based on configured format)
func (tail *Tail) processResults(searchResult *elastic.SearchResult) {
	Trace.Printf("Fetched page of %d results out of %d total.\n", len(searchResult.Hits.Hits), searchResult.Hits.TotalHits)
	hits := searchResult.Hits.Hits

	// We need to track last N entries that had the timestamp newer than cutoff timestamp. This is done to
	// avoid loosing entries that may have arrived to elasticsearch just as we were executing next query.
	// When tailing, we will
	// issue next query which will be filtered so that timestamps are greater or
	// equal to last timestamp minus tailing time window. Since we are tracking IDs of entries form previous query,
	// we can use the IDs to remove the duplicates. https://github.com/bonovoxly/elktail/issues/11

	if tail.order {
		for i := 0; i < len(hits); i++ {
			hit := hits[i]
			entry := tail.processHit(hit)
			timeStamp := entry[tail.queryDefinition.TimestampField].(string)
			if timeStamp != tail.lastTimeStamp {
				tail.lastTimeStamp = timeStamp
			}
			tail.lastIDs = append(tail.lastIDs, displayedEntry{timeStamp: timeStamp, id: hit.Id})
		}

	} else { //when results are in descending order, we need to process them in reverse
		for i := len(hits) - 1; i >= 0; i-- {
			hit := hits[i]
			entry := tail.processHit(hit)
			timeStamp := entry[tail.queryDefinition.TimestampField].(string)
			if timeStamp != tail.lastTimeStamp {
				tail.lastTimeStamp = timeStamp
			}
			tail.lastIDs = append(tail.lastIDs, displayedEntry{timeStamp: timeStamp, id: hit.Id})
		}
	}
	cutoffTime := formatElasticTimeStamp(parseElasticTimeStamp(tail.lastTimeStamp).Add(-tailingTimeWindow * time.Millisecond))
	drainOldEntries(&tail.lastIDs, cutoffTime)
	//fmt.Print("------------------------------------------------\n")
	//Debugging IDs
	//Info.Printf("CutOff time: %s", cutoffTime)
	//Info.Printf("IDs: %v", tail.lastIDs)
}

func parseElasticTimeStamp(elTimeStamp string) time.Time {
	timeStr, _ := time.Parse(dateFormatFull, elTimeStamp)
	return timeStr
}

func formatElasticTimeStamp(timeStamp time.Time) string {
	return timeStamp.Format(dateFormatFull)
}

func drainOldEntries(entries *[]displayedEntry, cutOffTimestamp string) {
	var i int
	for i = 0; i < len(*entries)-1 && (*entries)[i].timeStamp < cutOffTimestamp; i++ {
	}
	*entries = (*entries)[i:]
}

func (tail *Tail) processHit(hit *elastic.SearchHit) map[string]interface{} {
	var entry map[string]interface{}
	err := json.Unmarshal(*hit.Source, &entry)
	if err != nil {
		Error.Fatalln("Failed parsing ElasticSearch response.", err)
	}
	tail.printResult(entry)
	return entry
}

// Print result according to format
func (tail *Tail) printResult(entry map[string]interface{}) {
	Trace.Println("Result: ", entry)
	fields := formatRegexp.FindAllString(tail.queryDefinition.Format, -1)
	Trace.Println("Fields: ", fields)
	result := tail.queryDefinition.Format
	for _, f := range fields {
		value, _ := EvaluateExpression(entry, f[1:])
		result = strings.Replace(result, f, value, -1)
	}
	if len(result) > 0 {
		fmt.Println(result)
	}
}

func (tail *Tail) buildSearchQuery() elastic.Query {
	var query elastic.Query
	if len(tail.queryDefinition.Terms) > 0 {
		result := strings.Join(tail.queryDefinition.Terms, " ")
		Trace.Printf("Running query string query: %s", result)
		query = elastic.NewQueryStringQuery(result)
	} else {
		Trace.Print("Running query match all query.")
		query = elastic.NewMatchAllQuery()
	}

	if tail.queryDefinition.IsDateTimeFiltered() {
		// we have date filtering turned on, apply filter
		filter := tail.buildDateTimeRangeQuery()
		query = elastic.NewBoolQuery().Filter(query, filter)
	}
	return query
}

//Builds range filter on timestamp field. You should only call this if start or end date times are defined
//in query definition
func (tail *Tail) buildDateTimeRangeQuery() *elastic.RangeQuery {
	filter := elastic.NewRangeQuery(tail.queryDefinition.TimestampField)
	if tail.queryDefinition.AfterDateTime != "" {
		Trace.Printf("Date range query - timestamp after: %s", tail.queryDefinition.AfterDateTime)
		filter = filter.IncludeLower(true).
			From(tail.queryDefinition.AfterDateTime)
	}
	if tail.queryDefinition.BeforeDateTime != "" {
		Trace.Printf("Date range query - timestamp before: %s", tail.queryDefinition.BeforeDateTime)
		filter = filter.IncludeUpper(false).
			To(tail.queryDefinition.BeforeDateTime)
	}
	return filter
}

func (tail *Tail) buildTimestampFilteredQuery() elastic.Query {
	timeStamp := formatElasticTimeStamp(parseElasticTimeStamp(tail.lastTimeStamp).Add(-tailingTimeWindow * time.Millisecond))

	timeStampFilter := elastic.NewRangeQuery(tail.queryDefinition.TimestampField).
		Gte(timeStamp)

	idsToFilter := make([]string, len(tail.lastIDs))
	for i := range tail.lastIDs {
		idsToFilter[i] = tail.lastIDs[i].id
	}

	filter := elastic.NewBoolQuery().Filter(timeStampFilter).MustNot(elastic.NewIdsQuery().Ids(idsToFilter...))
	query := elastic.NewBoolQuery().Filter(tail.buildSearchQuery(), filter)
	return query
}

// Extracts and parses YMD date (year followed by month followed by day) from a given string. YMD values are separated by
// separator character given as argument.
func extractYMDDate(dateStr, separator string) time.Time {
	dateRegexp := regexp.MustCompile(fmt.Sprintf(`(\d{4}%s\d{2}%s\d{2})`, separator, separator))
	match := dateRegexp.FindAllStringSubmatch(dateStr, -1)
	if len(match) == 0 {
		Error.Fatalf("Failed to extract date: %s\n", dateStr)
	}
	result := match[0]
	parsed, err := time.Parse(fmt.Sprintf("2006%s01%s02", separator, separator), result[0])
	if err != nil {
		Error.Fatalf("Failed parsing date: %s", err)
	}
	return parsed
}

func findIndicesForDateRange(indices []string, indexPattern string, startDate string, endDate string) []string {
	start := extractYMDDate(startDate, "-")
	end := extractYMDDate(endDate, "-")
	result := make([]string, 0, len(indices))
	for _, idx := range indices {
		matched, _ := regexp.MatchString(indexPattern, idx)
		if matched {
			idxDate := extractYMDDate(idx, ".")
			if (idxDate.After(start) || idxDate.Equal(start)) && (idxDate.Before(end) || idxDate.Equal(end)) {
				result = append(result, idx)
			}
		}
	}
	return result
}

func findLastIndex(indices []string, indexPattern string) string {
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

	config := new(Configuration)
	app := cli.NewApp()
	app.Name = "elktail"
	app.Usage = "utility for tailing Logstash logs stored in ElasticSearch"
	app.HideHelp = true
	app.Version = VERSION
	app.ArgsUsage = "[query-string]\n   Options marked with (*) are saved between invocations of the command. Each time you specify an option marked with (*) previously stored settings are erased."
	app.Flags = config.Flags()
	app.Action = func(c *cli.Context) {

		if c.IsSet("help") {
			cli.ShowAppHelp(c)
			os.Exit(0)
		}
		if config.MoreVerbose || config.TraceRequests {
			InitLogging(os.Stderr, os.Stderr, os.Stderr, true)
		} else if config.Verbose {
			InitLogging(ioutil.Discard, os.Stderr, os.Stderr, false)
		} else {
			InitLogging(ioutil.Discard, ioutil.Discard, os.Stderr, false)
		}
		if !IsConfigRelevantFlagSet(c) {
			loadedConfig, err := LoadDefault()
			if err != nil {
				Info.Printf("Failed to find or open previous default configuration: %s\n", err)
			} else {
				Info.Printf("Loaded previous config and connecting to host %s.\n", loadedConfig.SearchTarget.Url)
				loadedConfig.CopyConfigRelevantSettingsTo(config)

				if config.MoreVerbose {
					confJs, _ := json.MarshalIndent(loadedConfig, "", "  ")
					Trace.Println("Loaded config:")
					Trace.Println(string(confJs))

					confJs, _ = json.MarshalIndent(loadedConfig, "", "  ")
					Trace.Println("Final (merged) config:")
					Trace.Println(string(confJs))
				}
			}
		}

		if config.User != "" {
			fmt.Print("Enter password: ")
			config.Password = readPasswd()
		}

		//reset TunnelUrl to nothing, we'll point to the tunnel if we actually manage to create it
		config.SearchTarget.TunnelUrl = ""
		if config.SSHTunnelParams != "" {
			//We need to start ssh tunnel and make el client connect to local port at localhost in order to pass
			//traffic through the tunnel
			elurl, err := url.Parse(config.SearchTarget.Url)
			if err != nil {
				Error.Fatalf("Failed to parse hostname/port from given URL: %s\n", config.SearchTarget.Url)
			}
			Trace.Printf("SSHTunnel remote host: %s\n", elurl.Host)

			tunnel := NewSSHTunnelFromHostStrings(config.SSHTunnelParams, elurl.Host)
			//Using the TunnelUrl configuration param, we will signify the client to connect to tunnel
			config.SearchTarget.TunnelUrl = fmt.Sprintf("http://localhost:%d", tunnel.Local.Port)

			Info.Printf("Starting SSH tunnel %d:%s@%s:%d to %s:%d", tunnel.Local.Port, tunnel.Config.User,
				tunnel.Server.Host, tunnel.Server.Port, tunnel.Remote.Host, tunnel.Remote.Port)
			go tunnel.Start()
			Trace.Print("Sleeping for a second until tunnel is established...")
			time.Sleep(1 * time.Second)
		}

		var configToSave *Configuration

		args := c.Args()

		if config.SaveQuery {
			if args.Present() {
				config.QueryDefinition.Terms = []string{args.First()}
				config.QueryDefinition.Terms = append(config.QueryDefinition.Terms, args.Tail()...)
			} else {
				config.QueryDefinition.Terms = []string{}
			}
			configToSave = config.Copy()
			Trace.Printf("Saving query terms. Total terms: %d\n", len(configToSave.QueryDefinition.Terms))
		} else {
			Trace.Printf("Not saving query terms. Total terms: %d\n", len(config.QueryDefinition.Terms))
			configToSave = config.Copy()
			if args.Present() {
				if len(config.QueryDefinition.Terms) > 1 {
					config.QueryDefinition.Terms = append(config.QueryDefinition.Terms, "AND")
					config.QueryDefinition.Terms = append(config.QueryDefinition.Terms, args...)
				} else {
					config.QueryDefinition.Terms = []string{args.First()}
					config.QueryDefinition.Terms = append(config.QueryDefinition.Terms, args.Tail()...)
				}
			}
		}

		tail := NewTail(config)
		//If we don't exit here we can save the defaults
		configToSave.SaveDefault()

		tail.Start(!config.IsListOnly(), config.InitialEntries)
	}

	app.Run(os.Args)

}

// Must is a helper function to avoid boilerplate error handling for regex matches
// this way they may be used in single value context
func Must(result bool, err error) bool {
	if err != nil {
		Error.Panic(err)
	}
	return result
}

// Read password from the console
func readPasswd() string {
	bytePassword, err := terminal.ReadPassword(0)
	if err != nil {
		Error.Fatalln("Failed to read password.")
	}
	fmt.Println()
	return string(bytePassword)
}

// EvaluateExpression Expression evaluation function. It uses map as a model and evaluates expression given as
// the parameter using dot syntax:
// "foo" evaluates to model[foo]
// "foo.bar" evaluates to model[foo][bar]
// If a key given in the expression does not exist in the model, function will return empty string and
// an error.
func EvaluateExpression(model interface{}, fieldExpression string) (string, error) {
	if fieldExpression == "" {
		return fmt.Sprintf("%v", model), nil
	}
	parts := strings.SplitN(fieldExpression, ".", 2)
	expression := parts[0]
	var nextModel interface{}
	modelMap, ok := model.(map[string]interface{})
	if ok {
		value := modelMap[expression]
		if value != nil {
			nextModel = value
		} else {
			return "", fmt.Errorf("Failed to evaluate expression %s on given model (model map does not contain that key?).", fieldExpression)
		}
	} else {
		return "", fmt.Errorf("Model on which %s is to be evaluated is not a map.", fieldExpression)
	}
	nextExpression := ""
	if len(parts) > 1 {
		nextExpression = parts[1]
	}
	return EvaluateExpression(nextModel, nextExpression)
}
