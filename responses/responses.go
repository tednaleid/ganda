package responses

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
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
			emitResponse := determineEmitResponseFn(context)

			if context.WriteFiles {
				responseSavingWorker(responses, context, emitResponse)
			} else {
				responsePrintingWorker(responses, context, emitResponse)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

// creates a worker that takes responses off the channel and saves each one to a file
// the directory path is based off the md5 hash of the url
// the filename is the url with all non-alphanumeric characters replaced with dashes
func responseSavingWorker(responses <-chan *http.Response, context *execcontext.Context, emitResponse emitResponseFn) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responses, func(response *http.Response) {
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		writeableFile := createWritableFile(context.BaseDirectory, context.SubdirLength, filename)
		defer writeableFile.WriteCloser.Close()

		_, err := emitResponse(response, writeableFile.WriteCloser)

		if err != nil {
			context.Logger.LogError(err, response.Request.URL.String()+" -> "+writeableFile.FullPath)
		} else {
			context.Logger.LogResponse(response.StatusCode, response.Request.URL.String()+" -> "+writeableFile.FullPath)
		}
	})
}

// creates a worker that takes responses off the channel and prints each one to stdout
// if the JsonEnvelope flag is set, it will wrap the response in a JSON envelope
// a newline will be emitted after each non-empty response
func responsePrintingWorker(responses <-chan *http.Response, context *execcontext.Context, emitResponse emitResponseFn) {
	out := context.Out
	newline := []byte("\n")
	responseWorker(responses, func(response *http.Response) {
		bytesWritten, err := emitResponse(response, out)
		if err != nil {
			context.Logger.LogError(err, response.Request.URL.String())
		} else {
			context.Logger.LogResponse(response.StatusCode, response.Request.URL.String())
			if bytesWritten > 0 {
				out.Write(newline)
			}
		}
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
		// if it's discard, we want to emit `null`. still need to call emitNothingBody so we close the body
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
		return emitNothingBody
	case config.Escaped:
		return emitNothingBody //TODO
	case config.Base64:
		return emitBase64Body
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

func emitBase64Body(response *http.Response, out io.Writer) (written int64, err error) {
	defer response.Body.Close()

	encoder := base64.NewEncoder(base64.StdEncoding, out)
	written, err = io.Copy(encoder, response.Body)
	if err != nil {
		return written, err
	}

	err = encoder.Close()
	return written, err
}

func emitNothingBody(response *http.Response, out io.Writer) (written int64, err error) {
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

type WritableFile struct {
	FullPath    string
	WriteCloser io.WriteCloser
}

func createWritableFile(baseDirectory string, subdirLength int64, filename string) *WritableFile {
	directory := directoryForFile(baseDirectory, filename, subdirLength)
	fullPath := directory + filename

	file, err := os.Create(fullPath)
	if err != nil {
		panic(err)
	}

	return &WritableFile{FullPath: fullPath, WriteCloser: file}
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
