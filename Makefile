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
