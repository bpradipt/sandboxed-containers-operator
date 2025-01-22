# Default variables

REGISTRY_AUTH_FILE := ocp_pull_secret.json
IMAGE_STD ?= quay.io/openshift_sandboxed_containers/rhcos-layer/ocp-4.16:latest
IMAGE_TDX ?= quay.io/openshift_sandboxed_containers/rhcos-layer/ocp-4.16:tdx-latest
IMAGE_SNP ?= quay.io/openshift_sandboxed_containers/rhcos-layer/ocp-4.16:snp-latest
KATA_COCO_ARTEFACTS ?= quay.io/openshift_sandboxed_containers/kata-coco-artefacts:latest

TEE ?=

# Default target
.PHONY: build
build:  check-podman get-pull-secret get-ocp-release-image ## Default target: Build the RHCOS layer
	@echo "Building RHCOS layer..."
	@if [ "$(TEE)" == "" ]; then \
		echo "Building standard RHCOS layer with kata binaries"  ;\
		podman build --authfile $(REGISTRY_AUTH_FILE) --build-arg OCP_RELEASE_IMAGE=$(shell cat ./.ocp-release-image) -t $(IMAGE_STD) -f standard/Containerfile . ;\
	elif [ "$(TEE)" == "tdx" ]; then \
		echo "Building tdx RHCOS layer with kata binaries" ;\
		podman create --name kata $(KATA_COCO_ARTEFACTS) ;\
		podman cp kata:/kata-containers-tdx-repo.tar.xz tdx/ ;\
		podman rm kata ;\
		podman build --authfile $(REGISTRY_AUTH_FILE) --build-arg OCP_RELEASE_IMAGE=$(shell cat ./.ocp-release-image) -t $(IMAGE_TDX) -f tdx/Containerfile . ;\
	else [ "$(TEE)" == "snp" ]; \
		echo "Building snp RHCOS layer with kata binaries" ;\
		podman create --name kata $(KATA_COCO_ARTEFACTS) ;\
		podman cp kata:/rhcos-coco-snp.initrd.tar.xz snp/ ;\
		podman rm kata ;\
		podman build --authfile $(REGISTRY_AUTH_FILE) --build-arg OCP_RELEASE_IMAGE=$(shell cat ./.ocp-release-image) -t $(IMAGE_SNP) -f snp/Containerfile . ;\
	fi

# Target to check if 'jq' is available
.PHONY: check-jq
check-jq: ## Check if 'jq' command is available
	@which jq > /dev/null || (echo "'jq' command not found, please install it" && exit 1)


# Target to check if 'oc' is available
.PHONY: check-oc
check-oc: ## Check if 'oc' command is available
	@which oc > /dev/null || (echo "'oc' command not found, please install it" && exit 1)

# Target to check if 'podman' is available
.PHONY: check-podman
check-podman: ## Check if 'podman' command is available
	@which podman > /dev/null || (echo "'podman' command not found, please install it" && exit 1)

# Target to get the pull secret from 'oc'
.PHONY: get-pull-secret
get-pull-secret: ## Get the pull secret from oc
	@echo "Fetching pull secret from the cluster..."
	@if [ ! -f "$(REGISTRY_AUTH_FILE)" ]; then \
		echo "\$$REGISTRY_AUTH_FILE is not set. Retrieving the pull secret..."; \
		$(MAKE) download-pull-secret; \
		echo "Downloaded the pull secret to: $(REGISTRY_AUTH_FILE)"; \
	else \
		echo "$(REGISTRY_AUTH_FILE) is already present. Skipping download."; \
	fi

# Target to download the pull secret
.PHONY: download-pull-secret
download-pull-secret: ## Download the pull secret
	@echo "Downloading the pull secret..."
	$(MAKE) check-oc
	$(MAKE) check-jq
	oc get -n openshift-config secret/pull-secret -ojson | jq -r '.data.".dockerconfigjson"' | base64 -d | jq '.' > $(REGISTRY_AUTH_FILE)
	@echo "Pull secret downloaded to auth file:$(REGISTRY_AUTH_FILE)"

# Target to get OCP release image details
.PHONY: get-ocp-release-image
get-ocp-release-image: ## Get the OCP release image to use as base
	@echo "Getting OCP release image to use as base..."
	$(MAKE) check-oc; \
	oc adm release info --image-for rhel-coreos > .ocp-release-image


# Cleanup target
.PHONY: clean
clean: ## Cleanup build env

	@echo "Removing pull .ocp-release-image and $(REGISTRY_AUTH_FILE)"
	rm -f .ocp-release-image $(REGISTRY_AUTH_FILE)

# Help target
.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make [target]\n"} /^[a-zA-Z_-]+:.*##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

