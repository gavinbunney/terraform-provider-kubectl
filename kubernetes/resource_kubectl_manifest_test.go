package kubernetes

import (
	"fmt"
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

func TestAccKubectlSensitiveFields_secret(t *testing.T) {

	yaml_body := `
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
type: Opaque
data:
  USER_NAME: YWRtaW4=
  PASSWORD: MWYyZDFlMmU2N2Rm
`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
type: Opaque
`),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_slice(t *testing.T) {

	yaml_body := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: name-here
spec:
  rules:
  - http:
      paths:
      - path: "/testpath"
        backend:
          serviceName: test
          servicePort: 80`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    sensitive_fields = [
      "spec.rules",
    ]

	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: name-here
spec:
  rules: (sensitive value)
`),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_unknown_field(t *testing.T) {

	yaml_body := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: name-here
spec:
  rules:
  - http:
      paths:
      - path: "/testpath"
        backend:
          serviceName: test
          servicePort: 80`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    sensitive_fields = [
      "spec.field.missing",
    ]

	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: name-here
spec:
  field:
    missing: (sensitive value)
  rules:
  - http:
      paths:
      - backend:
          serviceName: test
          servicePort: 80
        path: /testpath
`),
				),
			},
		},
	})
}
