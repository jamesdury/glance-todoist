.PHONY: run test build clean

run:
	go run .

test:
	go test ./...

build:
	go build -o build/output .

clean:
	rm -rf build
