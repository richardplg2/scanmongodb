.PHONY: build build-linux clean

build:
	go build -o scanmongodb .

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o scanmongodb-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o scanmongodb-linux-arm64 .

clean:
	rm -f scanmongodb scanmongodb-linux-*
