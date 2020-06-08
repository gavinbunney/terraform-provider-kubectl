package kubernetes

import (
	"github.com/hashicorp/terraform/helper/resource"
	"os"
	"regexp"
	"testing"
)

func TestKubectlManifest_RetryOnFailure(t *testing.T) {
	_ = os.Setenv("KUBECTL_PROVIDER_APPLY_RETRY_COUNT", "5")

	config := `
resource "kubectl_manifest" "test" {
	yaml_body = <<YAML
apiVersion: v1
kind: Ingress
YAML
}
	`

	expectedError, _ := regexp.Compile(".*failed to create kubernetes.*")
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				ExpectError: expectedError,
				Config:      config,
			},
		},
	})
}

func TestAccKubectlUnknownNamespace(t *testing.T) {

	config := `
resource "kubectl_manifest" "test" {
	yaml_body = <<EOT
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: name-here
  namespace: this-doesnt-exist
spec:
  rules:
  - http:
      paths:
      - path: "/testpath"
        backend:
          serviceName: test
          servicePort: 80
	EOT
		}
`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("\"this-doesnt-exist\" not found"),
			},
		},
	})
}
