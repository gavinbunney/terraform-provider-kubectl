CURRENT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=kubernetes
export GO111MODULE=on

export TESTARGS=-race -coverprofile=coverage.txt -covermode=atomic
export KUBECONFIG=$(CURRENT_DIR)/scripts/kubeconfig.yaml

default: build

build: fmtcheck
	go install

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
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

changelog:
	@if [ "$(GITHUB_TOKEN)" == "" ]; then echo "[!] GITHUB_TOKEN environment variable is required but not set" && exit 1; fi;
	@docker run --rm -v $(CURDIR):/usr/local/src/your-app -t ferrarimarco/github-changelog-generator:latest --user gavinbunney --project terraform-provider-kubectl --token ${GITHUB_TOKEN} --no-compare-link

vendor-status:
	@govendor status

build-binaries:
	@sh -c "'$(CURDIR)/scripts/build.sh'"

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

.PHONY: build test testacc vet fmt fmtcheck errcheck vendor-status test-compile build-binaries changelog
