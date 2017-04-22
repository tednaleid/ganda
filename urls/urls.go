package urls

import (
	"fmt"
	"bufio"
	"log"
	"os"
)

func UrlScanner(urlFilename string, logger *log.Logger) (*bufio.Scanner, error) {
	if len(urlFilename) > 0 {
		logger.Println("Opening file of urls at: ", urlFilename)
		return urlFileScanner(urlFilename)
	}
	return urlStdinScanner(), nil
}

func urlStdinScanner() *bufio.Scanner {
	return bufio.NewScanner(os.Stdin)
}

func urlFileScanner(urlFilename string) (*bufio.Scanner, error) {
	if _, err := os.Stat(urlFilename); os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to open specified file: %s", urlFilename)
	}

	file, err := os.Open(urlFilename)
	return bufio.NewScanner(file), err
}
