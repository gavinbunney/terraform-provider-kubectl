# Kubernetes YAML Provider 

*NOTE* this is a fork (of a fork!) of the original provider provided by [nabancard and lawrecncegripper](https://github.com/nabancard/terraform-provider-kubernetes-yaml). This fork adds support for inplace updates of kubernetes resources.

[![Build Status](https://travis-ci.org/gavinbunney/terraform-provider-kubernetes-yaml.svg?branch=master)](https://travis-ci.org/gavinbunney/terraform-provider-kubernetes-yaml)

This was originally proposed [as a PR to add a YAML resource](https://github.com/terraform-providers/terraform-provider-kubernetes/pull/195) into the official Terraform provider. 

While the work is ongoing to provide a better experience in the official provider I've pulled the code out into a standalone provider which just provides the YAML resource. This allows for it to be used alongside the existing providers. 

## Status: Experimental

Currently the code has been tried on a limited number of use cases. I would expect wider use to find issue, please raise them on the repository and make contributions to resolve them if you can. 

### Support around Issues/PRs

There is no support around this provider. If it missing a feature or it has a bug feel free to raise an issue but there is no time allocated to maintian and resolve issues.

Likewise PRs are welcome but the time to review and merge may vary based on my availability. 

## Using the provider

![demo](docs/yamldemo.gif)

Download a binary for your system from the release page and remove the `-os-arch` details so you're left with `terraform-provider-k8sraw`. Use `chmod +x` to make it executable and then either place it at the root of your Terraform folder or in the Terraform plugin folder on your system. 

Then you can create a YAML resources by using the following Terraform:

```hcl
provider "k8sraw" {}

resource "k8sraw_yaml" "test" {
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

The provider also support a retry when creating the resource `create_retry_count = 15`. This is useful for `CRD`s who's operators are also being created during the `terraform` operation to allow time for the `CRD` definition to be created in the cluster. 

```hcl
provider "k8sraw" {
  create_retry_count = 15
}

resource "k8sraw_yaml" "test" {
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


## Building The Provider

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-kubernetes`

```sh
$ mkdir -p $GOPATH/src/github.com/gavinbunney; cd $GOPATH/src/github.com/gavinbunney
$ git clone git@github.com:gavinbunney/terraform-provider-kubernetes-yaml
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/gavinbunney/terraform-provider-kubernetes-yaml
$ make build
```

## Developing the Provider

### Testing

The provide uses MiniKube to run integration tests. These tests look for any `*.tf` files in the `_examples` folder and run an `plan`, `apply`, `refresh` and `plan` loop over each file. 

Inside each file the string `name-here` is replaced with a unique name during test execution. This is a simple string replace before the TF is applied to ensure that tests don't fail due to naming clashes. 

Each scenario can be placed in a folder, to help others navigate and use the examples, and added to the [README.MD](./_examples/README.MD). 

> Note: The test infrastructure doesn't support multi-file TF configurations so ensure your test scenario is in a single file. 

### Debugging

Using `vscode` with `delve` and `golang` configured the launch task in the repo will start a debugging session for the tests. Open the repo and press `F5` to get started.

### Development Environment

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.9+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-kubernetes
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
