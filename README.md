# What is `ganda`?

Ganda lets you make HTTP/HTTPS requests to hundreds to millions of URLs in just a few minutes.

It's designed with the Unix philosophy of ["do one thing well"](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well) and wants to be used in a chain of command line pipes to make its requests in parallel. 

By default, it will echo all response bodies to standard out but can optionally save the results of each request in a directory for later analysis.

### Documentation Links

* [Installation](#installation)
* [A Tour of `ganda`](docs/GANDA_TOUR.ipynb)

# Quick Examples

Given a file with a list of IDs in it, you could do something like:

```
cat id_list.txt | awk '{printf "https://api.example.com/resource/%s?apikey=foo\n", $1}' | ganda
```
    
and that will pipe a stream of URLs into `ganda` in the format `https://api.example.com/resource/<ID>?apikey=foo`.

Alternatively, if you have a file full of URLs (one per line), you can just tell `ganda` to run that:

```
ganda my_file_of_urls.txt
```

If you give `ganda` a `-o <directory name>` parameter, it will save the body of each in a separate file inside `<directory name>`.  If you want a single file, just pipe stdout the normal way `... | ganda > result.txt`.

For many more examples, take a look at the [Tour of `ganda`](docs/GANDA_TOUR.ipynb).

# Why use `ganda` over `curl` (or `wget`, `httpie`, `postman-cli`, ...)?

All existing CLI tools for making HTTP requests are oriented around making a single request at a time.  They're great
at starting a pipe of commands (ex: `curl <url> | jq .`) but they're awkward to use beyond a few reqeusts.

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
```
brew tap tednaleid/homebrew-ganda
brew install ganda
```

2. download the appropriate binary from the [releases page](https://github.com/tednaleid/ganda/releases) and put it in your path

3. Compile from source with golang:

```
go install github.com/tednaleid/ganda@latest
```

or, if you have this repo downloaded locally:

```
make install
```

to install in your `$GOPATH/bin` (which you want in your `$PATH`)

# Usage

```
ganda help

NAME:
ganda - make http requests in parallel

USAGE:
<urls/requests on stdout> | ganda [options]

VERSION:
1.0.0

DESCRIPTION:
Pipe urls to ganda over stdout for it to make http requests to each url in parallel.

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
--output value, -o value                               if flag is present, save response bodies to files in the specified directory
--request value, -X value                              HTTP request method to use (default: "GET")
--response-workers value                               number of concurrent workers that will be processing responses, if not specified will be same as --workers (default: 0)
--retry value                                          max number of retries on transient errors (5XX status codes/timeouts) to attempt (default: 0)
--silent, -s                                           if flag is present, omit showing response code for each url only output response bodies (default: false)
--subdir-length value, -S value                        length of hashed subdirectory name to put saved files when using -o; use 2 for > 5k urls, 4 for > 5M urls (default: 0)
--throttle value, -t value                             max number of requests to process per second, default is unlimited (default: -1)
--workers value, -W value                              number of concurrent workers that will be making requests, increase this for more requests in parallel (default: 1)
--help, -h                                             show help (default: false)
--version, -v                                          print the version (default: false)
```