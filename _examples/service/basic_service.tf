provider "kubectl" {}

resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: v1
kind: Service
metadata:
  name: name-here
spec:
  ports:
    - name: https
      port: 443
      targetPort: 8443
    - name: http
      port: 80
      targetPort: 9090
YAML
}

