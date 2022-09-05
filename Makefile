.PHONY: all
all: bin/feeder

.PHONY: clean
clean:
	rm -Rf bin

bin/feeder: config.go
	mkdir -p bin
	go build -o bin/feeder

config.go:
	cp config.go.def config.go
