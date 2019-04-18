provider "k8sraw" {}


resource "k8sraw_yaml" "account" {
    yaml_body = <<YAML
apiVersion: v1
kind: ServiceAccount
metadata:
  name: name-here
  namespace: default
    YAML
}

resource "k8sraw_yaml" "test" {
  depends_on = ["k8sraw_yaml.account"]
  yaml_body = <<YAML
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: name-here
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: name-here
    namespace: kube-system
    YAML
}
