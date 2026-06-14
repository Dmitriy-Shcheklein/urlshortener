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

.PHONY: migrations-generate
migrations-generate:
	@if [ -n "$(name)" ]; then \
		echo "Migration generating: $(name)"; \
		migrate create -ext sql -dir ./migrations -seq $(name); \
		echo "✅ Migration created"; \
    else \
		echo "Usage: make migrations-generate name=<migration_name>"; \
		echo "Example: make migrations-generate name=create_users_table"; \
		exit 1; \
	fi

.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

.PHONY: bench-mem
bench-mem:
	@echo "Running benchmarks with memory profiling..."
	@mkdir -p profiles
	go test -bench=BenchmarkShorten -benchmem -count=5 -memprofile=profiles/base.pprof ./internal/services/shortener/
	@echo "Profile saved to profiles/base.pprof"

.PHONY: load-test
load-test:
	@echo "Running load test with k6..."
	k6 run scripts/load-test.js

.PHONY: profile-heap
profile-heap:
	@echo "Saving heap profile..."
	@mkdir -p profiles
	curl -s http://localhost:8080/debug/pprof/heap > profiles/base.pprof
	@echo "Heap profile saved to profiles/base.pprof"

.PHONY: pprof-mem
pprof-mem:
	@echo "Starting pprof web UI on http://localhost:8080..."
	go tool pprof -http=:8081 profiles/base.pprof

