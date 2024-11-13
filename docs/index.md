# Kubectl Provider

This provider is the best way of managing Kubernetes resources in Terraform, by allowing you to use the thing 
Kubernetes loves best - yaml!

This core of this provider is the `kubectl_manifest` resource, allowing free-form yaml to be processed and applied against Kubernetes.
This yaml object is then tracked and handles creation, updates and deleted seamlessly - including drift detection!

A set of helpful data resources to process directories of yaml files and inline templating is available.

This `terraform-provider-kubectl` provider has been used by many large Kubernetes installations to completely
manage the lifecycle of Kubernetes resources. 

## Installation

### Terraform 0.13+

The provider can be installed and managed automatically by Terraform. Sample `versions.tf` file :

```hcl
terraform {
  required_version = ">= 0.13"

  required_providers {
    kubectl = {
      source  = "FindHotel/kubectl"
      version = ">= 1.14.1"
    }
  }
}
```

## Configuration

The provider supports the same configuration parameters as the [Kubernetes Terraform Provider](https://www.terraform.io/docs/providers/kubernetes/index.html).

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
```

### Argument Reference

The following arguments are supported:

* `apply_retry_count` - (Optional) Defines the number of attempts any create/update action will take. Default `1`.
* `load_config_file` - (Optional) Flag to enable/disable loading of the local kubeconf file. Default `true`. Can be sourced from `KUBE_LOAD_CONFIG_FILE`.
* `host` - (Optional) The hostname (in form of URI) of the Kubernetes API. Can be sourced from `KUBE_HOST`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes API. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes API. Can be sourced from `KUBE_PASSWORD`.
* `insecure` - (Optional) Whether the server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`. Defaults to `false`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_path` - (Optional) A path to a kube config file. Can be sourced from `KUBE_CONFIG_PATH` or `KUBECONFIG`.
* `config_paths` - (Optional) A list of paths to the kube config files. Can be sourced from `KUBE_CONFIG_PATHS`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `config_context_auth_info` - (Optional) Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`). Can be sourced from `KUBE_CTX_AUTH_INFO`.
* `config_context_cluster` - (Optional) Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`). Can be sourced from `KUBE_CTX_CLUSTER`.
* `token` - (Optional) Token of your service account.  Can be sourced from `KUBE_TOKEN`.
* `proxy_url` - (Optional) URL to the proxy to be used for all API requests. URLs with "http", "https", and "socks5" schemes are supported. Can be sourced from `KUBE_PROXY_URL`.
* `exec` - (Optional) Configuration block to use an [exec-based credential plugin] (https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins), e.g. call an external command to receive user credentials.
    * `api_version` - (Required) API version to use when decoding the ExecCredentials resource, e.g. `client.authentication.k8s.io/v1beta1`.
    * `command` - (Required) Command to execute.
    * `args` - (Optional) List of arguments to pass when executing the plugin.
    * `env` - (Optional) Map of environment variables to set when executing the plugin.

### Exec Plugin Support

As with the Kubernetes Terraform Provider, this provider also supports using a `exec` based plugin (for example when running on EKS).

```hcl
provider "kubectl" {
  apply_retry_count      = 15
  host                   = var.eks_cluster_endpoint
  cluster_ca_certificate = base64decode(var.eks_cluster_ca)
  load_config_file       = false

  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    command     = "aws-iam-authenticator"
    args = [
      "token",
      "-i",
      module.eks.cluster_id,
    ]
  }
}
```

### Retry Support

The provider has an additional paramater `apply_retry_count` that allows kubernetes commands to be retried on failure.
This is useful if you have flaky CRDs or network connections and need to wait for the cluster state to be back in quorum.
This applies to both create and update operations. 

```hcl
provider "kubectl" {
  apply_retry_count = 15
}
```

## Example

Loading a raw yaml manifest into kubernetes is simple, just set the `yaml_body` argument:

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

Updates are also preserved so you can fully manage your kubernetes resources with ease!
