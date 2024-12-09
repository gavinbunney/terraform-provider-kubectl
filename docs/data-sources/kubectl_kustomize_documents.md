# Data Source: kubectl_kustomize_documents

This provider provides a `data` resource `kubectl_kustomize_documents` to
render a kustomize target to a series of yaml documents. See https://kustomize.io/
for more info.

## Example Usage

```hcl
data "kubectl_kustomize_documents" "manifests" {
    target = "https://github.com/kubernetes-sigs/kustomize/examples/multibases?ref=v1.0.6"
}

resource "kubectl_manifest" "test" {
    count     = length(data.kubectl_file_documents.manifests.documents)
    yaml_body = element(data.kubectl_file_documents.manifests.documents, count.index)
}
```

## Attribute Reference

* `documents` - List of YAML documents (string).
