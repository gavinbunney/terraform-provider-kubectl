package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestAccKubectlWithoutValidation(t *testing.T) {

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
    validate_schema = false

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
					resource.TestCheckResourceAttr("kubectl_manifest.test", "validate_schema", "false"),
				),
			},
		},
	})
}

func TestGetLiveManifestFilteredForUserProvidedOnly(t *testing.T) {
	testCases := []struct {
		description    string
		expectedString string
		userProvided   map[string]interface{}
		liveManifest   map[string]interface{}
		ignored        []string
	}{
		{
			description: "Simple map with string value",
			userProvided: map[string]interface{}{
				"test1": "test2",
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
			},
			expectedString: "test1=test2",
		},
		{
			// Ensure skippable fields are skipped
			description: "Simple map with string value and Skippable fields",
			userProvided: map[string]interface{}{
				"test1":           "test2",
				"resourceVersion": "1245",
			},
			liveManifest: map[string]interface{}{
				"test1":           "test2",
				"resourceVersion": "1245",
			},
			expectedString: "test1=test2",
		},
		{
			// Ensure ignored fields are skipped
			description: "Simple map with string value and ignored fields",
			userProvided: map[string]interface{}{
				"test1":      "test2",
				"ignoreThis": "1245",
			},
			liveManifest: map[string]interface{}{
				"test1":      "test2",
				"ignoreThis": "1245",
			},
			expectedString: "test1=test2",
			ignored:        []string{"ignoreThis"},
		},
		{
			// Ensure ignored sub fields are skipped
			description: "Simple map with string value and ignored fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "5432",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "1245",
				},
			},
			expectedString: "test1=test2",
			ignored:        []string{"ignore.this"},
		},
		{
			// Ensure ignored sub fields are skipped
			description: "Simple map with string ignore nested fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "5432",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "1245",
				},
			},
			expectedString: "test1=test2",
			ignored:        []string{"ignore"},
		},
		{
			// Ensure ignored sub fields are skipped
			description: "Simple map with string ignore highly nested fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "5432",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]interface{}{
					"this": "1245",
					"also": map[string]string{
						"these": "9876",
					},
				},
			},
			expectedString: "test1=test2",
			ignored:        []string{"ignore"},
		},
		{
			// Ensure nested `map[string]string` are supported
			description: "Map with nested map[string]string",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob": "bill",
				},
			},
			expectedString: "nest.bob=bill,test1=test2",
		},
		{
			// Ensure nested `map[string]string` with different ordering are supported
			description: "Map with nested map[string]string with different ordering",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob1": "bill",
					"bob2": "bill",
					"bob3": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob2": "bill",
					"bob1": "bill",
					"bob3": "bill",
				},
			},
			expectedString: "nest.bob1=bill,nest.bob2=bill,nest.bob3=bill,test1=test2",
		},
		{
			description: "Map with nested map[string]string with nested array",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						"a",
						"b",
						"c",
					},
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						"c",
						"b",
						"a",
					},
				},
			},
			expectedString: "nest.bob1.#=3,nest.bob1.0=c,nest.bob1.1=b,nest.bob1.2=a,test1=test2",
		},
		{
			description: "Map with nested map[string]string with nested array and nested map",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						map[string]string{
							"1": "1",
							"2": "2",
							"3": "3",
						},
						map[string]interface{}{
							"1": 1,
							"2": 2,
							"3": 3,
						},
					},
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						map[string]string{
							"2": "2",
							"1": "1",
							"3": "3",
						},
						map[string]interface{}{
							"2": 2,
							"1": 1,
							"3": 3,
						},
					},
				},
			},
			expectedString: "nest.bob1.#=2,nest.bob1.0.1=1,nest.bob1.0.2=2,nest.bob1.0.3=3,nest.bob1.1.1=1,nest.bob1.1.2=2,nest.bob1.1.3=3,test1=test2",
		},
		{
			// Ensure ordering of the fields doesn't affect matching
			description: "Different Ordering",
			userProvided: map[string]interface{}{
				"ztest1": "test2",
				"afield": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
				"ztest1": "test2",
			},
			expectedString: "afield=test2,ztest1=test2",
		},
		{
			// Ensure nested arrays are handled
			description: "Nested Array",
			userProvided: map[string]interface{}{
				"ztest1": []string{
					"1", "2",
				},
				"afield": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
				"ztest1": []string{
					"1", "2",
				},
			},
			expectedString: "afield=test2,ztest1.#=2,ztest1.0=1,ztest1.1=2",
		},
		{
			// Ensure fields added to the `liveManifest` which aren't present in the `originl` are ignored
			description: "Ignore additional fields",
			userProvided: map[string]interface{}{
				"afield": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
				"ztest1": []string{
					"1", "2",
				},
			},
			expectedString: "afield=test2",
		},
		{
			// Ensure that fields present in the `userProvided` but missing in the `liveManifest` are skipped
			description: "Handle removed fields",
			userProvided: map[string]interface{}{
				"afield":   "test2",
				"igetlost": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
			},
			expectedString: "afield=test2",
		},
		{
			description: "Handle integers",
			userProvided: map[string]interface{}{
				"afield": 1,
			},
			liveManifest: map[string]interface{}{
				"afield": 1,
			},
			expectedString: "afield=1",
		},
		{
			// Ensure that the updated value for `afield` on the `liveManifest` object is taken
			description: "Handle updated field. Expect liveManifest value to be shown",
			userProvided: map[string]interface{}{
				"afield": 1,
			},
			liveManifest: map[string]interface{}{
				"afield": 2,
			},
			expectedString: "afield=2",
		},
		{
			// Ensure that the updated value fo the `liveManifest` object is taken for the `willchange` field
			description: "Map with nested map[string]string with updated field",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"willchange": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"nest": map[string]string{
					"willchange": "updatedbill",
				},
			},
			expectedString: "nest.willchange=updatedbill",
		},
		{
			// Ensure that both fields are tracked in the output
			description: "Handle duplicate name fields in nested maps",
			userProvided: map[string]interface{}{
				"atest": "test",
				"nest": map[string]string{
					"atest": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"atest": "test",
				"nest": map[string]string{
					"atest": "bill",
				},
			},
			expectedString: "atest=test,nest.atest=bill",
		},
		{
			description: "Map with nested map[string]string with annotations",
			userProvided: map[string]interface{}{
				"atest": "test",
				"meta": map[string]interface{}{
					"annotations": map[string]string{
						"helm.sh/hook": "crd-install",
					},
				},
			},
			liveManifest: map[string]interface{}{
				"atest": "test",
				"meta": map[string]interface{}{
					"annotations": map[string]string{
						"helm.sh/hook": "crd-install",
					},
				},
			},
			expectedString: "atest=test,meta.annotations.helm.sh/hook=crd-install",
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			userProvided := &unstructured.Unstructured{Object: tcase.userProvided}
			liveManifest := &unstructured.Unstructured{Object: tcase.liveManifest}
			result, err := getLiveManifestFilteredForUserProvidedOnlyWithIgnoredFields(tcase.ignored, userProvided, liveManifest)
			assert.NoError(t, err, "Expect compareMaps to succeed")

			assert.Equal(t, tcase.expectedString, result, "Expect the builder output to match")
		})
	}
}
