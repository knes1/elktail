package main
import (
	"flag"
	"runtime"
	"os"
	"encoding/json"
	"io/ioutil"
)

type SearchTarget struct {
	Url          string
	IndexPattern string
}

type QueryDefinition struct {
	Terms          []string
	Format         string
	TimestampField string
}

type Configuration struct {
	SearchTarget    SearchTarget
	QueryDefinition QueryDefinition
	InitialEntries  int
	ListOnly        bool
	User            string
	Password        string  `json:"-"`
	Verbose         bool
	MoreVerbose     bool
	TraceRequests   bool
	Help            bool    `json:"-"`
}

var confDir = ".elktail"
var defaultConfFile = "default.json"

func setupConfiguration() *Configuration {
	config := new(Configuration)

	flag.StringVar(&config.SearchTarget.Url, "url", "http://127.0.0.1:9200", "`ElasticSearch URL`")
	flag.StringVar(&config.QueryDefinition.Format, "f", "%message", "message `format` for the entries - field names are referenced using % sign, for example '%@timestamp %message'")
	flag.StringVar(&config.SearchTarget.IndexPattern, "i", "logstash-[0-9].*", "index `pattern` - elktail will attempt to tail only the latest of logstash's indexes matched by the pattern")
	flag.StringVar(&config.QueryDefinition.TimestampField, "t", "@timestamp", "`timestamp field` name used for tailing entries")
	flag.BoolVar(&config.ListOnly, "l", false, "just `list the results once`, do not follow")
	flag.IntVar(&config.InitialEntries, "n", 50, "`number of entries` fetched intially")
	flag.StringVar(&config.User, "u", "", "`username` for http basic auth, password is supplied over password prompt")
	flag.BoolVar(&config.Verbose, "v", false, "enable verbose output (for debugging)")
	flag.BoolVar(&config.MoreVerbose, "vv", false, "enable even more verbose output (for debugging)")
	flag.BoolVar(&config.TraceRequests, "vvv", false, "also trace requests and responses (for debugigng")
	flag.BoolVar(&config.Help, "help", false, "print out help message")

	flag.Parse()
	config.saveDefault()
	return config
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func (c *Configuration) saveDefault() {
	confDirPath := userHomeDir() + string(os.PathSeparator) + confDir;
	if _, err := os.Stat(confDirPath); os.IsNotExist(err) {
		//conf directory doesn't exist, let's create it
		err := os.Mkdir(confDirPath, 0700)
		if (err != nil) {
			Error.Printf("Failed to create configuration directory %s, %s\n", confDirPath, err)
			return
		}
	}
	confJson, err := json.MarshalIndent(c, "", "  ")
	if (err != nil) {
		Error.Printf("Failed to marshall configuration to json: %s.\n", err)
		return
	}
	var confFile = confDirPath + string(os.PathSeparator) + defaultConfFile;
	err = ioutil.WriteFile(confFile, confJson, 0700)
	if (err != nil) {
		Error.Printf("Failed to save configuration to file %s, %s\n", confFile, err)
		return
	}
}

