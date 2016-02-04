# Elktail

Elktail is a command line utility to tail ES (elasticsearch, logstash) logs.

# Download

You can download pre-release version from the [Releases](https://github.com/knes1/elktail/releases) section.

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

# Other Parameters


<pre>
   --url "http://127.0.0.1:9200"            ElasticSearch URL
   
   -f, --format "%message"                  Message format for the entries - field names are 
                                            referenced using % sign, for example '%@timestamp %message'
                                            
   -i, --index-pattern "logstash-[0-9].*"   Index pattern - elktail will attempt to tail only the latest 
                                            of logstash's indexes matched by the pattern
                                            
   -t, --timestamp-field "@timestamp"       Timestamp field name used for tailing entries
   
   -l, --list-only                          Just list the results once, do not follow
   
   -n "50"                                  Number of entries fetched initially
   
   -s                                       Save query terms - next invocation of elktail (without parameters)
                                            will use saved query terms. Any additional terms specified will be 
                                            applied with AND operator to saved terms
                                            
   -u                                       Username for http basic auth, password is supplied over password prompt
   
   --ssh, --ssh-tunnel                      Use ssh tunnel to connect. Format for the argument 
                                            is [localport:][user@]sshhost.tld[:sshport]
                                            
   --v1                                     Enable verbose output (for debugging)
   
   --v2                                     Enable even more verbose output (for debugging)
   
   --v3                                     Same as v2 but also trace requests and responses (for debugging)
   
   --version, -v                            Print the version
   
   --help, -h                               Show help
</pre>
