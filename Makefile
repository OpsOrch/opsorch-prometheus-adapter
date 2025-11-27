GO ?= go
GOCACHE ?= $(PWD)/.gocache
GOMODCACHE ?= $(PWD)/.gocache/mod
CACHE_ENV = GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)

.PHONY: all fmt test build plugin clean

all: test

fmt:
	$(GO)fmt -w .

test:
	$(CACHE_ENV) $(GO) test ./...

build:
	$(CACHE_ENV) $(GO) build ./...

plugin:
	$(CACHE_ENV) $(GO) build -o bin/metricplugin ./cmd/metricplugin

integ-metric:
	$(CACHE_ENV) $(GO) run ./integ/metric.go

integ: integ-metric

clean:
	rm -rf $(GOCACHE) bin
