.PHONY: build run tidy clean

# Build the Ivai OS binary
build:
	@echo "Building Ivai OS..."
	go build -o bin/ivai cmd/ivai/main.go

# Run the Ivai OS
run:
	@echo "Starting Ivai OS..."
	go run cmd/ivai/main.go

# Tidy up Go modules
tidy:
	@echo "Tidying Go modules..."
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf bin/
