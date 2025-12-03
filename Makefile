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
	$(CACHE_ENV) $(GO) build -o bin/alertplugin ./cmd/alertplugin

integ-metric:
	$(CACHE_ENV) $(GO) run ./integ/metric.go

integ-alert:
	$(CACHE_ENV) $(GO) run ./integ/alert.go

integ: integ-metric integ-alert

clean:
	rm -rf $(GOCACHE) bin
