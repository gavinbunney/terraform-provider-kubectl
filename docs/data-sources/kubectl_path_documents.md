# Data Source: kubectl_path_documents

This provider provides a `data` resource `kubectl_path_documents` to enable ease of splitting multi-document yaml content, from a collection of matching files.
Think of is as a combination of both `kubectl_filename_list` and `kubectl_file_documents`

`kubectl_path_documents` also supports rendering of Terraform Templates (similar to the template provider).
This gives you the flexibility of parameterizing your manifests, and loading & templating in a single command.

## Example Usage

### Load all manifest documents via for_each (recommended)

The recommended approach is to use the `manifests` attribute and a `for_each` expression to apply the found manifests.
This ensures that any additional yaml documents or removals do not cause a large amount of terraform changes.

```hcl
data "kubectl_path_documents" "docs" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    for_each  = data.kubectl_path_documents.docs.manifests
    yaml_body = each.value
}
```

### Load all manifest documents via count

Raw documents can also be accessed via the `documents` attribute.

```hcl
data "kubectl_path_documents" "docs" {
    pattern = "./manifests/*.yaml"
}

resource "kubectl_manifest" "test" {
    count     = length(data.kubectl_path_documents.docs.documents)
    yaml_body = element(data.kubectl_path_documents.docs.documents, count.index)
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

### Example Template with Looping Directive

Using a directive to generate multiple manifests is possible with using a combination of split and directive within the template:

```hcl
#
# Given the following YAML template
#
%{ for namespace in split(",", namespaces) }
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: myvolume-claim
  namespace: ${namespace}
spec:
  accessModes:
    - ReadWriteMany
  volumeMode: Filesystem
  resources:
    requests:
      storage: 100Gi
%{ endfor }

#
# Loading the document is a comma-separated list of namespace
#
data "kubectl_path_documents" "manifests" {
    pattern = "./manifests/*.yaml"
    vars = {
        namespaces = "dev,test,prod"
    }
}

#
# Results in 3 documents:
#
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: myvolume-claim
  namespace: dev
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: myvolume-claim
  namespace: test
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: myvolume-claim
  namespace: prod
```

## Argument Reference

* `pattern` - Required. Glob pattern to search for.
* `force_new` - Optional. Forces delete & create of resources if the `yaml_body` changes. Default `false`.
* `vars` - Optional. Map of variables to use when rendering the loaded documents as templates. Currently only strings are supported.
* `sensitive_vars` - Optional. Map of sensitive variables to use when rendering the loaded documents as templates. Merged with the `vars` attribute. Currently only strings are supported.
* `disable_template` - Optional. Flag to disable template parsing of the loaded documents.

## Attribute Reference

* `manifests` - Map of YAML documents with key being the document id, and value being the document yaml. Best used with `for_each` expressions.
* `documents` - List of YAML documents (list[string]). Best used with `count` expressions.
