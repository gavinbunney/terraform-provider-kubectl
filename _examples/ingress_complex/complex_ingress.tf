provider "kubectl" {}

resource "kubectl_manifest" "test" {
  yaml_body = <<YAML
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/affinity: cookie
    nginx.ingress.kubernetes.io/proxy-body-size: 0m
    nginx.ingress.kubernetes.io/rewrite-target: "/"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  name: name-here
spec:
  ingressClassName: "nginx"
  rules:
    - host: "bob.example.com"
      http:
        paths:
          - path: "/"
            pathType: "Prefix"
            backend:
              service:
                name: jerry
                port:
                  number: 80
  tls:
    - secretName: name-here
      hosts:
      - bob
YAML
}