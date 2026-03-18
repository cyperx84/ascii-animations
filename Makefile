.PHONY: build install clean run

BINARY := showcase
CMD := ./cmd/showcase

build:
	go build -o $(BINARY) $(CMD)

install:
	go install $(CMD)

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf exported/

lint:
	go vet ./...

test:
	go test ./...
