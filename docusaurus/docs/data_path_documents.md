---
id: data_kubectl_path_documents
title: kubectl_path_documents
---

This provider provides a `data` resource `kubectl_path_documents` to enable ease of splitting multi-document yaml content, from a collection of matching files.
Think of is as a combination of both `kubectl_filename_list` and `kubectl_file_documents`

`kubectl_path_documents` also supports rendering of Terraform Templates (similar to the template provider).
This gives you the flexibility of parameterizing your manifests, and loading & templating in a single command.

## Example Usage

### Load all manifest documents

```hcl
data "kubectl_path_documents" "manifests" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    count     = length(data.kubectl_path_documents.manifests.documents)
    yaml_body = element(data.kubectl_path_documents.manifests.documents, count.index)
}
```

### Example Template

```hcl
#
# Given the following YAML template
#
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    name: nginx
spec:
  containers:
  - name: nginx
    image: ${docker_image}
    ports:
    - containerPort: 80


#
# Load the yaml file, parsing the ${docker_image} variable
#
data "kubectl_path_documents" "manifests" {
    pattern = "./manifests/*.yaml"
    vars = {
        docker_image = "https://myregistry.example.com/nginx"
    }
}
```

### Example Template with Directives

Templates even support directives, meaning you can add conditions and another logic to your template:

```hcl
#
# Given the following YAML template
#
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    name: nginx
spec:
  containers:
  - name: nginx
    image: %{ if docker_image != "" }${docker_image}%{ else }default-nginx%{ endif }
    ports:
    - containerPort: 80


#
# Load the yaml file, parsing the ${docker_registry} variable, resulting in `default-nginx`
#
data "kubectl_path_documents" "manifests" {
    pattern = "./manifests/*.yaml"
    vars = {
        docker_image = ""
    }
}
```

## Argument Reference

* `pattern` - Required. Glob pattern to search for.
* `force_new` - Optional. Forces delete & create of resources if the `yaml_body` changes. Default `false`.
* `vars` - Optional. Map of variables to use when rendering the loaded documents as templates. Currently only strings are supported.
* `disable_template` - Optional. Flag to disable template parsing of the loaded documents.

## Attribute Reference

* `documents` - List of YAML documents (list[string]).
