---
id: kubectl_manifest
title: kubectl_manifest
---

Create a Kubernetes resource using raw YAML manifests.

This resource handles creation, deletion and even updating your kubernetes resources. This allows complete lifecycle management of your kubernetes resources are terraform resources!

Behind the scenes, this provider uses the same capability as the `kubectl apply` command, that is, you can update the YAML inline and the resource will be updated in place in kubernetes.

## Example Usage

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
```

> Note: When the kind is a Deployment, this provider will wait for the deployment to be rolled out automatically for you!

## Argument Reference

* `yaml_body` - Required. YAML to apply to kubernetes.
* `force_new` - Optional. Forces delete & create of resources if the `yaml_body` changes. Default `false`.
* `ignore_fields` - Optional. List of map fields to ignore when applying the manifest. See below for more details.
* `wait_for_rollout` - Optional. Set this flag to wait or not for Deployments and APIService to complete rollout. Default `true`.

## Attribute Reference

* `api_version` - Extracted API Version from `yaml_body`.
* `kind` - Extracted object kind from `yaml_body`.
* `name` - Extracted object name from `yaml_body`.
* `namespace` - Extracted object namespace from `yaml_body`.
* `uid` - Kubernetes unique identifier from last run.
* `resource_version` - Resource version from kubernetes from last run.
* `live_uid` - Current uuid from kubernetes.
* `live_resource_version` - Current uuid from kubernetes.
* `yaml_incluster` - Current yaml within kubernetes.
* `live_manifest_incluster` - Current manifest within kubernetes.

## Ignore Manifest Fields

You can configure a list of yaml keys to ignore changes to via the `ignore_fields` field.
Set these for fields set by Operators or other processes in kubernetes and as such you don't want to update.

```hcl
resource "kubectl_manifest" "test" {
    ignore_fields = ["caBundle"]
    yaml_body = <<YAML
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: istio-sidecar-injector
webhooks:
  - clientConfig:
      caBundle: ""
YAML
}
```

## Waiting for Rollout

By default, this resource will wait for Deployments and APIServices to complete their rollout before proceeding.
You can disable this behavior by setting the `wait_for_rollout` field to `false`.

## Import

This provider supports importing existing resources. The ID format expected uses a double `//` as a deliminator (as apiVersion can have a forward-slash):

```
# Import the my-namespace Namespace
terraform import kubectl_manifest.my-namespace v1//Namespace//my-namespace

# Import the certmanager Issuer CRD named cluster-selfsigned-issuer-root-ca from the my-namespace namespace
$ terraform import -provider kubectl module.kubernetes.kubectl_manifest.crd-example certmanager.k8s.io/v1alpha1//Issuer//cluster-selfsigned-issuer-root-ca//my-namespace
```
