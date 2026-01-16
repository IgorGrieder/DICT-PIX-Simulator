.PHONY: run build test swagger clean

# Run the application
run:
	go run cmd/server/main.go

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Generate Swagger documentation
swagger:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Clean build artifacts
clean:
	rm -rf bin/
