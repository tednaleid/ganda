# What is `ganda`?

Ganda lets you make HTTP/HTTPS requests to hundreds to millions of URLs in just a few minutes.

It's designed with the Unix philosophy of ["do one thing well"](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well) and wants to be used in a chain of command line pipes to make its requests in parallel. 

By default, it will echo all response bodies to standard out but can optionally save the results of each request in a directory for later analysis.

### Documentation Links

* [Installation](#installation)
* [A Tour of `ganda`](docs/GANDA_TOUR.ipynb)

# Quick Examples

Given a file with a list of IDs in it, you could do something like:

```bash
cat id_list.txt | awk '{printf "https://api.example.com/resource/%s?key=foo\n", $1}' | ganda
```
    
and that will pipe a stream of URLs into `ganda` in the format `https://api.example.com/resource/<ID>?key=foo`.

Alternatively, if you have a file full of URLs (one per line), you can just tell `ganda` to run that:

```bash
ganda my_file_of_urls.txt
```

If you give `ganda` a `-o <directory name>` parameter, it will save the body of each in a separate file inside `<directory name>`.  If you want a single file, just pipe stdout the normal way `... | ganda > result.txt`.

For many more examples, take a look at the [Tour of `ganda`](docs/GANDA_TOUR.ipynb).

# Why use `ganda` over `curl` (or `wget`, `httpie`, `postman-cli`, ...)?

All existing CLI tools for making HTTP requests are oriented around making a single request at a time.  They're great
at starting a pipe of commands (ex: `curl <url> | jq .`) but they're awkward to use beyond a few requests.

The easiest way to use them is in a bash `for` loop or with something like `xargs`.  This is slow and expensive as they open up a new HTTP connection on every request.  

`ganda` makes many requests in parallel and can maintain context between the request and response.  It's designed to
be used in a pipeline of commands and can be used to make hundreds of thousands of requests in just a few minutes. 

`ganda` will reuse HTTP connections and can specify how many "worker" threads should be used to tightly control parallelism. 

The closest CLIs I've found to `ganda` are load-testing tools like `vegeta`.  They're able to make many requests in
parallel, but they're not designed to only call each URL once, don't maintain context between the request and response,
and don't have the same flexibility in how the response is handled.

`ganda` isn't for load testing, it's for making lots of requests in parallel and processing the results in a pipeline.


# Installation

You currently have 3 options:

1. on MacOS you can install using [homebrew](https://brew.sh/)
```bash
brew tap tednaleid/homebrew-ganda
brew install ganda
```

2. download the appropriate binary from the [releases page](https://github.com/tednaleid/ganda/releases) and put it in your path

3. Compile from source with golang:

```bash
go install github.com/tednaleid/ganda@latest
```

or, if you have this repo downloaded locally:

```bash
make install
```

to install in your `$GOPATH/bin` (which you want in your `$PATH`)

# Usage

```bash
ganda help

NAME:
ganda - make http requests in parallel

USAGE:
<urls/requests on stdout> | ganda [options]

VERSION:
   1.0.2

DESCRIPTION:
   Pipe urls to ganda over stdout to make http requests to each url in parallel.

AUTHOR:
   Ted Naleid <contact@naleid.com>

COMMANDS:
   echoserver  Starts an echo server, --port <port> to override the default port of 8080
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --base-retry-millis value                              the base number of milliseconds to wait before retrying a request, exponential backoff is used for retries (default: 1000)
   --response-body value, -B value                        transforms the body of the response. Values: 'raw' (unchanged), 'base64', 'discard' (don't emit body), 'escaped' (JSON escaped string), 'sha256' (default: raw)
   --connect-timeout-millis value                         number of milliseconds to wait for a connection to be established before timeout (default: 10000)
   --header value, -H value [ --header value, -H value ]  headers to send with every request, can be used multiple times (gzip and keep-alive are already there)
   --insecure, -k                                         if flag is present, skip verification of https certificates (default: false)
   --json-envelope, -J                                    emit result with JSON envelope with url, status, length, and body fields, assumes result is valid json (default: false)
   --color                                                if flag is present, add color to success/warn messages (default: false)
   --output-directory value                               if flag is present, save response bodies to files in the specified directory
   --request value, -X value                              HTTP request method to use (default: "GET")
   --retry value                                          max number of retries on transient errors (5XX status codes/timeouts) to attempt (default: 0)
   --silent, -s                                           if flag is present, omit showing response code for each url only output response bodies (default: false)
   --subdir-length value                                  length of hashed subdirectory name to put saved files when using --output-directory; use 2 for > 5k urls, 4 for > 5M urls (default: 0)
   --throttle-per-second value                            max number of requests to process per second, default is unlimited (default: -1)
   --workers value, -W value                              number of concurrent workers that will be making requests, increase this for more requests in parallel (default: 1)
   --help, -h                                             show help (default: false)
   --version, -v                                          print the version (default: false)
```

# Sample Advanced Use Cases

I've used `ganda` to quickly solve problems that would have otherwise required writing a custom program.  

#### Consuming Events From Kafka and Calling an API

Using `kcat` (https://github.com/edenhill/kcat) (or another Kafka CLI that emits events from Kafka topics), we can consume all the events on a Kafka topic, then use `jq` to pull an identifier out of an event and make an API call for every identifier:

```bash
# get all events on the `my-topic` topic
kcat -C -e -q -b broker.example.com:9092 -t my-topic |\
  # parse the identifier out of the JSON event
  jq -r '.identifier' |\
  # use awk to turn that identifier into an URL
  awk '{ printf "https://api.example.com/item/%s\n", $1}' |\
  # have 5 workers make requests and use a static header with and API key for every request
  ganda -s -W 5 -H "X-Api-Key: my-key" |\
  # parse the `value` out of the response and emit it on stdout
  jq -r '.value'
```

#### Ask for all pages/buckets from an API

Here, we ask for the first 100 pages from an API.  Each returns a JSON list of `status` fields.  Pull those `status` fields out and do a unique count on the distribution.

```bash
# emit a sequence of the numbers from 1 to 100
seq 100 |\
  # use awk to create an url asking for each of the buckets
  awk '{printf "https://example.com/items?type=BUCKET&value=%s\n", $1}' |\
  # use a single ganda worker to ask for each page in sequence
  ganda -s -W 1 -H "X-Api-Key: my-key" |\
  # use jq to parse the resulting json and grab the status
  jq -r '.items[].status' |\ 
  sort |\
  # get a unique count of how many times each status appears
  uniq -c

  41128 DELETED
   6491 INITIATED
  34222 PROCESSED
   5032 ERRORED
```