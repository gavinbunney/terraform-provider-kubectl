package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

var kustTargetUrl = "https://github.com/kubernetes-sigs/kustomize/examples/multibases?ref=v3.3.1"

func TestAccKubectlDataSourceKustomizeDocuments_url(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: kubectlKustomizeDocumentsConfig(kustTargetUrl),
				Check:  resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.#", "3"),
			},
		},
	})
}

func TestAccKubectlDataSourceKustomizeDocuments_localDir(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: kubectlKustomizeDocumentsConfig("../test/data/kustomize/helloWorld"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.#", "3"),
					resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.0", `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: hello
  name: the-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
        deployment: hello
    spec:
      containers:
      - command:
        - /hello
        - --port=8080
        - --enableRiskyFeature=$(ENABLE_RISKY)
        env:
        - name: ALT_GREETING
          valueFrom:
            configMapKeyRef:
              key: altGreeting
              name: the-map
        - name: ENABLE_RISKY
          valueFrom:
            configMapKeyRef:
              key: enableRisky
              name: the-map
        image: monopole/hello:1
        name: the-container
        ports:
        - containerPort: 8080
`),
					resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.1", `apiVersion: v1
kind: Service
metadata:
  labels:
    app: hello
  name: the-service
spec:
  ports:
  - port: 8666
    protocol: TCP
    targetPort: 8080
  selector:
    app: hello
    deployment: hello
  type: LoadBalancer
`),
					resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.2", `apiVersion: v1
data:
  altGreeting: Good Morning!
  enableRisky: "false"
kind: ConfigMap
metadata:
  labels:
    app: hello
  name: the-map
`),
				),
			},
		},
	})
}

func kubectlKustomizeDocumentsConfig(target string) string {
	return fmt.Sprintf(`
data "kubectl_kustomize_documents" "test" {
	target = "%s"
}
	`, target)
}
