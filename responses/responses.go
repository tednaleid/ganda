package responses

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/tednaleid/ganda/execcontext"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
	"crypto/sha256"
)

func StartResponseWorkers(responses <-chan *http.Response, context *execcontext.Context) *sync.WaitGroup {
	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(context.ResponseWorkers)

	for i := 1; i <= context.ResponseWorkers; i++ {
		go func() {
			if context.WriteFiles {
				responseSavingWorker(responses, context)
			} else {
				responsePrintingWorker(responses, context)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

func responseSavingWorker(responses <-chan *http.Response, context *execcontext.Context) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responses, func(response *http.Response) {
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		fullPath := saveBodyToFile(context.BaseDirectory, context.SubdirLength, filename, response.Body)
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String()+" -> "+fullPath)
	})
}

func responsePrintingWorker(responses <-chan *http.Response, context *execcontext.Context) {
	emitResponseFn := determineEmitResponseFn(context)
	out := context.Out
	responseWorker(responses, func(response *http.Response) {
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String())
		emitResponseFn(response, out)
	})
}

type emitResponseFn func(response *http.Response, out io.Writer)

func determineEmitResponseFn(context *execcontext.Context) emitResponseFn {
	if context.JsonEnvelope {
		if context.DiscardBody {
			return emitJsonMessagesWithoutBody
		} else if context.HashBody {
			return emitJsonMessageSha256
		}
		return emitJsonMessages
	}

	if context.DiscardBody {
		return emitNothing
	}

	return emitRawMessages
}

func emitRawMessages(response *http.Response, out io.Writer) {
	defer response.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	if buf.Len() > 0 {
		buf.WriteByte('\n')
		out.Write(buf.Bytes())
	}
}

func emitNothing(response *http.Response, out io.Writer) {
	response.Body.Close()
}

func emitJsonMessages(response *http.Response, out io.Writer) {
	defer response.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	if buf.Len() > 0 {
		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": %s }\n", response.Request.URL.String(), response.StatusCode, buf.Len(), buf)
	} else {
		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": null }\n", response.Request.URL.String(), response.StatusCode, 0)
	}
}

func emitJsonMessagesWithoutBody(response *http.Response, out io.Writer) {
	response.Body.Close()
	fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d}\n", response.Request.URL.String(), response.StatusCode)
}

func emitJsonMessageSha256(response *http.Response, out io.Writer) {
	defer response.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	if buf.Len() > 0 {
		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": \"%x\" }\n", response.Request.URL.String(), response.StatusCode, buf.Len(), sha256.Sum256(buf.Bytes()))
	} else {
		fmt.Fprintf(out, "{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": null }\n", response.Request.URL.String(), response.StatusCode, 0)
	}
}

func responseWorker(responses <-chan *http.Response, responseHandler func(*http.Response)) {
	for response := range responses {
		responseHandler(response)
	}
}

func saveBodyToFile(baseDirectory string, subdirLength int, filename string, body io.ReadCloser) string {
	defer body.Close()

	directory := directoryForFile(baseDirectory, filename, subdirLength)
	fullPath := directory + filename

	file, err := os.Create(fullPath)
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(file, body)
	if err != nil {
		panic(err)
	}

	return fullPath
}

func directoryForFile(baseDirectory string, filename string, subdirLength int) string {
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
