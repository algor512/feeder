OS        = linux
ARCH      = amd64

.PHONY: all
all: bin/parse_feeds bin/update_page

.PHONY: clean
clean:
	rm -Rf bin

bin/parse_feeds: cmd/parse_feeds/main.go
	GOOS=$(OS) GOARCH=$(ARCH) go build -o bin/parse_feeds ./cmd/parse_feeds

bin/update_page: cmd/update_page/main.go
	GOOS=$(OS) GOARCH=$(ARCH) go build -o bin/update_page ./cmd/update_page
