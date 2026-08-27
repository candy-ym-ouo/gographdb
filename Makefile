APP := gographdb
GO ?= go
GOOS_VALUE := $(shell $(GO) env GOOS)
GOARCH_VALUE := $(shell $(GO) env GOARCH)
PACKAGE := $(APP)-$(GOOS_VALUE)-$(GOARCH_VALUE)

.PHONY: fmt test vet build package clean stats

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	mkdir -p bin
	$(GO) build -trimpath -o bin/$(APP) ./cmd/$(APP)

package: test vet build
	rm -rf dist/$(PACKAGE)
	mkdir -p dist/$(PACKAGE)/bin
	cp bin/$(APP) dist/$(PACKAGE)/bin/
	cp -R web docs README.md dist/$(PACKAGE)/
	tar -C dist -czf dist/$(PACKAGE).tar.gz $(PACKAGE)
	rm -rf dist/$(PACKAGE)
	@echo "created dist/$(PACKAGE).tar.gz"

stats:
	@printf "non-test Go files: "
	@find . -name '*.go' ! -name '*_test.go' | wc -l
	@printf "non-test Go lines: "
	@find . -name '*.go' ! -name '*_test.go' -print0 | xargs -0 wc -l | tail -1 | awk '{print $$1}'

clean:
	rm -rf bin dist
