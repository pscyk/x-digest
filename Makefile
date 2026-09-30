.PHONY: build test bench fuzz clean demo

BINARY := x-digest$(shell go env GOEXE)

build:
	go build -o $(BINARY) ./cmd/x-digest

demo: build
	vhs docs/demo.tape
	ffmpeg -v error -ss 6 -i docs/demo.mp4 -frames:v 1 -y docs/demo-poster.png

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./... | tee bench.txt

fuzz:
	go test -fuzz=. -fuzztime=30s ./internal/display

clean:
	rm -f x-digest x-digest.exe bench.txt
