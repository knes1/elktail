/* Copyright (C) 2016 Krešimir Nesek
 *
 * This software may be modified and distributed under the terms
 * of the MIT license.  See the LICENSE file for details.
 */

package main
import (
	"runtime"
	"os"
	"encoding/json"
	"io/ioutil"
	"github.com/urfave/cli"
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
	AfterDateTime  string  `json:"-"`
	BeforeDateTime string  `json:"-"`
}

type Configuration struct {
	SearchTarget    SearchTarget
	QueryDefinition QueryDefinition
	InitialEntries  int
	ListOnly        bool	`json:"-"`
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

//When changing this array, make sure to also make appropriate changes in CopyConfigRelevantSettingsTo
var configRelevantFlags = []string{"url", "f", "i", "t", "u", "ssh"}



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

	c.CopyConfigRelevantSettingsTo(result)
	c.CopyNonConfigRelevantSettingsTo(result)

	return result
}

//When making change here make sure configRelevantFlags global var is also changed
func (c *Configuration) CopyConfigRelevantSettingsTo(dest *Configuration) {
	//copy config relevant configuration settings
	dest.SearchTarget.TunnelUrl = c.SearchTarget.TunnelUrl
	dest.SearchTarget.Url = c.SearchTarget.Url
	dest.SearchTarget.IndexPattern = c.SearchTarget.IndexPattern
	dest.QueryDefinition.Format = c.QueryDefinition.Format
	dest.QueryDefinition.Terms = make([]string, len(c.QueryDefinition.Terms))
	copy(dest.QueryDefinition.Terms, c.QueryDefinition.Terms)
	dest.User = c.User
	dest.SSHTunnelParams = c.SSHTunnelParams
}

func (c *Configuration) CopyNonConfigRelevantSettingsTo(dest *Configuration) {
	//copy non-config relevant settings
	dest.QueryDefinition.TimestampField = c.QueryDefinition.TimestampField
	dest.QueryDefinition.AfterDateTime = c.QueryDefinition.AfterDateTime
	dest.QueryDefinition.BeforeDateTime = c.QueryDefinition.BeforeDateTime
	dest.ListOnly = c.ListOnly
	dest.InitialEntries = c.InitialEntries
	dest.Password = c.Password
	dest.Verbose = c.Verbose
	dest.MoreVerbose = c.MoreVerbose
	dest.TraceRequests = c.TraceRequests
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
	cli.VersionFlag = cli.BoolFlag{
	  Name: "print-version, V",
	  Usage: "print only the version",
	}
	cli.HelpFlag = cli.BoolFlag{Name: "halp"}
	return []cli.Flag {
		cli.StringFlag{
			Name:        "url",
			Value:       "http://127.0.0.1:9200",
			Usage:       "(*) ElasticSearch URL",
			Destination: &config.SearchTarget.Url,
		},
		cli.StringFlag{
			Name:        "f,format",
			Value:       "%message",
			Usage:       "(*) Message format for the entries - field names are referenced using % sign, for example '%@timestamp %message'",
			Destination: &config.QueryDefinition.Format,
		},
		cli.StringFlag{
			Name:        "i,index-pattern",
			Value:       "logstash-[0-9].*",
			Usage:       "(*) Index pattern - elktail will attempt to tail only the latest of logstash's indexes matched by the pattern",
			Destination: &config.SearchTarget.IndexPattern,
		},
		cli.StringFlag{
			Name:        "t,timestamp-field",
			Value:       "@timestamp",
			Usage:       "(*) Timestamp field name used for tailing entries",
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
		cli.StringFlag{
			Name:        "a,after",
			Value:       "",
			Usage:       "List results after specified date (example: -a \"2016-06-17T15:00\")",
			Destination: &config.QueryDefinition.AfterDateTime,
		},
		cli.StringFlag{
			Name:        "b,before",
			Value:       "",
			Usage:       "List results before specified date (example: -b \"2016-06-17T15:00\")",
			Destination: &config.QueryDefinition.BeforeDateTime,
		},
		cli.BoolFlag{
			Name:        "s",
			Usage:       "Save query terms - next invocation of elktail (without parameters) will use saved query terms. Any additional terms specified will be applied with AND operator to saved terms",
			Destination: &config.SaveQuery,
		},
		cli.StringFlag{
			Name:        "u",
			Value:       "",
			Usage:       "(*) Username for http basic auth, password is supplied over password prompt",
			Destination: &config.User,
		},
		cli.StringFlag{
			Name:        "ssh,ssh-tunnel",
			Value:       "",
			Usage:       "(*) Use ssh tunnel to connect. Format for the argument is [localport:][user@]sshhost.tld[:sshport]",
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

//Elktail will work in list-only (no follow) mode if appropriate flag is set or if query has date-time filtering enabled
func (c *Configuration) IsListOnly() bool {
	return c.ListOnly || c.QueryDefinition.IsDateTimeFiltered()
}

func (q *QueryDefinition) IsDateTimeFiltered() bool {
	return q.AfterDateTime != "" || q.BeforeDateTime != ""
}

func IsConfigRelevantFlagSet(c *cli.Context) bool {
	for _, flag := range configRelevantFlags {
		if c.IsSet(flag) {
			return true
		}
	}
	return false
}

