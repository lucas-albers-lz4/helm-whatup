HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= $(HELM_HOME)/plugins/helm-whatup
VERSION := $(shell sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)
DIST := $(CURDIR)/_dist
LDFLAGS := "-X main.version=${VERSION}"

.PHONY: build
build:
	go build -o bin/helm-whatup -ldflags $(LDFLAGS) ./main.go

.PHONY: test
test:
	go test -v ./...

.PHONY: dist
dist:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -o bin/helm-whatup ./main.go
	tar -zcvf $(DIST)/helm-whatup-$(VERSION)-linux-amd64.tar.gz bin/helm-whatup README.md LICENSE plugin.yaml
	GOOS=darwin GOARCH=amd64 go build -o bin/helm-whatup ./main.go
	tar -zcvf $(DIST)/helm-whatup-$(VERSION)-darwin-amd64.tar.gz bin/helm-whatup README.md LICENSE plugin.yaml

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: bootstrap
bootstrap:
	# Initialize modules and download dependencies
	go mod tidy
	# Explicitly download modules for which we have replacements
	go mod download k8s.io/helm
	go mod download github.com/gosuri/uitable
	go mod download github.com/spf13/cobra
	go mod download gopkg.in/yaml.v2
	# Verify all dependencies are available
	go mod verify

.PHONY: clean
clean:
	rm -rf $(DIST)
	rm -rf bin
