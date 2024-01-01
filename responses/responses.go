package responses

import (
	"bytes"
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
type emitResponseFn func(response *http.Response, out io.Writer) (bytesWritten int64, err error)

// we might wrap the body response in a JSON envelope
func determineEmitResponseFn(context *execcontext.Context) emitResponseFn {
	bodyResponseFn := determineEmitBodyResponseFn(context)

	if context.JsonEnvelope {
		return jsonEnvelopeResponseFn(bodyResponseFn, context)
	}

	return bodyResponseFn
}

// returns a function that will emit the JSON envelope around the response body
// the JSON envelope will include the url and http code along with the response body
// TODO: add the request values and the response headers to the JSON envelope
func jsonEnvelopeResponseFn(bodyResponseFn emitResponseFn, context *execcontext.Context) emitResponseFn {
	return func(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
		var bodyBytesWritten int64

		// everything before emitting the body response
		bytesWritten, err = appendString(0, out, fmt.Sprintf(
			"{ \"url\": \"%s\", \"code\": %d, \"body\": ",
			response.Request.URL.String(),
			response.StatusCode,
		))
		if err != nil {
			return bytesWritten, err
		}

		// emit the body response
		if context.ResponseBody == config.Discard || context.ResponseBody == config.Raw {
			// no need to wrap either of these in quotes, Raw is assumed to be JSON
			bodyBytesWritten, err = bodyResponseFn(response, out)
			bytesWritten += bodyBytesWritten
		} else {
			// for all other ResponseBody types we want to encapsulate the body in quotes if it exists,
			// so we need to use a temp buffer to see if there's anything to quote
			tempBuffer := new(bytes.Buffer)
			bodyBytesWritten, err = bodyResponseFn(response, tempBuffer)
			if err == nil && bodyBytesWritten > 0 {
				bytesWritten, err = appendString(bytesWritten, out, "\""+tempBuffer.String()+"\"")
			}
		}

		if err != nil {
			return bytesWritten, err
		}

		// if we didn't write anything for the body response, we emit a `null`
		if bodyBytesWritten == 0 {
			bytesWritten, err = appendString(bytesWritten, out, "null")
			if err != nil {
				return bytesWritten, err
			}
		}

		// close out the JSON envelope
		bytesWritten, err = appendString(bytesWritten, out, " }")
		if err != nil {
			return bytesWritten, err
		}

		return bytesWritten, err
	}
}

// writes a string to the writer and updates the number of bytes written
func appendString(bytesPreviouslyWritten int64, out io.Writer, s string) (int64, error) {
	appendedBytes, err := fmt.Fprint(out, s)
	return bytesPreviouslyWritten + int64(appendedBytes), err
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

func emitRawBody(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	defer response.Body.Close()
	return io.Copy(out, response.Body)
}

func emitSha256BodyFn() func(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	hasher := sha256.New()
	return func(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
		return emitHashedBody(hasher, response, out)
	}
}

func emitHashedBody(hasher hash.Hash, response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	defer response.Body.Close()

	hasher.Reset()

	hashedBytesWritten, err := io.Copy(hasher, response.Body)
	if err != nil || hashedBytesWritten == 0 {
		return 0, err
	}

	n, err := fmt.Fprint(out, hex.EncodeToString(hasher.Sum(nil)))
	return int64(n), err
}

func emitBase64Body(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	defer response.Body.Close()

	encoder := base64.NewEncoder(base64.StdEncoding, out)
	bytesWritten, err = io.Copy(encoder, response.Body)
	if err != nil {
		return bytesWritten, err
	}

	err = encoder.Close()
	return bytesWritten, err
}

func emitNothingBody(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	response.Body.Close()
	return 0, nil
}

//func emitJsonMessagesWithoutBody(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
//	response.Body.Close()
//	return fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d}\n", response.Request.URL.String(), response.StatusCode)
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
