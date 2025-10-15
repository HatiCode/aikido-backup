.PHONY: build fmt vet clean

BINARY=app

build: fmt vet test
	@echo "Building..."
	@go build -o $(BINARY)
	@echo "Build complete: ./$(BINARY)"

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)
	@echo "Clean complete"

test:
	@go test -v ./...

fmt:
	@go fmt ./...

vet:
	@go vet ./...