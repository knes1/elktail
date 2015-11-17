# elktail

Elktail is a command line utility to tail EL (elasticsearch, logstash) logs written in Go.

# Download

You can download pre-release version from the [Releases](https://github.com/knes1/elktail/releases) section.

# Usage

If `elktail` is invoked without any parameters, it will attempt to connect to ES instance at `localhost:9200` and tail the logs in the latest logstash index (index that matches pattern `logstash-[0-9].*`), displaying the contents of `message` field. If your logstash logs do not have `message` field, you can change the output format using -f parameter. For example:

`elktail -f '%@timestamp %log'`

Elktail also supports ES query string searches as the argument. For example, in order to tail logs from host `myhost.example.com` that have log level of ERROR you could do the following:

`elktail host:myhost.example.com AND level:error`

Here's the list of all supported parameters:

<pre>
Usage of elktail:
  -f format
    	message format for the entries - field names are referenced using % sign, for example '%@timestamp %message' (default "%message")
  -help
    	print out help message
  -i pattern
    	index pattern - elktail will attempt to tail only the latest of logstash's indexes matched by the pattern (default "logstash-[0-9].*")
  -l list the results once
    	just list the results once, do not follow
  -n number of entries
    	number of entries fetched intially (default 50)
  -t timestamp field
    	timestamp field name used for tailing entries (default "@timestamp")
  -url ElasticSearch URL
    	ElasticSearch URL (default "http://127.0.0.1:9200")
  -v	enable verbose output
  -vv
    	enable even more verbose output
</pre>
