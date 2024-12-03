provider "kubectl" {}

resource "kubectl_manifest" "test" {
    yaml_body = <<YAML
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: daemonset-name-here
  namespace: kube-system
  labels:
    app.kubernetes.io/name: daemonset-name-here
    k8s-app: daemonset-name-here
spec:
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 10%
    type: RollingUpdate
  selector:
    matchLabels:
      k8s-app: daemonset-name-here
  template:
    metadata:
      labels:
        app.kubernetes.io/name: daemonset-name-here
        k8s-app: daemonset-name-here
    spec:
      priorityClassName: "system-node-critical"
      serviceAccountName: daemonset-name-here
      terminationGracePeriodSeconds: 10
      containers:
        - name: main
          image: registry.k8s.io/pause:3.5
    YAML
}

