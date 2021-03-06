# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
IMAGE_TAG_BASE ?= ghcr.io/tenzen-y/imperator/imperator-controller
IMAGE_TAG_VERSION ?= latest

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):$(IMAGE_TAG_VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

LDFLAGS := $(shell hack/version.sh)

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: check
check: manifests generate fmt vet golangci-lint bundle-manifests

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate/boilerplate.go.txt" paths="./..."

.PHONY: bundle-manifests
bundle-manifests:
	kustomize build config/default  > deploy/imperator.yaml
	kustomize build config/crd > deploy/crds.yaml

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: golangci-lint
golangci-lint:
ifeq ("$(shell golangci-lint version 2>/dev/null)", "")
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
endif
	golangci-lint run ./... --timeout 3m

.PHONY: test
test: envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out.tmp -covermode=atomic
	@awk '!/.*zz_generated.deepcopy.go.*/' cover.out.tmp > cover.out
	@rm -f cover.out.tmp
	go tool cover -func cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build imperator-controller binary.
	go build -ldflags "$(LDFLAGS)" -o bin/imperator-controller cmd/imperator-controller/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run -ldflags "$(LDFLAGS)" ./cmd/imperator-controller/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the imperator-controller.
	docker build -t ${IMG} -f cmd/imperator-controller/Dockerfile --build-arg LDFLAGS="$(LDFLAGS)" .

.PHONY: docker-push
docker-push: ## Push docker image with the imperator-controller.
	docker push ${IMG}

.PHONY: docker-release
docker-release: docker-push
	docker tag ${IMG} ${IMAGE_TAG_BASE}:latest
	docker push ${IMAGE_TAG_BASE}:latest

RELEASE_VERSION ?= v0.0.0
.PHONY: prepare-release
prepare-release: check test
	RELEASE_VERSION="$(RELEASE_VERSION)" yq eval -i '.images[0].newTag|=env(RELEASE_VERSION)' ./config/manager/kustomization.yaml
	$(MAKE) bundle-manifests

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kustomize build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kustomize build config/crd | kubectl delete -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kustomize build config/default | kubectl delete -f -

.PHONY: kind-start
kind-start:
	bash ./hack/kind-start.sh

.PHONY: load-image
load-image: docker-build
	kind load docker-image ${IMG}

.PHONY: integration-test
integration-test: load-image
	bash ./hack/integration-test.sh

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
