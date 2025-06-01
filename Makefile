.PHONY: build clean

# Build the binary for the commentor application
build:
	go build -o commentor cmd/commentor/main.go

# Clean up generated files and binaries
clean:
	rm -f commentor

