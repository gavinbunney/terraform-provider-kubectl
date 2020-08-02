provider "kubectl" {}

resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: name-here
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: "/"
    azure/frontdoor: enabled
    azure/sensitive: "this is a big secret"
spec:
  rules:
  - http:
      paths:
      - path: "/testpath"
        backend:
          serviceName: test
          servicePort: 80
    YAML

  sensitive_fields = [
    "metadata.annotations.azure/sensitive",
  ]
}

