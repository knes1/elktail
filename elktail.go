package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/olivere/elastic.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/codegangsta/cli"
	"net/url"
)

//
// Structure that holds data necessary to perform tailing.
//
type Tail struct {
	client          *elastic.Client  //elastic search client that we'll use to contact EL
	queryDefinition *QueryDefinition //structure containing query definition and formatting
	index           string           //latest logstash index name, we will tail this index
	lastTimeStamp   string           //timestamp of the last result
}

// Regexp for parsing out format fields
var formatRegexp = regexp.MustCompile("%[A-Za-z0-9@_-]+")

// Create a new Tailer using configuration
func NewTail(configuration *Configuration) *Tail {
	tail := new(Tail)

	var client *elastic.Client
	var err error
	var url = configuration.SearchTarget.Url;
	if configuration.SearchTarget.Url != "" {
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
		Error.Fatalf("Could not connect Elasticsearch client to %s: %s.", configuration.SearchTarget.Url, err)
	}
	tail.client = client

	tail.queryDefinition = &configuration.QueryDefinition

	indices, err := tail.client.IndexNames()
	if err != nil {
		Error.Fatalln("Could not fetch available indices.", err)
	}

	tail.index = tail.findLastIndex(indices, configuration.SearchTarget.IndexPattern)
	return tail
}

// Start the tailer
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
			Size(9000). //TODO: needs rewrite this using scrolling, as this implementation may loose entries if there's more than 9K entries per sleep period
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

// Process the results (e.g. prints them out based on configured format)
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
		t.lastTimeStamp = entry["@timestamp"].(string)
		t.printResult(entry)
	}
}

// Print result according to format
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

	config := new(Configuration)
	app := cli.NewApp()
	app.Name = "elktail"
	app.Usage = "utility for tailing Logstash logs stored in ElasticSearch"
	app.HideHelp = true
	app.Version = VERSION
	app.ArgsUsage = "[query-string]"
	app.Flags = config.Flags()
	app.Action = func(c *cli.Context) {

		if (c.IsSet("help")) {
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
		if (!IsConfigRelevantFlagSet(c)) {
			loadedConfig, err := LoadDefault()
			if (err != nil) {
				Info.Printf("Failed to find or open previous default configuration: %s\n", err)
			} else {
				Info.Printf("Loaded previous config and connecting to host %s.\n", loadedConfig.SearchTarget.Url)
				if (config.MoreVerbose) {
					confJs, _ := json.MarshalIndent(loadedConfig, "", "  ")
					Trace.Println("Loaded config:")
					Trace.Println(string(confJs))
				}
				config = loadedConfig
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
			elurl, err := url.Parse(config.SearchTarget.Url);
			if (err != nil) {
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

		tail.Start(!config.ListOnly, config.InitialEntries)
	}

	app.Run(os.Args)

}

func readPasswd() string {
	bytePassword, err := terminal.ReadPassword(0)
	if err != nil {
		Error.Fatalln("Failed to read password.")
	}
	fmt.Println()
	return string(bytePassword)
}