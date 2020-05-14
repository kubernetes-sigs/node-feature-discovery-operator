REGISTRY       ?= quay.io
ORG            ?= kubernetes_incubator
TAG            ?= $(shell git describe --always --dirty)
IMAGE          ?= $(REGISTRY)/$(ORG)/node-feature-discovery-operator:$(TAG)
NAMESPACE      ?= node-feature-discovery-operator
PULLPOLICY     ?= Always
TEMPLATE_CMD    = sed 's+REPLACE_IMAGE+$(IMAGE)+g; s+REPLACE_NAMESPACE+$(NAMESPACE)+g; s+IfNotPresent+$(PULLPOLICY)+'
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

verify:	verify-gofmt

verify-gofmt:
	@./scripts/verify-gofmt.sh	

clean:
	go clean
	rm -f $(BIN)

clean-labels:
	@$(shell kubectl get no -o yaml | sed -e '/^\s*nfd.node.kubernetes.io/d' -e '/^\s*feature.node.kubernetes.io/d' | kubectl replace -f -)

image:
	podman build --no-cache -t $(IMAGE) -f $(DOCKERFILE) .

test:
	go test ./cmd/... ./pkg/... -coverprofile cover.out

image-push:
	podman push $(IMAGE) 

.PHONY: all build test generate verify verify-gofmt clean local-image local-image-push deploy-objects deploy-operator deploy-crds 
.SILENT: go_mod
