# Kubernetes "kubectl" Provider 

This provider is the best way of managing Kubernetes resources in Terraform, by allowing you to use the thing 
Kubernetes loves best - yaml!

This core of this provider is the `kubectl_manifest` resource, allowing free-form yaml to be processed and applied against Kubernetes.
This yaml object is then tracked and handles creation, updates and deleted seamlessly - including drift detection!

A set of helpful data resources to process directories of yaml files and inline templating is available.

This terraform-provider-kubectl provider has been originally forked from `gavinbunney/kubectl` and synced with `alekc/kubectl`.

## Installation

### Terraform 0.13+

The provider can be installed and managed automatically by Terraform. Sample `versions.tf` file :

```hcl
terraform {
  required_version = ">= 0.13"

  required_providers {
    kubectl = {
      source  = "FindHotel/kubectl"
      version = ">= 2.0.0"
    }
  }
}
```

## Quick Start

```hcl
provider "kubectl" {
  host                   = var.eks_cluster_endpoint
  cluster_ca_certificate = base64decode(var.eks_cluster_ca)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}

resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: couchbase.com/v1
kind: CouchbaseCluster
metadata:
  name: name-here-cluster
spec:
  baseImage: name-here-image
  version: name-here-image-version
  authSecret: name-here-operator-secret-name
  exposeAdminConsole: true
  adminConsoleServices:
    - data
  cluster:
    dataServiceMemoryQuota: 256
    indexServiceMemoryQuota: 256
    searchServiceMemoryQuota: 256
    eventingServiceMemoryQuota: 256
    analyticsServiceMemoryQuota: 1024
    indexStorageSetting: memory_optimized
    autoFailoverTimeout: 120
    autoFailoverMaxCount: 3
    autoFailoverOnDataDiskIssues: true
    autoFailoverOnDataDiskIssuesTimePeriod: 120
    autoFailoverServerGroup: false
YAML
}
```

See [User Guide](https://registry.terraform.io/providers/FindHotel/kubectl/latest) for details on installation and all the provided data and resource types.

---

## Development Guide

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.12+ is *required*).
You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

### Building The Provider

```sh
$ go get github.com/FindHotel/terraform-provider-kubectl
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/FindHotel/terraform-provider-kubectl
$ make build
```

### Testing

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

The provider uses k3s to run integration tests. These tests look for any `*.tf` files in the `_examples` folder and run an `plan`, `apply`, `refresh` and `plan` loop over each file. 

Inside each file the string `name-here` is replaced with a unique name during test execution. This is a simple string replace before the TF is applied to ensure that tests don't fail due to naming clashes. 

Each scenario can be placed in a folder, to help others navigate and use the examples, and added to the [README.MD](./_examples/README.MD). 

> Note: The test infrastructure doesn't support multi-file TF configurations so ensure your test scenario is in a single file. 

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
