.PHONY: build build-debug test bench fuzz clean

build:
	go build -o x-digest.exe .

build-debug:
	go build -tags debug -o x-digest-debug.exe .

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem | tee bench.txt

fuzz:
	go test -fuzz=. -fuzztime=30s

clean:
	rm -f x-digest.exe x-digest-debug.exe bench.txt
