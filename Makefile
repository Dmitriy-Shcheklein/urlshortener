.PHONY: test
test:
	@echo "Running tests..."
	go test -cover -coverprofile=cover.profile -v ./...
	@if [ -f "cover.profile" ]; then \
		go tool cover -func cover.profile; \
		rm -f cover.profile; \
	fi

.PHONY: build
build:
	go build -o bundle ./cmd/shortener

.PHONY: run
run: build
	./bundle

.PHONY: lint
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run -v --config .golangci.yml; \
	else \
		echo "⚠️  golangci-lint not installed, skipping linting"; \
	fi

.PHONY: mock-generate
mock-generate: mock-clean
	@echo "📦 Generating mocks..."
	@mockery
	@echo "✅ Mock generation completed"

.PHONY: mock-clean
mock-clean:
	@echo "🧹 Cleaning generated mocks..."
	@find . -name "mocks_test.go" -type f -delete
	@echo "✅ Mocks cleaned"


.PHONY: fmt
fmt:
	@echo "✨ Formatting code..."
	@if command -v gofumpt > /dev/null; then \
		gofumpt -w -l .; \
	else \
		echo "⚠️  gofumpt not installed, skipping formatting"; \
	fi