CURRENT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=kubernetes
export GO111MODULE=on

export TESTARGS=-race -coverprofile=coverage.txt -covermode=atomic
export KUBECONFIG=$(CURRENT_DIR)/scripts/kubeconfig.yaml

default: build

build:
	go install

dist:
	goreleaser build --single-target --skip-validate --rm-dist

test:
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc:
	TF_ACC=1 go test ./kubernetes -v $(TESTARGS) -timeout 120m -count=1

k3s-start:
	@bash scripts/start-k3s.sh

k3s-stop:
	@bash scripts/stop-k3s.sh

publish:
	goreleaser release --rm-dist

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

ci-build-setup:
	sudo rm /usr/local/bin/docker-compose
	curl -L https://github.com/docker/compose/releases/download/1.25.4/docker-compose-`uname -s`-`uname -m` > docker-compose
	chmod +x docker-compose
	sudo mv docker-compose /usr/local/bin
	curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.20.7/bin/linux/amd64/kubectl
	chmod +x kubectl
	sudo mv kubectl /usr/local/bin/
	bash scripts/gogetcookie.sh

.PHONY: build dist test testacc k3s-start k3s-stop publish vet fmt fmtcheck errcheck ci-build-setup
