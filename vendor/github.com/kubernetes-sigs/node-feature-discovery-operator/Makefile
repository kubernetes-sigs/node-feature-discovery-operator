REGISTRY       ?= quay.io
ORG            ?= kubernetes_incubator
TAG            ?= $(shell git rev-parse --abbrev-ref HEAD)
IMAGE          ?= ${REGISTRY}/${ORG}/node-feature-discovery-operator:${TAG}
NAMESPACE      ?= node-feature-discovery-operator
PULLPOLICY     ?= IfNotPresent
TEMPLATE_CMD    = sed 's+REPLACE_IMAGE+${IMAGE}+g; s+REPLACE_NAMESPACE+${NAMESPACE}+g; s+IfNotPresent+${PULLPOLICY}+'
DEPLOY_OBJECTS  = namespace.yaml service_account.yaml role.yaml role_binding.yaml cluster_role_binding.yaml operator.yaml
DEPLOY_CRD      = crds/nfd_v1alpha1_nodefeaturediscovery_crd.yaml
DEPLOY_CR       = crds/nfd_v1alpha1_nodefeaturediscovery_cr.yaml

PACKAGE=github.com/kubernetes-sigs/node-feature-discovery-operator
MAIN_PACKAGE=$(PACKAGE)/cmd/manager

DOCKERFILE=Dockerfile
ENVVAR=GOOS=linux CGO_ENABLED=0
GOOS=linux
GO_BUILD_RECIPE=GOOS=$(GOOS) go build -mod=vendor -o $(BIN) $(MAIN_PACKAGE)


BIN=$(lastword $(subst /, ,$(PACKAGE)))

GOFMT_CHECK=$(shell find . -not \( \( -wholename './.*' -o -wholename '*/vendor/*' \) -prune \) -name '*.go' | sort -u | xargs gofmt -s -l)


all: build

build:
	$(GO_BUILD_RECIPE)

test-e2e: 
	-kubectl delete clusterroles        node-feature-discovery-operator > /dev/null 2>&1
	-kubectl delete clusterrolebindings node-feature-discovery-operator > /dev/null 2>&1

	$(eval TEST_RESOURCES := $(shell mktemp -d)/test-init.yaml)
	$(eval PULLPOLICY := Always)

	@${TEMPLATE_CMD} deploy/service_account.yaml > $(TEST_RESOURCES)
	echo -e "\n---\n" >> $(TEST_RESOURCES)
	@${TEMPLATE_CMD} deploy/role.yaml >> $(TEST_RESOURCES)
	echo -e "\n---\n" >> $(TEST_RESOURCES)
	@${TEMPLATE_CMD} deploy/role_binding.yaml >> $(TEST_RESOURCES)
	echo -e "\n---\n" >> $(TEST_RESOURCES)
	@${TEMPLATE_CMD} deploy/operator.yaml >> $(TEST_RESOURCES)

	go test -v ./test/e2e/... -root $(PWD) -kubeconfig=$(KUBECONFIG) -tags e2e  -globalMan ./deploy/crds/nfd_v1alpha1_nodefeaturediscovery_crd.yaml -namespacedMan $(TEST_RESOURCES)



$(DEPLOY_CRD):
	@${TEMPLATE_CMD} deploy/$@ | kubectl apply -f -

deploy-crd: $(DEPLOY_CRD) 
	sleep 1

deploy-objects: deploy-crd
	for obj in $(DEPLOY_OBJECTS) $(DEPLOY_CR); do               \
		$(TEMPLATE_CMD) deploy/$$obj | kubectl apply -f - ; \
	done	

deploy: deploy-objects
	@${TEMPLATE_CMD} deploy/$(DEPLOY_CR) | kubectl apply -f -

undeploy:
	for obj in $(DEPLOY_CRD) $(DEPLOY_CR) $(DEPLOY_OBJECTS); do  \
		$(TEMPLATE_CMD) deploy/$$obj | kubectl delete -f - ; \
	done	

verify:	verify-gofmt

verify-gofmt:
ifeq (, $(GOFMT_CHECK))
	@echo "verify-gofmt: OK"
else
	@echo "verify-gofmt: ERROR: gofmt failed on the following files:"
	@echo "$(GOFMT_CHECK)"
	@echo ""
	@echo "For details, run: gofmt -d -s $(GOFMT_CHECK)"
	@echo ""
	@exit 1
endif

clean:
	go clean
	rm -f $(BIN)

local-image:
	podman build --no-cache -t $(IMAGE) -f $(DOCKERFILE) .
test:
	go test ./cmd/... ./pkg/... -coverprofile cover.out

local-image-push:
	podman push $(IMAGE) 

.PHONY: all build generate verify verify-gofmt clean local-image local-image-push $(DEPLOY_CRD) 

