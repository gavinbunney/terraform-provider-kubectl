---
id: data_kubectl_path_documents
title: kubectl_path_documents
---

This provider provides a `data` resource `kubectl_path_documents` to enable ease of splitting multi-document yaml content, from a collection of matching files.
Think of is as a combination of both `kubectl_filename_list` and `kubectl_file_documents`

## Example Usage

```hcl
data "kubectl_path_documents" "manifests" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    count = length(data.kubectl_path_documents.manifests.documents)
    yaml_body = element(data.kubectl_path_documents.manifests.documents, count.index)
}
```

## Attribute Reference

* `documents` - List of YAML documents (string).
