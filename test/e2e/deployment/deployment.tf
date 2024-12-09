provider "kubectl" {}

resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment-name-here
  labels:
    app: deployment-name-here
spec:
  replicas: 1
  selector:
    matchLabels:
      app: deployment-name-here
  template:
    metadata:
      labels:
        app: deployment-name-here
    spec:
      containers:
      - name: main
        image: registry.k8s.io/pause:3.5
    YAML
}
