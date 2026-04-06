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
mock-generate: install-minimock
	@echo "📦 Generating mocks..."
	@go generate ./...
	@echo "✅ Mock generation completed"

.PHONY: mock-clean
mock-clean:
	@echo "🧹 Cleaning generated mocks..."
	@find . -name "*_mock_test.go" -type f -delete
	@echo "✅ Mocks cleaned"

.PHONY: install-minimock
install-minimock:
	@echo "Installing minimock"
	@go install github.com/gojuno/minimock/v3/cmd/minimock@latest
	@echo "Minimock installed"