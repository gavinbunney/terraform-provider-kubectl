---
id: data_kubectl_filename_list
title: kubectl_filename_list
---

This data source allows you to retrieve version information and other application properties of Bitbucket Server.

This provider provides a `data` resource `kubectl_filename_list` to enable ease of working with directories of kubernetes manifests.

## Example Usage

```hcl
data "kubectl_filename_list" "manifests" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    count = length(data.kubectl_filename_list.manifests.matches)
    yaml_body = file(element(data.kubectl_filename_list.manifests.matches, count.index))
}
```

## Attribute Reference

* `matches` - List of matching file names.
