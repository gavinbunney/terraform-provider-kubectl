# Kubernetes "kubectl" Provider 

[![Build Status](https://travis-ci.org/gavinbunney/terraform-provider-kubectl.svg?branch=master)](https://travis-ci.org/gavinbunney/terraform-provider-kubectl)

This is a fork (of a fork!) of the original provider provided by [nabancard and lawrecncegripper](https://github.com/nabancard/terraform-provider-kubernetes-yaml).

This fork adds :
1. Support for in-place updates of kubernetes resources
2. Data resource to iterate over directories of manifests

## Using the provider

Download a binary for your system from the release page and remove the `-os-arch` details so you're left with `terraform-provider-kubectl`.
Use `chmod +x` to make it executable and then either place it at the root of your Terraform folder or in the Terraform plugin folder on your system. 

### Quick Start

```hcl
provider "kubectl" {
  host                   = var.eks_cluster_endpoint
  cluster_ca_certificate = base64decode(var.eks_cluster_ca)
  token                  = data.aws_eks_cluster_auth.main.token
  load_config_file       = false
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

### Provider Configuration

The provider supports the same configuration parameters as the [Kubernetes Terraform Provider](https://www.terraform.io/docs/providers/kubernetes/index.html)

```hcl
provider "kubectl" {
  host                   = var.eks_cluster_endpoint
  cluster_ca_certificate = base64decode(var.eks_cluster_ca)
  token                  = data.aws_eks_cluster_auth.main.token
  load_config_file       = false
}
```

The provider has an additional paramater `create_retry_count` that allows kubernetes commands to be retried on failure.
This is useful if you have flaky CRDs or network connections and need to wait for the cluster state to be back in quorum. 

```hcl
provider "kubectl" {
  create_retry_count = 15
}
```

### Create Kubernetes Resources from YAML

Then you can create a YAML resources by using the following Terraform:

```hcl
resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    azure/frontdoor: enabled
spec:
  rules:
  - http:
      paths:
      - path: /testpath
        backend:
          serviceName: test
          servicePort: 80
YAML
}
```

> Note: When the kind is a Deployment, this provider will wait for the deployment to be rolled out automatically for you

### Import Kubernetes Resource

This provider supports importing existing resources. The ID format expected uses a double `//` as a deliminator (as apiVersion can have a forward-slash):

```
apiVersion//kind//name//namespace[optional]
```

Example:

```bash
# Import the my-namespace-name namespace
$ terraform import -provider kubectl module.kubernetes.kubectl_manifest.namespace-example v1//Namespace//my-namespace-name

# Import the certmanager Issuer CRD named cluster-selfsigned-issuer-root-ca from the my-namespace namespace
$ terraform import -provider kubectl module.kubernetes.kubectl_manifest.crd-example certmanager.k8s.io/v1alpha1//Issuer//cluster-selfsigned-issuer-root-ca//my-namespace
```

### Load Kubernetes Manifests from file

This provider provides a `data` resource `kubectl_filename_list` to enable ease of working with directories of kubernetes manifests.

```hcl
data "kubectl_filename_list" "manifests" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    count = length(data.kubectl_filename_list.manifests.matches)
    yaml_body = file(element(data.kubectl_filename_list.manifests.matches, count.index))
}
```

### Split Multi-Document YAML Manifests

This provider provides a `data` resource `kubectl_file_documents` to enable ease of splitting multi-document yaml content.

```hcl
data "kubectl_file_documents" "manifests" {
    content = file("multi-doc-manifest.yaml")
}

resource "kubectl_manifest" "test" {
    count = length(data.kubectl_file_documents.manifests.documents)
    yaml_body = file(element(data.kubectl_file_documents.manifests.documents, count.index))
}
```

### Split Multi-Document YAML Manifests from Path

This provider provides a `data` resource `kubectl_path_documents` to enable ease of splitting multi-document yaml content, from a collection of matching files.
Think of is as a combination of both `kubectl_filename_list` and `kubectl_file_documents`

```hcl
data "kubectl_path_documents" "manifests" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    count = length(data.kubectl_path_documents.manifests.documents)
    yaml_body = file(element(data.kubectl_path_documents.manifests.documents, count.index))
}
```

## Development Guide

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.11+ is *required*).
You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

### Building The Provider

```sh
$ go get github.com/gavinbunney/terraform-provider-kubectl
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/gavinbunney/terraform-provider-kubectl
$ make build
```

### Testing

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

The provide uses MiniKube to run integration tests. These tests look for any `*.tf` files in the `_examples` folder and run an `plan`, `apply`, `refresh` and `plan` loop over each file. 

Inside each file the string `name-here` is replaced with a unique name during test execution. This is a simple string replace before the TF is applied to ensure that tests don't fail due to naming clashes. 

Each scenario can be placed in a folder, to help others navigate and use the examples, and added to the [README.MD](./_examples/README.MD). 

> Note: The test infrastructure doesn't support multi-file TF configurations so ensure your test scenario is in a single file. 

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
