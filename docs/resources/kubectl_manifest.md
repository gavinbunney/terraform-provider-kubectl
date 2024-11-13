# Resource: kubectl_manifest

Create a Kubernetes resource using raw YAML manifests.

This resource handles creation, deletion and even updating your kubernetes resources. This allows complete lifecycle management of your kubernetes resources as terraform resources!

Behind the scenes, this provider uses the same capability as the `kubectl apply` command, that is, you can update the YAML inline and the resource will be updated in place in kubernetes.

> **TIP:** This resource only supports a single yaml resource. If you have a list of documents in your yaml file,
> use the [kubectl_path_documents](https://registry.terraform.io/providers/FindHotel/kubectl/latest/docs/data-sources/kubectl_path_documents) data source to split the files into individual resources.

## Example Usage

```hcl
resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: networking.k8s.io/v1
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
        pathType: "Prefix"
        backend:
          serviceName: test
          servicePort: 80
YAML
}
```

> Note: When the kind is a Deployment, this provider will wait for the deployment to be rolled out automatically for you!

## Argument Reference

* `yaml_body` - Required. YAML to apply to kubernetes.
* `sensitive_fields` - Optional. List of fields (dot-syntax) which are sensitive and should be obfuscated in output. Defaults to `["data"]` for Secrets.
* `force_new` - Optional. Forces delete & create of resources if the `yaml_body` changes. Default `false`.
* `server_side_apply` - Optional. Allow using server-side-apply method. Default `false`.
* `force_conflicts` - Optional. Allow using force_conflicts. Default `false`.
* `apply_only` - Optional. It does not delete resource in any case Default `false`.
* `ignore_fields` - Optional. List of map fields to ignore when applying the manifest. See below for more details.
* `override_namespace` - Optional. Override the namespace to apply the kubernetes resource to, ignoring any declared namespace in the `yaml_body`.
* `validate_schema` - Optional. Setting to `false` will mimic `kubectl apply --validate=false` mode. Default `true`.
* `wait` - Optional. Set this flag to wait or not for finalized to complete for deleted objects. Default `false`.
* `wait_for_rollout` - Optional. Set this flag to wait or not for Deployments and APIService to complete rollout. Default `true`.

## Attribute Reference

* `yaml_body_parsed` - Obfuscated version of `yaml_body`, with `sensitive_fields` hidden.
* `api_version` - Extracted API Version from `yaml_body`.
* `kind` - Extracted object kind from `yaml_body`.
* `name` - Extracted object name from `yaml_body`.
* `namespace` - Extracted object namespace from `yaml_body`.
* `uid` - Kubernetes unique identifier from last run.
* `live_uid` - Current uuid from kubernetes.
* `yaml_incluster` - Current yaml within kubernetes.
* `live_manifest_incluster` - Current manifest within kubernetes.

## Sensitive Fields

You can obfuscate fields in the diff output by setting the `sensitive_fields` option. This allows you to hide arbitrary field content by suppressing the information in the diff.

By default, this is set to `["data"]` for all `v1/Secret` manifests.

The fields provided should use dot-separater syntax to specify the field to obfuscate.

```hcl
resource "kubectl_manifest" "test" {
    sensitive_fields = [
        "metadata.annotations.my-secret-annotation"
    ]

    yaml_body = <<YAML
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: istio-sidecar-injector
  annotations:
    my-secret-annotation: "this is very secret"
webhooks:
  - clientConfig:
      caBundle: ""
YAML
}
```

> Note: Only Map values are supported to be made sensitive. If you need to make a value from a list (or sub-list) sensitive, you can set the high-level key as sensitive to suppress the entire tree output.


## Ignore Manifest Fields

You can configure a list of yaml keys to ignore changes to via the `ignore_fields` field.
Set these for fields set by Operators or other processes in kubernetes and as such you don't want to update.

By default, the following control fields are ignored:
  - `status`
  - `metadata.finalizers`
  - `metadata.initializers`
  - `metadata.ownerReferences`
  - `metadata.creationTimestamp`
  - `metadata.generation`
  - `metadata.resourceVersion`
  - `metadata.uid`
  - `metadata.annotations.kubectl.kubernetes.io/last-applied-configuration`

These syntax matches the Terraform style flattened-map syntax, whereby keys are separated by `.` paths.

For example, to ignore the `annotations`, set the `ignore_fields` path to `metadata.annotations`:

```hcl
resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: v1
kind: ServiceAccount
metadata:
  name: name-here
  namespace: default
  annotations:
    this.should.be.ignored: "true"
YAML

    ignore_fields = ["metadata.annotations"]
}
```

For arrays, the syntax is indexed based on the element position. For example, to ignore the `caBundle` field in the
below manifest, would be: `webhooks.0.clientConfig.caBundle`

```hcl
resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: istio-sidecar-injector
webhooks:
  - clientConfig:
      caBundle: ""
YAML

    ignore_fields = ["webhooks.0.clientConfig.caBundle"]
}
```

More examples can be found in the provider tests.

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
