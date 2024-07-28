# What is `ganda`?

Ganda lets you make HTTP/HTTPS requests to hundreds to millions of URLs in just a few minutes.

It's designed with the unix philosophy of ["do one thing well"](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well) and wants to be used in a chain of command line pipes to make its requests in parallel. 

By default, it will echo all response bodies to standard out but can optionally save the results of each request in a directory for later analysis.

Given a file with a list of IDs in it, you could do something like:

    cat id_list.txt | awk '{printf "https://api.example.com/resource/%s?apikey=foo\n", $1}' | ganda
    
and that will pipe a stream of urls into `ganda` in the format `https://api.example.com/resource/<ID>?apikey=foo`.

Alternatively, if you have a file full of urls (one per line), you can just tell `ganda` to run that:

    ganda my_file_of_urls.txt

If you give `ganda` a `-o <directory name>` parameter, it will save the body of each in a separate file inside `<directory name>`.  If you want a single file, just pipe stdout the normal way `... | ganda > result.txt`.

For many more examples, see ["Using HTTP APIs on the Command Line - Part 3 - ganda"](http://www.naleid.com/2018/04/04/using-http-apis-on-the-command-line-3-ganda.html).

# Installing

You currently have 3 options:

1. on MacOS you can install with [homebrew](https://brew.sh/)
```
brew tap tednaleid/homebrew-ganda
brew install ganda
```

2. download the appropriate binary from the [releases page](https://github.com/tednaleid/ganda/releases) and put it in your path

3. Compile from source with golang:

```
go install github.com/tednaleid/ganda@latest
```

to install in your `$GOPATH/bin` (which you want in your `$PATH`)

# Usage


-- TODO: update

       
# Example

This command takes the first 1000 words from the macOS dictionary file, then turns each of them into a [Wikipedia API](https://www.mediawiki.org/wiki/API:Main_page) url.

Those urls are then piped into `ganda` and saved in a directory called `out` in the current directory.


    head -1000 /usr/share/dict/words |\
    awk '{printf "https://en.wikipedia.org/w/api.php?action=query&titles=%s&prop=revisions&rvprop=content&format=json\n", $1}' |\
    ganda -o out --subdir-length 2
    
Output (shows the HTTP status code of 200 OK for each along with the resulting output file that each was saved at):

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
    
As `ganda` is designed to make many thousands of requests, you can use the `--subdir-length` to avoid making your filesystem unhappy with 1M files in a single directory.  That switch will hash each url and place the response in a subdirectory (similar to how git stores its objects).

example run:

![ganda example run against wikipedia API](https://cdn.rawgit.com/tednaleid/ganda/gh-pages/images/ganda-example.gif)

