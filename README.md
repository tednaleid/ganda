# ganda

Quick golang app that you can pipe urls to for it to make parallel GET requests.  It will save the results of all of the requests in a directory that you can then analyze.

Given a file with a list of IDs in it, you could do something like:

    cat id_list.txt | sed 's/\(.*\)/https:\/\/api.example.com\/resource\/\1/' | ./ganda
    
and that will pipe a stream of urls into `ganda` in the format `https://api.example.com/resource/<ID>`.

It will then save the results into subdirectories in the `/tmp/ganda` directory.

# installing

Compile with golang:  `go build`.  

If you have docker installed, you can use `./build.sh` to download a golang container and compile it into a `ganda` binary usable on linux (but not OSX).

Then you can just put it somewhere in your path to use it.

