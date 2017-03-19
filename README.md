# What is `ganda`?

A utility app that you can pipe urls to and it will request them all in parallel.  It optionally also allows saving the results of each request in a directory for later analysis.

Given a file with a list of IDs in it, you could do something like:

    cat id_list.txt | sed 's/\(.*\)/https:\/\/api.example.com\/resource\/\1/' | ./ganda
    
and that will pipe a stream of urls into `ganda` in the format `https://api.example.com/resource/<ID>`.

If you give `ganda` a `-o <directory name>` parameter, it will save the body of each in a subdirectory.

`ganda` is designed to be able to save the results of hundreds of thousands of requests within a few minutes.

# Installing

Compile with golang, use either 

    go get github.com/tednaleid/ganda
    
to install in your `$GOPATH` or clone the repo and 

    go build 
    
to create the `ganda` binary and then copy it somewhere into your path.

If you have docker installed, you can use `./build.sh` to download a golang container and compile it into a `ganda` binary usable on linux (but not OSX).

Then you can just put it somewhere in your path to use it.

# Usage

    ganda --help
    NAME:
       ganda
    
    USAGE:
       ganda [options] [file of urls]  OR  <urls on stdout> | ganda [options]
    
    VERSION:
       0.0.1
    
    DESCRIPTION:
       Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel
    
    AUTHOR:
       Ted Naleid <contact@naleid.com>
    
    COMMANDS:
         help, h  Shows a list of commands or help for one command
    
    GLOBAL OPTIONS:
       --output value, -o value   The output base directory to save downloaded files instead of stdout
       --request value, -X value  The HTTP request method to use (default: "GET")
       --header value, -H value   Header to send along on every request, can be used multiple times
       --workers value, -W value  Number of concurrent workers that will be making requests (default: 30)
       --connect-timeout value    Number of seconds to wait for a connection to be established before timeout (default: 3)
       --help, -h                 show help
       --version, -v              print the version
       
# Example

This command takes the first 1000 words from the macOS dictionary file, then turns each of them into a [Wikipedia API](https://www.mediawiki.org/wiki/API:Main_page) url.

Those urls are then piped into `ganda` and saved in a directory called `out` in the current directory.


    head -1000 /usr/share/dict/words |\
    awk '{print "https://en.wikipedia.org/w/api.php?action=query&titles="$1"&prop=revisions&rvprop=content&format=json"}' |\
    ganda -o out
    
Output (shows hte HTTP status code of 200 OK for each along with the resulting output file that each was saved at):

    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=aam&prop=revisions&rvprop=content&format=json -> out/95/https-en-wikipedia-org-w-api-php-action-query-titles-aam-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=A&prop=revisions&rvprop=content&format=json -> out/71/https-en-wikipedia-org-w-api-php-action-query-titles-A-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=aal&prop=revisions&rvprop=content&format=json -> out/99/https-en-wikipedia-org-w-api-php-action-query-titles-aal-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=a&prop=revisions&rvprop=content&format=json -> out/69/https-en-wikipedia-org-w-api-php-action-query-titles-a-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=aardwolf&prop=revisions&rvprop=content&format=json -> out/31/https-en-wikipedia-org-w-api-php-action-query-titles-aardwolf-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=aalii&prop=revisions&rvprop=content&format=json -> out/91/https-en-wikipedia-org-w-api-php-action-query-titles-aalii-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=aa&prop=revisions&rvprop=content&format=json -> out/ae/https-en-wikipedia-org-w-api-php-action-query-titles-aa-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=Aani&prop=revisions&rvprop=content&format=json -> out/7f/https-en-wikipedia-org-w-api-php-action-query-titles-Aani-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=Aaron&prop=revisions&rvprop=content&format=json -> out/db/https-en-wikipedia-org-w-api-php-action-query-titles-Aaron-prop-revisions-rvprop-content-format-json
    Response:  200 https://en.wikipedia.org/w/api.php?action=query&titles=aardvark&prop=revisions&rvprop=content&format=json -> out/c4/https-en-wikipedia-org-w-api-php-action-query-titles-aardvark-prop-revisions-rvprop-content-format-json
    ... 990 more lines
    
As `ganda` is designed to make many thousands of requests, the files are not saved in the root of the output directory.  Instead the url is hashed and turned into a 2 character subdirectory (similar to how git stores its objects).

example run:

![ganda example run against wikipedia API](https://cdn.rawgit.com/tednaleid/ganda/gh-pages/images/ganda-example.gif)

