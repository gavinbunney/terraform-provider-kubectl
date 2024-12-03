SHELL := /usr/bin/env bash
CURRENT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
export GO111MODULE=on

export KUBECONFIG=$(CURRENT_DIR)/scripts/kubeconfig.yaml

default: build

build:
	go install

dist:
	goreleaser build --single-target --skip validate --clean

test-setup:
	@go install gotest.tools/gotestsum@latest
	@go install github.com/boumenot/gocover-cobertura@latest
	rm -rf ./build
	mkdir -p ./build

test: test-setup
	gotestsum --format testname --hide-summary=skipped -- -coverprofile=build/test-coverage.out -covermode=atomic -timeout=30s -count=1 -race ./... || exit 1
	go tool cover -html ./build/test-coverage.out -o ./build/test-coverage.html
	gocover-cobertura < ./build/test-coverage.out > ./build/test-coverage.xml

testacc:
	TF_ACC=1 gotestsum --format testname -- -coverprofile=build/testacc-coverage.out -covermode=atomic -timeout=600s -count=1 ./kubernetes || exit 1
	go tool cover -html ./build/testacc-coverage.out -o ./build/testacc-coverage.html
	gocover-cobertura < ./build/testacc-coverage.out > ./build/testacc-coverage.xml

k3s-start:
	@bash scripts/start-k3s.sh

k3s-stop:
	@bash scripts/stop-k3s.sh

publish:
	goreleaser release --clean

lint: test-setup
	golangci-lint run ./...

vet:
	@echo "go vet ."
	@go vet ./... ; if [ $$? -eq 1 ]; then \
		echo "[!] Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

update-deps:
	go get -u ./...
	go mod tidy

fmt:
	gofmt -s -w .

fmtcheck:
	@if [[ -n `gofmt -l .` ]]; then \
		echo "[!] Found unformatted files. Run formatting with \`make fmt\`"; \
		exit 1; \
	else \
		echo "All files are formatted"; \
	fi

ci-build-setup: test-setup
	sudo rm -f /usr/local/bin/docker-compose
	curl -L https://github.com/docker/compose/releases/download/v2.30.3/docker-compose-`uname -s`-`uname -m` > docker-compose
	chmod +x docker-compose
	sudo mv docker-compose /usr/local/bin
	curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.20.7/bin/linux/amd64/kubectl
	curl -LO "https://dl.k8s.io/release/v1.31.3/bin/linux/amd64/kubectl"
	chmod +x kubectl
	sudo mv kubectl /usr/local/bin/
	bash scripts/gogetcookie.sh

.PHONY: build dist test-setup test testacc k3s-start k3s-stop publish lint vet fmt fmtcheck ci-build-setup
