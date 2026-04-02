.PHONY: build test bench fuzz clean

build:
	go build -o x-digest.exe ./cmd/x-digest

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./... | tee bench.txt

fuzz:
	go test -fuzz=. -fuzztime=30s ./internal/display

clean:
	rm -f x-digest.exe bench.txt
