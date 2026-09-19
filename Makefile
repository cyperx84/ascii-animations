.PHONY: build install clean run

BINARY := asciifx
CMD := ./cmd/asciifx

build:
	go build -o $(BINARY) $(CMD)

install:
	go install $(CMD)

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

lint:
	go vet ./...

test:
	go test ./...
