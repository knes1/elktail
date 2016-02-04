package elktail
import (
	"runtime"
	"os"
	"encoding/json"
	"io/ioutil"
	"github.com/codegangsta/cli"
)

type SearchTarget struct {
	Url          string
	TunnelUrl	 string	`json:"-"`
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
	Verbose         bool	`json:"-"`
	MoreVerbose     bool	`json:"-"`
	TraceRequests   bool	`json:"-"`
	SSHTunnelParams string
	SaveQuery		bool	`json:"-"`
}

var confDir = ".elktail"
var defaultConfFile = "default.json"
var configRelevantFlags = []string{"url", "f", "i", "l", "t", "n", "u", "ssh"}



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

func (c *Configuration) Copy() *Configuration {
	result := new(Configuration)
	result.SearchTarget.TunnelUrl = c.SearchTarget.TunnelUrl
	result.SearchTarget.Url = c.SearchTarget.Url
	result.SearchTarget.IndexPattern = c.SearchTarget.IndexPattern
	result.QueryDefinition.Format = c.QueryDefinition.Format
	result.QueryDefinition.Terms = make([]string, len(c.QueryDefinition.Terms))
	copy(result.QueryDefinition.Terms, c.QueryDefinition.Terms)
	result.QueryDefinition.TimestampField = c.QueryDefinition.TimestampField
	result.InitialEntries = c.InitialEntries
	result.ListOnly = c.ListOnly
	result.User = c.User
	result.Password = c.Password
	result.Verbose = c.Verbose
	result.MoreVerbose = c.MoreVerbose
	result.TraceRequests = c.TraceRequests
	result.SSHTunnelParams = c.SSHTunnelParams
	return result
}

func (c *Configuration) SaveDefault() {
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
	confFile := confDirPath + string(os.PathSeparator) + defaultConfFile;
	err = ioutil.WriteFile(confFile, confJson, 0700)
	if (err != nil) {
		Error.Printf("Failed to save configuration to file %s, %s\n", confFile, err)
		return
	}
}

func LoadDefault() (conf *Configuration, err error)  {
	confDirPath := userHomeDir() + string(os.PathSeparator) + confDir;
	if _, err := os.Stat(confDirPath); os.IsNotExist(err) {
		//conf directory doesn't exist, let's create it
		err := os.Mkdir(confDirPath, 0700)
		if (err != nil) {
			return nil, err
		}
	}
	confFile := confDirPath + string(os.PathSeparator) + defaultConfFile;
	var config *Configuration
	confBytes, err := ioutil.ReadFile(confFile)
	if (err != nil) {
		return nil, err
	}
	err = json.Unmarshal(confBytes, &config)
	if (err != nil) {
		return nil, err
	}
	return config, nil
}


func (config *Configuration) Flags() []cli.Flag {
	cli.VersionFlag.Usage = "Print the version"
	cli.HelpFlag.Usage = "Show help"
	return []cli.Flag {
		cli.StringFlag{
			Name:        "url",
			Value:       "http://127.0.0.1:9200",
			Usage:       "ElasticSearch URL",
			Destination: &config.SearchTarget.Url,
		},
		cli.StringFlag{
			Name:        "f,format",
			Value:       "%message",
			Usage:       "Message format for the entries - field names are referenced using % sign, for example '%@timestamp %message'",
			Destination: &config.QueryDefinition.Format,
		},
		cli.StringFlag{
			Name:        "i,index-pattern",
			Value:       "logstash-[0-9].*",
			Usage:       "Index pattern - elktail will attempt to tail only the latest of logstash's indexes matched by the pattern",
			Destination: &config.SearchTarget.IndexPattern,
		},
		cli.StringFlag{
			Name:        "t,timestamp-field",
			Value:       "@timestamp",
			Usage:       "Timestamp field name used for tailing entries",
			Destination: &config.QueryDefinition.TimestampField,
		},
		cli.BoolFlag{
			Name:        "l,list-only",
			Usage:       "Just list the results once, do not follow",
			Destination: &config.ListOnly,
		},
		cli.IntFlag{
			Name:        "n",
			Value:       50,
			Usage:       "Number of entries fetched initially",
			Destination: &config.InitialEntries,
		},
		cli.BoolFlag{
			Name:        "s",
			Usage:       "Save query terms - next invocation of elktail (without parameters) will use saved query terms. Any additional terms specified will be applied with AND operator to saved terms",
			Destination: &config.SaveQuery,
		},
		cli.StringFlag{
			Name:        "u",
			Value:       "",
			Usage:       "Username for http basic auth, password is supplied over password prompt",
			Destination: &config.User,
		},
		cli.StringFlag{
			Name:        "ssh,ssh-tunnel",
			Value:       "",
			Usage:       "Use ssh tunnel to connect. Format for the argument is [localport:][user@]sshhost.tld[:sshport]",
			Destination: &config.SSHTunnelParams,
		},
		cli.BoolFlag{
			Name:        "v1",
			Usage:       "Enable verbose output (for debugging)",
			Destination: &config.Verbose,
		},
		cli.BoolFlag{
			Name:        "v2",
			Usage:       "Enable even more verbose output (for debugging)",
			Destination: &config.MoreVerbose,
		},
		cli.BoolFlag{
			Name:        "v3",
			Usage:       "Same as v2 but also trace requests and responses (for debugging)",
			Destination: &config.TraceRequests,
		},
		cli.VersionFlag,
		cli.HelpFlag,
	}
}

func IsConfigRelevantFlagSet(c *cli.Context)  bool {
	for _, flag := range configRelevantFlags {
		if c.IsSet(flag) {
			return true
		}
	}
	return false
}

