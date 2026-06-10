IMG_TAG ?= latest
CHART_DIR=chart
ALL_CHART_NAME=physical-infrastructure
MODULES=cdi maas network exporter log

build-%:
	sudo buildah rmi -f localhost/dci_physical_infrastructure/$(patsubst build-%,%,$@):$(IMG_TAG) || true
	sudo buildah bud -t dci_physical_infrastructure/$(patsubst build-%,%,$@):$(IMG_TAG) -f $(patsubst build-%,%,$@)/Dockerfile .

deploy-%:
	helm upgrade --install $(patsubst deploy-%,%,$@) $(CHART_DIR)/charts/$(patsubst deploy-%,%,$@)

clean-%:
	helm uninstall $(patsubst clean-%,%,$@)

build-all: $(patsubst %,build-%,$(MODULES))

deploy:
	helm upgrade --install $(ALL_CHART_NAME) $(CHART_DIR)

clean:
	helm uninstall $(ALL_CHART_NAME)

.PHONY: build push deploy clean build-all push-all
