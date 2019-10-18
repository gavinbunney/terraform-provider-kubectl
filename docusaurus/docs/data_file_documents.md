---
id: data_kubectl_file_documents
title: kubectl_file_documents
---

This provider provides a `data` resource `kubectl_file_documents` to enable ease of splitting multi-document yaml content.

## Example Usage

```hcl
data "kubectl_file_documents" "manifests" {
    content = file("multi-doc-manifest.yaml")
}

resource "kubectl_manifest" "test" {
    count = length(data.kubectl_file_documents.manifests.documents)
    yaml_body = file(element(data.kubectl_file_documents.manifests.documents, count.index))
}
```

## Attribute Reference

* `documents` - List of YAML documents (string).
