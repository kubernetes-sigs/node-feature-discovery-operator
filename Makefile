.PHONY: all build test generate verify verify-gofmt clean deploy-objects deploy-operator deploy-crds push image
.SILENT: go_mod
.FORCE:

GO_CMD ?= go
GO_FMT ?= gofmt
CONTAINER_RUN_CMD ?= docker run -u "`id -u`:`id -g`"

# Docker base command for working with html documentation.
# Use host networking because 'jekyll serve' is stupid enough to use the
# same site url than the "host" it binds to. Thus, all the links will be
# broken if we'd bind to 0.0.0.0
RUBY_IMAGE_VERSION := 3.1
JEKYLL_ENV ?= development
SITE_BUILD_CMD := $(CONTAINER_RUN_CMD) --rm -i -u "`id -u`:`id -g`" \
	$(shell [ -t 0 ] && echo '-t') \
	-e JEKYLL_ENV=$(JEKYLL_ENV) \
	--volume="$$PWD/docs:/work" \
	--volume="$$PWD/docs/vendor/bundle:/usr/local/bundle" \
	-w /work \
	--network=host ruby:$(RUBY_IMAGE_VERSION)
SITE_BASEURL ?=
SITE_DESTDIR ?= _site
JEKYLL_OPTS := -d '$(SITE_DESTDIR)' $(if $(SITE_BASEURL),-b '$(SITE_BASEURL)',)

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.2.1)
# - use environment variables to overwrite this value (e.g export VERSION=0.2.1)
VERSION := $(shell git describe --tags --dirty --always)
BUNDLE_VERSION := $(shell git tag | sort -V | tail -1 | awk '{print substr($$1,2); }')

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= controller-bundle:$(VERSION)

# Image URL to use all building/pushing image targets
IMAGE_BUILD_CMD ?= docker build
IMAGE_PUSH_CMD ?= docker push
IMAGE_BUILD_EXTRA_OPTS ?=
IMAGE_REGISTRY ?= registry.k8s.io/nfd
IMAGE_NAME := node-feature-discovery-operator
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_EXTRA_TAG_NAMES ?=
IMAGE_REPO ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG ?= $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))
BUILDER_IMAGE ?= golang:1.26-trixie
BASE_IMAGE_DEBUG ?= debian:bookworm-slim
BASE_IMAGE_PROD ?= gcr.io/distroless/base


# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO_CMD) env GOBIN))
GOBIN=$(shell $(GO_CMD) env GOPATH)/bin
else
GOBIN=$(shell $(GO_CMD) env GOBIN)
endif

GOOS=linux

PACKAGE=sigs.k8s.io/node-feature-discovery-operator
MAIN_PACKAGE=main.go
BIN=node-feature-discovery-operator
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

all: build

# Run tests
test: generate fmt vet manifests
	$(GO_CMD) test ./... -coverprofile cover.out
go_mod:
	@$(GO_CMD) mod download

# Build binary
build: go_mod
	@GOOS=$(GOOS) GO111MODULE=on CGO_ENABLED=0 $(GO_CMD) build -o $(BIN) $(LDFLAGS) $(MAIN_PACKAGE)

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	$(GO_CMD) run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

clean-manifests = (cd config/manager && $(KUSTOMIZE) edit set image controller=registry.k8s.io/nfd/node-feature-discovery-operator:0.4.2)

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: kustomize
	cd $(PROJECT_DIR)/config/manager && \
		$(KUSTOMIZE) edit set image controller=${IMAGE_TAG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	@$(call clean-manifests)

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD etc.
# No rbac generator: the operator's RBAC is maintained by hand in config/rbac
# (and the Helm chart), and nothing consumes a role generated from the
# +kubebuilder:rbac markers, so generating one only leaves an orphan file.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	@$(GO_FMT) -w -l $$(find . -name '*.go')

# Run go vet against code
vet:
	$(GO_CMD)  vet ./...

TESTS ?= ./...

.PHONY: unit-test
unit-test: vet ## Run tests.
	$(GO_CMD) test $(TESTS) -coverprofile cover.out

verify:	verify-gofmt ci-lint

verify-gofmt:
	@./scripts/verify-gofmt.sh

ci-lint:
	golangci-lint run --timeout 5m0s

mdlint:
	${CONTAINER_RUN_CMD} \
	--rm \
	--volume "${PWD}:/workdir:ro,z" \
	--workdir /workdir \
	ruby:slim \
	/workdir/scripts/test-infra/mdlint.sh

clean:
	$(GO_CMD)  clean
	rm -f $(BIN)

# clean NFD labels on all nodes
# devel only
clean-labels:
	kubectl get no -o yaml | sed -e '/^\s*nfd.node.kubernetes.io/d' -e '/^\s*feature.node.kubernetes.io/d' | kubectl replace -f -

# Generate code
generate: controller-gen mockgen
	$(CONTROLLER_GEN) object:headerFile="utils/boilerplate.go.txt" paths="./..."
	PATH=$(PROJECT_DIR)/bin:$$PATH $(GO_CMD) generate ./...

# Build the container image
image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		--target prod \
		--build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
		--build-arg BASE_IMAGE_PROD=$(BASE_IMAGE_PROD) \
		--build-arg BASE_IMAGE_DEBUG=$(BASE_IMAGE_DEBUG) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

image-debug:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG)-debug \
		--target debug \
		--build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
		--build-arg BASE_IMAGE_PROD=$(BASE_IMAGE_PROD) \
		--build-arg BASE_IMAGE_DEBUG=$(BASE_IMAGE_DEBUG) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)-debug) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

