VERSION ?= v0.1.0-dev
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test check install demo
build:
	go build -tags demo -ldflags "$(LDFLAGS)" -o sidea .
test:
	go test -race -tags demo ./...
check: test
	go vet -tags demo ./...
	git diff --check
install:
	go install -tags demo -ldflags "$(LDFLAGS)" .
demo: build
	./sidea demo
