package responses

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"hash"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func StartResponseWorkers(responses <-chan *http.Response, context *execcontext.Context) *sync.WaitGroup {
	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(context.ResponseWorkers)

	for i := 1; i <= context.ResponseWorkers; i++ {
		go func() {
			/*
				high level algorithm
					- if we're saving files, we want each response to create it's own file and have an io.Writer
					- if we're emitting to stdout, we've already got an io.Writer at context.Out
					- this needs to be called per response, so we can't just pass in the io.Writer to the worker
					- if we're emitting to stdout, we want to emit a \n between each non zero length response
				if we're emitting a json-envelope
					-  we want to emit part of the JSON up to the body, then emit the body function, then close the JSON
					- if we're not emitting JSON envelope, no need to wrap it
				separately, we want the response-body to take an io.writer from the above and be able to write to it
					- create higher-order function that returns a function that takes a response and an io.Writer
			*/

			emitResponse := determineEmitResponseFn(context)

			if context.WriteFiles {
				// TODO change this so that it calls determineEmitResponseFn regardless of if it is saving or printing
				// and then passes that responseFn to the worker we create

				responseSavingWorker(responses, emitResponse, context)
			} else {
				responsePrintingWorker(responses, emitResponse, context)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

func responseSavingWorker(responses <-chan *http.Response, emitResponse emitResponseFn, context *execcontext.Context) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responses, func(response *http.Response) {
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		// this should be changed to get an io.Writer for the file that we can pass in

		fullPath := saveBodyToFile(context.BaseDirectory, context.SubdirLength, filename, response.Body)
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String()+" -> "+fullPath)
	})
}

func responsePrintingWorker(responses <-chan *http.Response, emitResponse emitResponseFn, context *execcontext.Context) {
	out := context.Out
	responseWorker(responses, func(response *http.Response) {
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String())
		emitResponse(response, out)
	})
}

// takes a response and writes it to the writer, returns true if it wrote anything
type emitResponseFn func(response *http.Response, out io.Writer) (written int64, err error)

// we might wrap the body response in a JSON envelope
func determineEmitResponseFn(context *execcontext.Context) emitResponseFn {
	bodyResponseFn := determineEmitBodyResponseFn(context)

	if context.JsonEnvelope {
		// it matters what the body response function is:
		// if it's raw, we want to just emit it
		// if it's discard, we want to emit `null`. still need to call emitNothing so we close the body
		// if it's base64, we want to encapsulate it in quotes for a JSON string
		// if it's escaped, we want to encapsulate it in quotes for a JSON string
		// if it's sha256, we want to encapsulate it in quotes for a JSON string

		//if context.DiscardBody {
		//	return emitJsonMessagesWithoutBody
		//} else if context.HashBody {
		//	return emitJsonMessageSha256
		//}
		//return emitJsonMessages
	}

	return bodyResponseFn
}

func determineEmitBodyResponseFn(context *execcontext.Context) emitResponseFn {
	switch context.ResponseBody {
	case config.Raw:
		return emitRawBody
	case config.Sha256:
		return emitSha256BodyFn()
	case config.Discard:
		return emitNothing
	case config.Escaped:
		return emitNothing //TODO
	case config.Base64:
		return emitNothing //TODO
	default:
		panic(fmt.Sprintf("unknown response body type %s", context.ResponseBody))
	}
}

func emitRawBody(response *http.Response, out io.Writer) (written int64, err error) {
	defer response.Body.Close()
	return io.Copy(out, response.Body)
}

func emitSha256BodyFn() func(response *http.Response, out io.Writer) (written int64, err error) {
	hasher := sha256.New()
	return func(response *http.Response, out io.Writer) (written int64, err error) {
		return emitHashedBody(hasher, response, out)
	}
}

func emitHashedBody(hasher hash.Hash, response *http.Response, out io.Writer) (written int64, err error) {
	defer response.Body.Close()

	hasher.Reset()
	if _, err := io.Copy(hasher, response.Body); err != nil {
		return 0, err
	}

	n, err := fmt.Fprint(out, hex.EncodeToString(hasher.Sum(nil)))
	return int64(n), err
}

func emitNothing(response *http.Response, out io.Writer) (written int64, err error) {
	response.Body.Close()
	return 0, nil
}

//func emitJsonMessages(response *http.Response, out io.Writer) {
//	defer response.Body.Close()
//	buf := new(bytes.Buffer)
//	buf.ReadFrom(response.Body)
//
//	if buf.Len() > 0 {
//		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": %s }\n", response.Request.URL.String(), response.StatusCode, buf.Len(), buf)
//	} else {
//		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": null }\n", response.Request.URL.String(), response.StatusCode, 0)
//	}
//}

//func emitJsonMessagesWithoutBody(response *http.Response, out io.Writer) (written int64, err error) {
//	response.Body.Close()
//	return fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d}\n", response.Request.URL.String(), response.StatusCode)
//}
//
//func emitJsonMessageSha256(response *http.Response, out io.Writer) bool {
//	defer response.Body.Close()
//	buf := new(bytes.Buffer)
//	buf.ReadFrom(response.Body)
//
//	if buf.Len() > 0 {
//		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": \"%x\" }\n", response.Request.URL.String(), response.StatusCode, buf.Len(), sha256.Sum256(buf.Bytes()))
//	} else {
//		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": null }\n", response.Request.URL.String(), response.StatusCode, 0)
//	}
//	return true
//}

func responseWorker(responses <-chan *http.Response, responseHandler func(*http.Response)) {
	for response := range responses {
		responseHandler(response)
	}
}

func createWritableFile(baseDirectory string, subdirLength int64, filename string) io.WriteCloser {
	directory := directoryForFile(baseDirectory, filename, subdirLength)
	fullPath := directory + filename

	file, err := os.Create(fullPath)
	if err != nil {
		panic(err)
	}

	return file
}

func saveBodyToFile(baseDirectory string, subdirLength int64, filename string, body io.ReadCloser) string {
	defer body.Close()

	directory := directoryForFile(baseDirectory, filename, subdirLength)
	fullPath := directory + filename

	file, err := os.Create(fullPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	if err != nil {
		panic(err)
	}

	return fullPath
}

func directoryForFile(baseDirectory string, filename string, subdirLength int64) string {
	var directory string
	if subdirLength <= 0 {
		directory = fmt.Sprintf("%s/", baseDirectory)
	} else {
		sliceEnd := 1

		// don't create directories longer than 4 binary hex characters (4^16 = 65k directories)
		if subdirLength > 2 {
			sliceEnd = 2
		}

		md5val := md5.Sum([]byte(filename))
		directory = fmt.Sprintf("%s/%x/", baseDirectory, md5val[0:sliceEnd])
	}

	os.MkdirAll(directory, os.ModePerm)
	return directory
}
