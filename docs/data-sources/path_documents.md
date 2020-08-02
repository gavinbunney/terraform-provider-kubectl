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
* `disable_template` - Optional. Flag to disable template parsing of the loaded documents.

## Attribute Reference

* `documents` - List of YAML documents (list[string]).
