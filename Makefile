CURRENT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TEST?=$$(go list ./... |grep -v 'vendor')
PKG_NAME=kubernetes
export GO111MODULE=on

export TESTARGS=-race -coverprofile=coverage.txt -covermode=atomic
export KUBECONFIG=$(CURRENT_DIR)/scripts/kubeconfig.yaml

default: build

build:
	go install

dist:
	goreleaser build --single-target --skip validate

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
	goreleaser release --clean

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

update-deps:
	go get -u ./...
	go mod tidy

fmt:
	gofmt -s -w .

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

ci-build-setup:
	sudo rm -f /usr/local/bin/docker-compose
	curl -L https://github.com/docker/compose/releases/download/v2.30.3/docker-compose-`uname -s`-`uname -m` > docker-compose
	chmod +x docker-compose
	sudo mv docker-compose /usr/local/bin
	curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.20.7/bin/linux/amd64/kubectl
	curl -LO "https://dl.k8s.io/release/v1.31.3/bin/linux/amd64/kubectl"
	chmod +x kubectl
	sudo mv kubectl /usr/local/bin/
	bash scripts/gogetcookie.sh

.PHONY: build dist test testacc k3s-start k3s-stop publish vet fmt fmtcheck errcheck ci-build-setup
