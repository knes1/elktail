# Elktail

Elktail is a command line utility to query and tail ELK (elasticsearch, logstash, kibana) logs. Even though it's powerful, using Kibana's web interface to search and analyse the logs is not always practical. Sometimes you just wish to `tail -f` the logs that you normally view in kibana to see what's happening right now. Elktail allows you to do just that, and more. Tail the logs. Search for errors and specific events on commandline. Pipe the search results to any of the standard unix tools.  Use it in scripts. Redirect the output to a file to effectively download a log from es / kibana etc...

For additional information and usage examples take a look at this post: [Elktail - Command Line Tool for Tailing and Querying ELK Logs](http://knes1.github.io/blog/2016/2016-03-06-elktail-command-line-tool-for-tailing-and-querying-ELK-logs.html)

## Feature Requests

Please feel free to use the [Issue Tracker](https://github.com/knes1/elktail/issues) if you have any feature ideas or requests (and, of course, to report bugs).

## Picking The Right Version

Elktail major versions follow ElasticSearch versions. Here's the table indicating which version Elktail you should for your  ElasticSearch install:

| Elktail       | ElasticSearch |
| ------------- | ------------- |
| v6.x.x        | >= 6.x.x      |
| v5.x.x        | >= 5.x.x      |
| v1.x.x        | 1.x.x, 2.x.x  |

# Installation

#### Install Using Go

Elktail is written in Go language, and if you have [Go installed](https://golang.org/doc/install#install), you can just type in:

`go get github.com/knes1/elktail`

This will automatically download, compile and install the latest version of the app.
After that you should have `elktail` executable in your `$GOPATH/bin`.

#### Install Using Hombrew (OS/X)

To install `elktail` using homebrew packet manager type in the following in the Terminal:

`brew tap knes1/tap`

`brew install elktail`

#### Download Binary

You can also download the executable binary from the [releases page](https://github.com/knes1/elktail/releases).

# Basic Usage

If `elktail` is invoked without any parameters, it will attempt to connect to ES instance at `localhost:9200` and tail the logs in the latest logstash index (index that matches pattern `logstash-[0-9].*`), displaying the contents of `message` field. If your logstash logs do not have `message` field, you can change the output format using -f parameter. For example:

`elktail -f '%@timestamp %log'`

# Connecting Through SSH Tunnel

If ES instance's endpoint is not publicly available over the internet, you can also connect to it through ssh tunnel. For example, if ES instance is installed on elastic.example.com, but port 9200 is firewalled, you can connect through SSH Tunnel:

`elktail -ssh elastic.example.com`

Elktail will connect as current user to elastic.example.com and establish ssh tunnel to port 9200 and then connect to ES through it.
You can also specifiy the ssh user, ssh port and tunnel local port (9199 by default) in the following format: 

`elktail -ssh [localport:][user@]sshhost.tld[:sshport]`


# Elktail Remembers Last Successful Connection

Once you successsfully connect to ES, `elktail` will remember connection parameters for future invocations. You can than invoke `elktail` without any parameters and it will connect to the last ES server it successfully connected to previously.

For example, once you successfully connect to ES using:

`elktail -url "http://elastic.example.com:9200"`

You can then invoke `elktail` without any parameters and it will again attempt to connect to `elastic.example.com:9200`.

Configuration parameters for last successful connection are stored in `~/.elktail/` directory.


# Queries

Elktail also supports ES query string searches as the argument. For example, in order to tail logs from host `myhost.example.com` that have log level of ERROR you could do the following:

`elktail host:myhost.example.com AND level:error`

## Specifying Date Ranges

Elktail supports specifying date range in order to query the logs at specific times. You can specify the date range by using after `-a` and before `-b` options followed by the date. When specifying dates use the following format: YYYY-MM-ddTHH:mm:ss.SSS (e.g 2016-06-17T15:20:00.000). Time part is optional and you can ommit it (e.g. you can leave out seconds, miliseconds, or the whole time part and only specify the date).

Since tailing the logs when using date ranges does not really make sense, when you spacify date range options list-only mode will be implied and following is automatically disabled (e.g. `elktail` will behave as if you specified `-l` option)

#### Date Ranges and Elastic's Logstash Indices

Logstash stores the logs in elasticsearch in one-per-day indices. When specifying date range, `elktail` needs to search through appropriate indices depending on the dates selected. Currently, this will only work if your index name pattern contains dates in YYYY.MM.dd format (which is logstash's default). 

#### Examples

Search for errors after 3PM, April 1st, 2016:
`elktail -a 2016-04-01T15:00 level:error`

Search for errors betweem 1PM and 3PM on July 1st, 2016:
`elktail -a 2016-07-01T13:00 -b 2016-07-01T15:00 level:error`


# Other Options


<pre>
   Options marked with (*) are saved between invocations of the command. Each time you specify an option marked with (*) previously
   stored settings are erased.
   
   --url "http://127.0.0.1:9200"           (*) ElasticSearch URL
   -f, --format "%message"                 (*) Message format for the entries - field names are referenced using % sign,
                                           for example '%@timestamp %message'
                                          
   -i, --index-pattern "logstash-[0-9].*"  (*) Index pattern - elktail will attempt to tail only the latest of logstash's indexes
                                           matched by the pattern
                                          
   -t, --timestamp-field "@timestamp"      (*) Timestamp field name used for tailing entries
   -l, --list-only                         Just list the results once, do not follow
   -n "50"                                 Number of entries fetched initially
   -a, --after                             List results after specified date (example: -a "2016-06-17T15:00")
   -b, --before                            List results before specified date (example: -b "2016-06-17T15:00")
   -s                                      Save query terms - next invocation of elktail (without parameters) will use saved query
                                           terms. Any additional terms specified will be applied with AND operator to saved terms
                                           
   -u                                      (*) Username for http basic auth, password is supplied over password prompt
   --ssh, --ssh-tunnel                     (*) Use ssh tunnel to connect. Format for the 
                                           argument is [localport:][user@]sshhost.tld[:sshport]
                                          
   --v1                                    Enable verbose output (for debugging)
   --v2                                    Enable even more verbose output (for debugging)
   --v3                                    Same as v2 but also trace requests and responses (for debugging)
   --version, -v                           Print the version
   --help, -h                              Show help
   
</pre>
