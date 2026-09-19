.PHONY: build install clean run

BINARY := bin/asciifx
CMD := ./cmd/asciifx

build:
	mkdir -p bin
	go build -o $(BINARY) $(CMD)

install:
	go install $(CMD)

run: build
	./$(BINARY)

clean:
	rm -rf bin

lint:
	go vet ./...

test:
	go test ./...
