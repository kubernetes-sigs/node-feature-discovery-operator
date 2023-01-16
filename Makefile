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
JEKYLL_VERSION := 3.8
JEKYLL_ENV ?= development
SITE_BUILD_CMD := $(CONTAINER_RUN_CMD) --rm -i \
	-e JEKYLL_ENV=$(JEKYLL_ENV) \
	--volume="$$PWD/docs:/srv/jekyll":Z \
	--volume="$$PWD/docs/vendor/bundle:/usr/local/bundle":Z \
	--network=host jekyll/jekyll:$(JEKYLL_VERSION)
SITE_BASEURL ?=
SITE_DESTDIR ?= _site
JEKYLL_OPTS := -d '$(SITE_DESTDIR)' $(if $(SITE_BASEURL),-b '$(SITE_BASEURL)',)

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.2.1)
# - use environment variables to overwrite this value (e.g export VERSION=0.2.1)
VERSION := $(shell git describe --tags --dirty --always)

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
IMAGE_REGISTRY ?= k8s.gcr.io/nfd
IMAGE_NAME := node-feature-discovery-operator
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_EXTRA_TAG_NAMES ?=
IMAGE_REPO ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG ?= $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))
BASE_IMAGE_FULL ?= debian:buster-slim
BASE_IMAGE_MINIMAL ?= gcr.io/distroless/base

IMAGE_TAG_RBAC_PROXY ?= gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0

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
LDFLAGS = -ldflags "-s -w -X sigs.k8s.io/node-feature-discovery-operator/pkg/version.version=$(VERSION)"

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

all: build

# Run tests
ENVTEST_ASSETS_DIR=$(PROJECT_DIR)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); $(GO_CMD) test ./... -coverprofile cover.out
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

clean-manifests = (cd config/manager && $(KUSTOMIZE) edit set image controller=k8s.gcr.io/nfd/node-feature-discovery-operator:0.4.2)

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: kustomize
	cd $(PROJECT_DIR)/config/manager && \
		$(KUSTOMIZE) edit set image controller=${IMAGE_TAG}-minimal
	cd $(PROJECT_DIR)/config/default && \
		$(KUSTOMIZE) edit set image kube-rbac-proxy=${IMAGE_TAG_RBAC_PROXY}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	@$(call clean-manifests)

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	@$(GO_FMT) -w -l $$(find . -name '*.go')

# Run go vet against code
vet:
	$(GO_CMD)  vet ./...

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
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="utils/boilerplate.go.txt" paths="./..."

# Build the container image
image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		--target full \
		--build-arg BASE_IMAGE_FULL=$(BASE_IMAGE_FULL) \
		--build-arg BASE_IMAGE_MINIMAL=$(BASE_IMAGE_MINIMAL) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

image-minimal:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG)-minimal \
		--target minimal \
		--build-arg BASE_IMAGE_FULL=$(BASE_IMAGE_FULL) \
		--build-arg BASE_IMAGE_MINIMAL=$(BASE_IMAGE_MINIMAL) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)-minimal) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

# Push the container image
push: 
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag; done

push-minimal:
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)-minimal
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag; done

site-build:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c '/usr/local/bin/bundle install && "$$BUNDLE_BIN/jekyll" build $(JEKYLL_OPTS)'

site-serve:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c '/usr/local/bin/bundle install && "$$BUNDLE_BIN/jekyll" serve $(JEKYLL_OPTS) -H 127.0.0.1'

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(PROJECT_DIR)/bin/controller-gen
controller-gen:
	@GOBIN=$(PROJECT_DIR)/bin GO111MODULE=on $(GO_CMD) install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0

# Download kustomize locally if necessary
KUSTOMIZE = $(PROJECT_DIR)/bin/kustomize
kustomize:
	@GOBIN=$(PROJECT_DIR)/bin GO111MODULE=on $(GO_CMD) install sigs.k8s.io/kustomize/kustomize/v4@v4.5.2

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_TAG)-minimal
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	$(IMAGE_BUILD_CMD)  -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# push the bundle image.
.PHONY: bundle-push
bundle-push:
	$(IMAGE_PUSH_CMD) $(BUNDLE_IMG)
