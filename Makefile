IMAGE_BUILD_CMD ?= docker build
IMAGE_BUILD_EXTRA_OPTS ?=
IMAGE_PUSH_CMD ?= docker push
CONTAINER_RUN_CMD ?= docker run -u "`id -u`:`id -g`"

MDL ?= mdl

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

VERSION := $(shell git describe --tags --dirty --always)

IMAGE_REGISTRY ?= k8s.gcr.io/nfd
IMAGE_NAME := node-feature-discovery-operator
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_EXTRA_TAG_NAMES ?=
IMAGE_REPO := $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG := $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))

NAMESPACE      := node-feature-discovery-operator
PULLPOLICY     ?= Always
TEMPLATE_CMD    = sed 's+REPLACE_IMAGE+$(IMAGE_TAG)+g; s+REPLACE_NAMESPACE+$(NAMESPACE)+g; s+IfNotPresent+$(PULLPOLICY)+'
GOFMT_CHECK=$(shell find . -not \( \( -wholename './.*' -o -wholename '*/vendor/*' \) -prune \) -name '*.go' | sort -u | xargs gofmt -s -l)

DEPLOY_OBJECTS  = manifests/0100_namespace.yaml manifests/0200_service_account.yaml manifests/0300_cluster_role.yaml manifests/0400_cluster_role_binding.yaml manifests/0600_operator.yaml
DEPLOY_CRDS     = manifests/0500_crd.yaml
DEPLOY_CRS      = manifests/0700_cr.yaml

PACKAGE=github.com/kubernetes-sigs/node-feature-discovery-operator
MAIN_PACKAGE=$(PACKAGE)/cmd/manager

BIN=node-feature-discovery-operator

DOCKERFILE=Dockerfile

GOOS=linux

all: build

go_mod:
	@go mod download

build: go_mod
	@GOOS=$(GOOS) go build -o $(BIN) $(MAIN_PACKAGE)

deploy-crds:
	for obj in $(DEPLOY_CRDS); do \
		$(TEMPLATE_CMD) $$obj | kubectl apply -f - ;\
		sleep 1;\
	done

deploy-objects: deploy-crds
	for obj in $(DEPLOY_OBJECTS); do \
		$(TEMPLATE_CMD) $$obj | kubectl apply -f - ;\
		sleep 1;\
	done

deploy: deploy-objects
	for obj in $(DEPLOY_CRS); do \
		$(TEMPLATE_CMD) $$obj | kubectl apply -f - ;\
		sleep 1;\
	done

undeploy:
	for obj in $(DEPLOY_OBJECTS) $(DEPLOY_CRDS) $(DEPLOY_CRS); do \
		$(TEMPLATE_CMD) $$obj | kubectl delete -f - ;\
	done

verify:	verify-gofmt ci-lint

verify-gofmt:
	@./scripts/verify-gofmt.sh

ci-lint:
	golangci-lint run --timeout 5m0s

mdlint:
	find docs/ -path docs/vendor -prune -false -o -name '*.md' | xargs $(MDL) -s docs/mdl-style.rb

clean:
	go clean
	rm -f $(BIN)

clean-labels:
	kubectl get no -o yaml | sed -e '/^\s*nfd.node.kubernetes.io/d' -e '/^\s*feature.node.kubernetes.io/d' | kubectl replace -f -

image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

test:
	go test ./cmd/... ./pkg/... -coverprofile cover.out

push:
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag; done

site-build:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c '/usr/local/bin/bundle install && "$$BUNDLE_BIN/jekyll" build $(JEKYLL_OPTS)'

site-serve:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c '/usr/local/bin/bundle install && "$$BUNDLE_BIN/jekyll" serve $(JEKYLL_OPTS) -H 127.0.0.1'

.PHONY: all build test generate verify verify-gofmt clean deploy-objects deploy-operator deploy-crds push image
.SILENT: go_mod
