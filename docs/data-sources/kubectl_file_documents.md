# Data Source: kubectl_file_documents

This provider provides a `data` resource `kubectl_file_documents` to enable ease of splitting multi-document yaml content.

## Example Usage

### Example Usage with for_each

The recommended approach is to use the `manifests` attribute and a `for_each` expression to apply the found manifests.
This ensures that any additional yaml documents or removals do not cause a large amount of terraform changes.

```hcl
data "kubectl_file_documents" "docs" {
    content = file("multi-doc-manifest.yaml")
}

resource "kubectl_manifest" "test" {
    for_each  = data.kubectl_file_documents.docs.manifests
    yaml_body = each.value
}
```

### Example Usage via count

Raw documents can also be accessed via the `documents` attribute.

```hcl
data "kubectl_file_documents" "docs" {
    content = file("multi-doc-manifest.yaml")
}

resource "kubectl_manifest" "test" {
    count     = length(data.kubectl_file_documents.docs.documents)
    yaml_body = element(data.kubectl_file_documents.docs.documents, count.index)
}
```

## Attribute Reference

* `manifests` - Map of YAML documents with key being the document id, and value being the document yaml. Best used with `for_each` expressions.
* `documents` - List of raw YAML documents (string). Best used with `count` expressions.
