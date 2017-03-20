package urls

import (
	"bufio"
	"github.com/tednaleid/ganda/base"
	"os"
)

func ProcessUrls(requests chan<- string, urlFilename string) {
	urls := urlScanner(urlFilename)
	for urls.Scan() {
		url := urls.Text()
		requests <- url
	}
}

func urlScanner(urlFilename string) *bufio.Scanner {
	if len(urlFilename) > 0 {
		base.Logger.Println("Opening url file: ", urlFilename)
		return urlFileScanner(urlFilename)
	}
	return urlStdinScanner()
}

func urlStdinScanner() *bufio.Scanner {
	return bufio.NewScanner(os.Stdin)
}

func urlFileScanner(urlFilename string) *bufio.Scanner {
	file, err := os.Open(urlFilename)
	base.Check(err)
	return bufio.NewScanner(file)
}