# Push the container image
push:
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag; done

push-debug:
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)-debug
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag-debug; done

site-build:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c "bundle plugin install bundler-override && bundle install && jekyll build $(JEKYLL_OPTS)"

site-serve:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c "bundle plugin install bundler-override && bundle install && jekyll serve $(JEKYLL_OPTS) -H 127.0.0.1"

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(PROJECT_DIR)/bin/controller-gen
controller-gen:
	@GOBIN=$(PROJECT_DIR)/bin GO111MODULE=on $(GO_CMD) install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0

.PHONY: mockgen
mockgen: ## Install mockgen locally.
	@GOBIN=$(PROJECT_DIR)/bin GO111MODULE=on $(GO_CMD) install go.uber.org/mock/mockgen@v0.6.0

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	@GOBIN=$(PROJECT_DIR)/bin  GO111MODULE=on $(GO_CMD) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4

# Download kustomize locally if necessary
KUSTOMIZE = $(PROJECT_DIR)/bin/kustomize
kustomize:
	@GOBIN=$(PROJECT_DIR)/bin GO111MODULE=on $(GO_CMD) install sigs.k8s.io/kustomize/kustomize/v5@v5.8.1

# Download operator-sdk locally if necessary.
# v1.37.0 is the last release with the go.kubebuilder.io/v3 plugin that
# PROJECT declares; v1.38.0 removed the go/v2 and go/v3 layouts.
# The binary is checked against the SHA-256 from that release's checksums.txt,
# pinned here so a changed download is rejected. Update the sums with the
# version.
OPERATOR_SDK_VERSION ?= v1.37.0
OPERATOR_SDK_SHA256_linux_amd64 = 20da1fcba9ef70b1e23283ae820a2c3387b529f04ce09cf318597b33f5d59a52
OPERATOR_SDK_SHA256_linux_arm64 = df746275d76c0570f00de57128a063e1118dd67c7235a12cb84d7e0ade93098a
OPERATOR_SDK_SHA256_darwin_amd64 = ca3e4028cd62f21f4ed988907b884be530098e7c40523e89046dd8c5b0178eb9
OPERATOR_SDK_SHA256_darwin_arm64 = 2a58cd10865655937c3a298368b45379937e18be205f8cb429a4a1c51a5f92af
OPERATOR_SDK_PLATFORM = $(shell $(GO_CMD) env GOOS)_$(shell $(GO_CMD) env GOARCH)
OPERATOR_SDK_SHA256 = $(OPERATOR_SDK_SHA256_$(OPERATOR_SDK_PLATFORM))
OPERATOR_SDK = $(PROJECT_DIR)/bin/operator-sdk
.PHONY: operator-sdk
operator-sdk:
	@test -x $(OPERATOR_SDK) && $(OPERATOR_SDK) version | grep -q '"$(OPERATOR_SDK_VERSION)"' || { \
		set -e; \
		test -n "$(OPERATOR_SDK_SHA256)" || { echo "no pinned operator-sdk checksum for $(OPERATOR_SDK_PLATFORM)" >&2; exit 1; }; \
		mkdir -p $(PROJECT_DIR)/bin; \
		curl -sSfL -o $(OPERATOR_SDK).download https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$(OPERATOR_SDK_PLATFORM); \
		sum=$$( (sha256sum $(OPERATOR_SDK).download 2>/dev/null || shasum -a 256 $(OPERATOR_SDK).download) | cut -d' ' -f1); \
		if [ "$$sum" != "$(OPERATOR_SDK_SHA256)" ]; then \
			rm -f $(OPERATOR_SDK).download; \
			echo "operator-sdk checksum mismatch: got $$sum, want $(OPERATOR_SDK_SHA256)" >&2; exit 1; \
		fi; \
		chmod +x $(OPERATOR_SDK).download; mv -f $(OPERATOR_SDK).download $(OPERATOR_SDK); }

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize operator-sdk
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_TAG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(BUNDLE_VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	$(IMAGE_BUILD_CMD)  -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# push the bundle image.
.PHONY: bundle-push
bundle-push:
	$(IMAGE_PUSH_CMD) $(BUNDLE_IMG)
