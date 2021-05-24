provider "kubectl" {}

resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: "/"
    azure/frontdoor: enabled
    azure/sensitive: "this is a big secret"
spec:
  ingressClassName: "nginx"
  rules:
  - host: "*.example.com"
    http:
      paths:
      - path: "/testpath"
        pathType: "Prefix"
        backend:
          service:
            name: test
            port:
              number: 80
    YAML

  sensitive_fields = [
    "metadata.annotations.azure/sensitive",
  ]
}

