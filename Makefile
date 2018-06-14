all: lint test build

lint:
	go fmt
	golint

build: 
	go build -o ganda -v

test:
	go test -v ./...

install: lint test build
	go install

clean: 
	go clean
	rm -f ganda
	rm -f ganda-amd64

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ganda-amd64 -v