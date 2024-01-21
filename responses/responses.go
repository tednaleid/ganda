package responses

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

type ResponseWithContext struct {
	Response       *http.Response
	RequestContext interface{}
}

func StartResponseWorkers(responsesWithContext <-chan *ResponseWithContext, context *execcontext.Context) *sync.WaitGroup {
	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(context.ResponseWorkers)

	for i := 1; i <= context.ResponseWorkers; i++ {
		go func() {
			emitResponseWithContext := determineEmitResponseWithContextFn(context)

			if context.WriteFiles {
				responseSavingWorker(responsesWithContext, context, emitResponseWithContext)
			} else {
				responsePrintingWorker(responsesWithContext, context, emitResponseWithContext)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

// creates a worker that takes responses off the channel and saves each one to a file
// the directory path is based off the md5 hash of the url
// the filename is the url with all non-alphanumeric characters replaced with dashes
func responseSavingWorker(
	responsesWithContext <-chan *ResponseWithContext,
	context *execcontext.Context,
	emitResponseWithContextFn emitResponseWithContextFn,
) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responsesWithContext, func(responseWithContext *ResponseWithContext) {
		response := responseWithContext.Response
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		writeableFile := createWritableFile(context.BaseDirectory, context.SubdirLength, filename)
		defer writeableFile.WriteCloser.Close()

		_, err := emitResponseWithContextFn(responseWithContext, writeableFile.WriteCloser)

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
func responsePrintingWorker(
	responsesWithContext <-chan *ResponseWithContext,
	context *execcontext.Context,
	emitResponseWithContext emitResponseWithContextFn,
) {
	out := context.Out
	newline := []byte("\n")
	responseWorker(responsesWithContext, func(responseWithContext *ResponseWithContext) {
		response := responseWithContext.Response
		bytesWritten, err := emitResponseWithContext(responseWithContext, out)

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
type emitResponseWithContextFn func(responseWithContext *ResponseWithContext, out io.Writer) (bytesWritten int64, err error)

// we might wrap the body response in a JSON envelope
func determineEmitResponseWithContextFn(context *execcontext.Context) emitResponseWithContextFn {
	bodyResponseFn := determineEmitBodyResponseFn(context)

	if context.JsonEnvelope {
		return jsonEnvelopeResponseFn(bodyResponseFn, context)
	}

	// not emitting the context, just the body response
	return func(responseWithContext *ResponseWithContext, out io.Writer) (bytesWritten int64, err error) {
		return bodyResponseFn(responseWithContext.Response, out)
	}
}

// returns a function that will emit the JSON envelope around the response body
// the JSON envelope will include the url and http code along with the response body
func jsonEnvelopeResponseFn(bodyResponseFn emitResponseFn, context *execcontext.Context) emitResponseWithContextFn {
	return func(responseWithContext *ResponseWithContext, out io.Writer) (bytesWritten int64, err error) {
		var bodyBytesWritten int64
		var contextBytesWritten int64
		var closingBytesWritten int64

		response := responseWithContext.Response

		requestContext := responseWithContext.RequestContext

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
		} else {
			// for all other ResponseBody types we want to encapsulate the body in quotes if it exists,
			// so we need to use a temp buffer to see if there's anything to quote
			tempBuffer := new(bytes.Buffer)
			bodyBytesWritten, err = bodyResponseFn(response, tempBuffer)
			if err == nil && bodyBytesWritten > 0 {
				bytesWritten, err = appendString(bytesWritten, out, "\""+tempBuffer.String()+"\"")
			}
		}

		bytesWritten += bodyBytesWritten

		if err != nil {
			return bytesWritten, err
		}

		// if we didn't write anything for the body response, we emit a `null`
		if bodyBytesWritten == 0 {
			bodyBytesWritten, err = appendString(bytesWritten, out, "null")
			bytesWritten += bodyBytesWritten
			if err != nil {
				return bytesWritten, err
			}
		}

		// Add requestContext to JSON if it is not nil
		if requestContext != nil {
			requestContextJson, err := json.Marshal(requestContext)
			if err != nil {
				return bytesWritten, err
			}
			contextBytesWritten, err = appendString(bytesWritten, out, fmt.Sprintf(", \"context\": %s", string(requestContextJson)))
			bytesWritten += contextBytesWritten
			if err != nil {
				return bytesWritten, err
			}
		}

		// close out the JSON envelope
		closingBytesWritten, err = appendString(bytesWritten, out, " }")
		bytesWritten += closingBytesWritten
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
		return emitEscapedBodyFn()
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

func emitEscapedBodyFn() func(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	buffer := new(bytes.Buffer)
	return func(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
		return emitEscapedBody(buffer, response, out)
	}
}

func emitEscapedBody(buffer *bytes.Buffer, response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	defer response.Body.Close()

	buffer.Reset()

	_, err = io.Copy(buffer, response.Body)
	if err != nil {
		return 0, err
	}

	if buffer.Len() == 0 {
		return 0, nil
	}

	// Marshal the buffer's contents to JSON
	jsonBytes, err := json.Marshal(buffer.String())
	if err != nil {
		return 0, err
	}

	// Write the JSON bytes to the output writer
	n, err := out.Write(jsonBytes)
	return int64(n), err
}

func emitNothingBody(response *http.Response, out io.Writer) (bytesWritten int64, err error) {
	response.Body.Close()
	return 0, nil
}

func responseWorker(responsesWithContext <-chan *ResponseWithContext, responseHandler func(*ResponseWithContext)) {
	for responseWithContext := range responsesWithContext {
		responseHandler(responseWithContext)
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
