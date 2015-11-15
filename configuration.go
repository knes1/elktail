package main
import (
	"flag"
)

type SearchTarget struct {
	url				string
	indexPattern 	string
}

type QueryDefinition struct {
	terms 			[]string
	format			string
	timestampField 	string
}

type Configuration struct {
	searchTarget	SearchTarget
	queryDefinition QueryDefinition
	initialEntries int
	listOnly		bool
	verbose			bool
	moreVerbose		bool
	help			bool
}

func setupConfiguration() *Configuration {
	config := new(Configuration)
	//config.searchTarget = new(SearchTarget)
	//config.queryDefinition = new(QueryDefinition)

	flag.StringVar(&config.searchTarget.url, "url", "http://127.0.0.1:9200", "`ElasticSearch URL`")
	flag.StringVar(&config.queryDefinition.format, "f", "%message", "message `format` for the entries - field names are referenced using % sign, for example '%@timestamp %message'")
	flag.StringVar(&config.searchTarget.indexPattern, "i", "logstash-[0-9].*", "index `pattern` - elktail will attempt to tail only the latest of logstash's indexes matched by the pattern")
	flag.StringVar(&config.queryDefinition.timestampField, "t", "@timestamp", "`timestamp field` name used for tailing entries")
	flag.BoolVar(&config.listOnly, "l", false, "just `list the results once`, do not follow")
	flag.IntVar(&config.initialEntries, "n", 50, "`number of entries` fetched intially")
	flag.BoolVar(&config.verbose, "v", false, "enable verbose output")
	flag.BoolVar(&config.moreVerbose, "vv", false, "enable even more verbose output")
	flag.BoolVar(&config.help, "help", false, "print out help message")

	flag.Parse()
	config.queryDefinition.terms = flag.Args()
	return config
}