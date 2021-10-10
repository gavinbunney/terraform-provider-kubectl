package kubernetes

import (
	"bytes"
	"fmt"
	"github.com/gavinbunney/terraform-provider-kubectl/yaml"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
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
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
  namespace: this-doesnt-exist
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

func TestAccKubectlOverrideNamespace(t *testing.T) {

	namespace := "dev-" + acctest.RandString(10)
	yaml_body := `
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  namespace: prod 
type: Opaque
data:
`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "ns" {
	yaml_body = <<EOT
apiVersion: v1
kind: Namespace
metadata:
  name: %s
    EOT
}

resource "kubectl_manifest" "test" {
	depends_on = [kubectl_manifest.ns]
    override_namespace = "%s"
	yaml_body = <<EOT
%s
	EOT
		}
`, namespace, namespace, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "override_namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", fmt.Sprintf(`apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
  namespace: %s
type: Opaque
`, namespace)),

					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_incluster", getFingerprint(fmt.Sprintf(`apiVersion=v1,kind=Secret,metadata.name=mysecret,metadata.namespace=%s,type=Opaque`, namespace))),
				),
			},
		},
	})
}

func TestAccKubectlSetNamespace(t *testing.T) {

	namespace := "dev-" + acctest.RandString(10)
	yaml_body := `
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
type: Opaque
data:
`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "ns" {
	yaml_body = <<EOT
apiVersion: v1
kind: Namespace
metadata:
  name: %s
    EOT
}

resource "kubectl_manifest" "test" {
    depends_on = [kubectl_manifest.ns]
    override_namespace = "%s"
	yaml_body = <<EOT
%s
	EOT
		}
`, namespace, namespace, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "id", "/api/v1/namespaces/"+namespace+"/secrets/mysecret"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "override_namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", fmt.Sprintf(`apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
  namespace: %s
type: Opaque
`, namespace)),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_incluster", getFingerprint(fmt.Sprintf(`apiVersion=v1,kind=Secret,metadata.name=mysecret,metadata.namespace=%s,type=Opaque`, namespace))),
				),
			},
		},
	})
}

func TestAccKubectlSetNamespace_nonnamespaced_resource(t *testing.T) {

	namespace := "dev-" + acctest.RandString(10)
	yaml_body := fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mysuperrole-%s
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "watch", "list"]
`, namespace)

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    override_namespace = "%s"
	yaml_body = <<EOT
%s
	EOT
		}
`, namespace, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "override_namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mysuperrole-%s
  namespace: %s
rules:
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - get
  - watch
  - list
`, namespace, namespace)),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_incluster", getFingerprint(fmt.Sprintf(`apiVersion=rbac.authorization.k8s.io/v1,kind=ClusterRole,metadata.name=mysuperrole-%s,rules.#=1,rules.0.apiGroups.#=1,rules.0.apiGroups.0=,rules.0.resources.#=1,rules.0.resources.0=secrets,rules.0.verbs.#=3,rules.0.verbs.0=get,rules.0.verbs.1=watch,rules.0.verbs.2=list`, namespace))),
				),
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
  namespace: default
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
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", "default"),
					resource.TestCheckNoResourceAttr("kubectl_manifest.test", "override_namespace"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
  namespace: default
type: Opaque
`),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_slice(t *testing.T) {

	yaml_body := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
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
              number: 80`

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
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: nginx
  rules: (sensitive value)
`),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_unknown_field(t *testing.T) {

	yaml_body := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
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
              number: 80`

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
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: nginx
  rules:
  - host: '*.example.com'
    http:
      paths:
      - backend:
          service:
            name: test
            port:
              number: 80
        path: /testpath
        pathType: Prefix
`),
				),
			},
		},
	})
}

func TestAccKubectlWithoutValidation(t *testing.T) {

	yaml_body := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
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
              number: 80`

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
		description         string
		expectedFields      string
		expectedFingerprint string
		userProvided        map[string]interface{}
		liveManifest        map[string]interface{}
		ignored             []string
		expectedDrift       bool
	}{
		{
			description: "Simple map with string value",
			userProvided: map[string]interface{}{
				"test1": "test2",
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
			},
			expectedFields:      "test1=test2",
			expectedFingerprint: "9369bac4ce5d012a79110117b871e20bb3484dab079d1471ee5981da42fb4a30",
			expectedDrift:       false,
		},
		{
			// Ensure skippable fields are skipped
			description: "Simple map with string value and Skippable fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"metadata": map[string]interface{}{
					"resourceVersion": "1245",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"metadata": map[string]interface{}{
					"resourceVersion": "1245",
				},
			},
			expectedFields:      "test1=test2",
			expectedFingerprint: "9369bac4ce5d012a79110117b871e20bb3484dab079d1471ee5981da42fb4a30",
			expectedDrift:       false,
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
			expectedFields:      "test1=test2",
			expectedFingerprint: "9369bac4ce5d012a79110117b871e20bb3484dab079d1471ee5981da42fb4a30",
			ignored:             []string{"ignoreThis"},
			expectedDrift:       false,
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
			expectedFields:      "test1=test2",
			expectedFingerprint: "9369bac4ce5d012a79110117b871e20bb3484dab079d1471ee5981da42fb4a30",
			ignored:             []string{"ignore.this"},
			expectedDrift:       false,
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
			expectedFields:      "test1=test2",
			expectedFingerprint: "9369bac4ce5d012a79110117b871e20bb3484dab079d1471ee5981da42fb4a30",
			ignored:             []string{"ignore"},
			expectedDrift:       false,
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
			expectedFields:      "test1=test2",
			expectedFingerprint: "9369bac4ce5d012a79110117b871e20bb3484dab079d1471ee5981da42fb4a30",
			ignored:             []string{"ignore"},
			expectedDrift:       false,
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
			expectedFields:      "nest.bob=bill,test1=test2",
			expectedFingerprint: "3101bf7d8f32b48993efa15e0fdd439237e63ef093d23e92deb9b8485e3faa03",
			expectedDrift:       false,
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
			expectedFields:      "nest.bob1=bill,nest.bob2=bill,nest.bob3=bill,test1=test2",
			expectedFingerprint: "0ad7f5a7682d24a2105a457f9093ab406d9a3c92a14d1e67e25ac0a1fea79ca9",
			expectedDrift:       false,
		},
		{
			description: "Map with nested map[string]string with nested slice",
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
			expectedFields:      "nest.bob1.#=3,nest.bob1.0=c,nest.bob1.1=b,nest.bob1.2=a,test1=test2",
			expectedFingerprint: "7c234055ab3af4bfc4541b4f11ebe41f089f65ff2276454783fd066c4e890bb9",
			expectedDrift:       true,
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
			expectedFields:      "nest.bob1.#=2,nest.bob1.0.1=1,nest.bob1.0.2=2,nest.bob1.0.3=3,nest.bob1.1.1=1,nest.bob1.1.2=2,nest.bob1.1.3=3,test1=test2",
			expectedFingerprint: "f3efd8721cbfa6421a4230c6fffdac94d63a51e57097a45979972e6654a992da",
			expectedDrift:       false,
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
			expectedFields:      "afield=test2,ztest1=test2",
			expectedFingerprint: "6ddd159d93a55b78442c74cacfff5a2afb04ead770f87ac0af1b7471e71ddead",
			expectedDrift:       false,
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
			expectedFields:      "afield=test2,ztest1.#=2,ztest1.0=1,ztest1.1=2",
			expectedFingerprint: "d09ba05ec3c744be7174243acfd2370a6d0dabfbe7980bc5ee02c0790d383960",
			expectedDrift:       false,
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
			expectedFields:      "afield=test2",
			expectedFingerprint: "18cf5c716095e42b64da5d4929c605022b6799fb3866bf9f1d12f4e30d40c185",
			expectedDrift:       false,
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
			expectedFields:      "afield=test2",
			expectedFingerprint: "18cf5c716095e42b64da5d4929c605022b6799fb3866bf9f1d12f4e30d40c185",
			expectedDrift:       true,
		},
		{
			description: "Handle integers",
			userProvided: map[string]interface{}{
				"afield": 1,
			},
			liveManifest: map[string]interface{}{
				"afield": 1,
			},
			expectedFields:      "afield=1",
			expectedFingerprint: "b4636ba2492c0110641065ccef19d47ac718f317d4541608587954c924e9d521",
			expectedDrift:       false,
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
			expectedFields:      "afield=2",
			expectedFingerprint: "e99abf0780d7d15a43b75f39a1e82a7ec6342d8efb5b077c46a6b85ec2b2efcb",
			expectedDrift:       true,
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
			expectedFields:      "nest.willchange=updatedbill",
			expectedFingerprint: "ebbab7294a88055e1b6af53fdb0da8366054e1c7b88d79294d8424b85d4eb769",
			expectedDrift:       true,
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
			expectedFields:      "atest=test,nest.atest=bill",
			expectedFingerprint: "0a926a0980a93f7360e2badadbb8c362dd345fd53c641d1096e5680fd66c11e3",
			expectedDrift:       false,
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
			expectedFields:      "atest=test,meta.annotations.helm.sh/hook=crd-install",
			expectedFingerprint: "5d9a5cd23ce01763e52f171e6bf2d98ca3cfed982974579af4c011ff6010694f",
			expectedDrift:       false,
		},
		{
			description: "Map with empty annotations in user manifest",
			userProvided: map[string]interface{}{
				"atest": "test",
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{},
				},
			},
			liveManifest: map[string]interface{}{
				"atest": "test",
				"metadata": map[string]interface{}{
					"annotations": map[string]string{
						"kubectl.kubernetes.io/last-applied-configuration": "{\"should-be-ignored\"}",
					},
				},
			},
			expectedFields:      "atest=test",
			expectedFingerprint: "df296364dd3346f0aa05c63f0b0df19b7aa850e44e9f4a80cf6ac06a889d9868",
			expectedDrift:       false,
		},
		{
			description:         "Deployment manifest without changes",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        loadRealDeploymentManifest().Raw.Object,
			expectedFields:      "apiVersion=apps/v1,kind=Deployment,metadata.annotations.artifact.spinnaker.io/location=service-prod,metadata.annotations.artifact.spinnaker.io/name=testapp,metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,metadata.annotations.artifact.spinnaker.io/version=,metadata.annotations.deployment.kubernetes.io/revision=3,metadata.annotations.moniker.spinnaker.io/application=testapp,metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,metadata.labels.app.kubernetes.io/managed-by=spinnaker,metadata.labels.app.kubernetes.io/name=testapp,metadata.name=testapp,metadata.namespace=service-prod,spec.progressDeadlineSeconds=600,spec.replicas=1,spec.revisionHistoryLimit=3,spec.selector.matchLabels.app.kubernetes.io/name=testapp,spec.strategy.rollingUpdate.maxSurge=25%,spec.strategy.rollingUpdate.maxUnavailable=25%,spec.strategy.type=RollingUpdate,spec.template.metadata.annotations.artifact.spinnaker.io/location=service-prod,spec.template.metadata.annotations.artifact.spinnaker.io/name=testapp,spec.template.metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,spec.template.metadata.annotations.artifact.spinnaker.io/version=,spec.template.metadata.annotations.kubectl.terraform.test/envoy=true,spec.template.metadata.annotations.kubectl.terraform.test/telegraf=statsd,spec.template.metadata.annotations.moniker.spinnaker.io/application=testapp,spec.template.metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,spec.template.metadata.creationTimestamp=<invalid reflect.Value>,spec.template.metadata.labels.app.kubernetes.io/managed-by=spinnaker,spec.template.metadata.labels.app.kubernetes.io/name=testapp,spec.template.metadata.labels.app.kubernetes.io/version=1.0.918,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=kubectl.terraform.test/instance-class,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=In,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=cpu,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=2,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=topology.kubernetes.io/zone,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=25,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=kubernetes.io/hostname,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=100,spec.template.spec.containers.#=1,spec.template.spec.containers.0.envFrom.#=1,spec.template.spec.containers.0.envFrom.0.configMapRef.name=testapp-config-v001,spec.template.spec.containers.0.image=my-registry/application/testapp:1.0.918,spec.template.spec.containers.0.imagePullPolicy=Always,spec.template.spec.containers.0.name=application,spec.template.spec.containers.0.resources.limits.cpu=1,spec.template.spec.containers.0.resources.limits.memory=1Gi,spec.template.spec.containers.0.resources.requests.cpu=50m,spec.template.spec.containers.0.resources.requests.memory=256Mi,spec.template.spec.containers.0.terminationMessagePath=/dev/termination-log,spec.template.spec.containers.0.terminationMessagePolicy=File,spec.template.spec.containers.0.volumeMounts.#=1,spec.template.spec.containers.0.volumeMounts.0.mountPath=/share,spec.template.spec.containers.0.volumeMounts.0.name=shared-data,spec.template.spec.dnsPolicy=ClusterFirst,spec.template.spec.priorityClassName=spinnaker-managed-deployment,spec.template.spec.restartPolicy=Always,spec.template.spec.schedulerName=default-scheduler,spec.template.spec.securityContext.fsGroup=1100,spec.template.spec.securityContext.runAsUser=1100,spec.template.spec.serviceAccount=testapp,spec.template.spec.serviceAccountName=testapp,spec.template.spec.terminationGracePeriodSeconds=30,spec.template.spec.volumes.#=2,spec.template.spec.volumes.0.name=cert,spec.template.spec.volumes.0.secret.defaultMode=420,spec.template.spec.volumes.0.secret.secretName=testapp-cert,spec.template.spec.volumes.1.name=shared-data",
			expectedFingerprint: "1f1c6ba90e13b44ec07790933eef29604f8ad74be76bd38eff1bf7411f479eb5",
			expectedDrift:       false,
		},
		{
			description:         "Deployment manifest with changes should show drift",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        withAlteredField(loadRealDeploymentManifest(), "name-changed", "metadata", "name").Raw.Object,
			expectedFields:      "apiVersion=apps/v1,kind=Deployment,metadata.annotations.artifact.spinnaker.io/location=service-prod,metadata.annotations.artifact.spinnaker.io/name=testapp,metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,metadata.annotations.artifact.spinnaker.io/version=,metadata.annotations.deployment.kubernetes.io/revision=3,metadata.annotations.moniker.spinnaker.io/application=testapp,metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,metadata.labels.app.kubernetes.io/managed-by=spinnaker,metadata.labels.app.kubernetes.io/name=testapp,metadata.name=name-changed,metadata.namespace=service-prod,spec.progressDeadlineSeconds=600,spec.replicas=1,spec.revisionHistoryLimit=3,spec.selector.matchLabels.app.kubernetes.io/name=testapp,spec.strategy.rollingUpdate.maxSurge=25%,spec.strategy.rollingUpdate.maxUnavailable=25%,spec.strategy.type=RollingUpdate,spec.template.metadata.annotations.artifact.spinnaker.io/location=service-prod,spec.template.metadata.annotations.artifact.spinnaker.io/name=testapp,spec.template.metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,spec.template.metadata.annotations.artifact.spinnaker.io/version=,spec.template.metadata.annotations.kubectl.terraform.test/envoy=true,spec.template.metadata.annotations.kubectl.terraform.test/telegraf=statsd,spec.template.metadata.annotations.moniker.spinnaker.io/application=testapp,spec.template.metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,spec.template.metadata.creationTimestamp=<invalid reflect.Value>,spec.template.metadata.labels.app.kubernetes.io/managed-by=spinnaker,spec.template.metadata.labels.app.kubernetes.io/name=testapp,spec.template.metadata.labels.app.kubernetes.io/version=1.0.918,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=kubectl.terraform.test/instance-class,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=In,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=cpu,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=2,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=topology.kubernetes.io/zone,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=25,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=kubernetes.io/hostname,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=100,spec.template.spec.containers.#=1,spec.template.spec.containers.0.envFrom.#=1,spec.template.spec.containers.0.envFrom.0.configMapRef.name=testapp-config-v001,spec.template.spec.containers.0.image=my-registry/application/testapp:1.0.918,spec.template.spec.containers.0.imagePullPolicy=Always,spec.template.spec.containers.0.name=application,spec.template.spec.containers.0.resources.limits.cpu=1,spec.template.spec.containers.0.resources.limits.memory=1Gi,spec.template.spec.containers.0.resources.requests.cpu=50m,spec.template.spec.containers.0.resources.requests.memory=256Mi,spec.template.spec.containers.0.terminationMessagePath=/dev/termination-log,spec.template.spec.containers.0.terminationMessagePolicy=File,spec.template.spec.containers.0.volumeMounts.#=1,spec.template.spec.containers.0.volumeMounts.0.mountPath=/share,spec.template.spec.containers.0.volumeMounts.0.name=shared-data,spec.template.spec.dnsPolicy=ClusterFirst,spec.template.spec.priorityClassName=spinnaker-managed-deployment,spec.template.spec.restartPolicy=Always,spec.template.spec.schedulerName=default-scheduler,spec.template.spec.securityContext.fsGroup=1100,spec.template.spec.securityContext.runAsUser=1100,spec.template.spec.serviceAccount=testapp,spec.template.spec.serviceAccountName=testapp,spec.template.spec.terminationGracePeriodSeconds=30,spec.template.spec.volumes.#=2,spec.template.spec.volumes.0.name=cert,spec.template.spec.volumes.0.secret.defaultMode=420,spec.template.spec.volumes.0.secret.secretName=testapp-cert,spec.template.spec.volumes.1.name=shared-data",
			expectedFingerprint: "e468a45e883f696efa3afe82a76268be4b79dcf40efc26b033d9c8b3b454a829",
			expectedDrift:       true,
		},
		{
			description:         "Deployment manifest with generation changes should not show drift",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        withAlteredField(loadRealDeploymentManifest(), "123", "metadata", "generation").Raw.Object,
			expectedFields:      "apiVersion=apps/v1,kind=Deployment,metadata.annotations.artifact.spinnaker.io/location=service-prod,metadata.annotations.artifact.spinnaker.io/name=testapp,metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,metadata.annotations.artifact.spinnaker.io/version=,metadata.annotations.deployment.kubernetes.io/revision=3,metadata.annotations.moniker.spinnaker.io/application=testapp,metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,metadata.labels.app.kubernetes.io/managed-by=spinnaker,metadata.labels.app.kubernetes.io/name=testapp,metadata.name=testapp,metadata.namespace=service-prod,spec.progressDeadlineSeconds=600,spec.replicas=1,spec.revisionHistoryLimit=3,spec.selector.matchLabels.app.kubernetes.io/name=testapp,spec.strategy.rollingUpdate.maxSurge=25%,spec.strategy.rollingUpdate.maxUnavailable=25%,spec.strategy.type=RollingUpdate,spec.template.metadata.annotations.artifact.spinnaker.io/location=service-prod,spec.template.metadata.annotations.artifact.spinnaker.io/name=testapp,spec.template.metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,spec.template.metadata.annotations.artifact.spinnaker.io/version=,spec.template.metadata.annotations.kubectl.terraform.test/envoy=true,spec.template.metadata.annotations.kubectl.terraform.test/telegraf=statsd,spec.template.metadata.annotations.moniker.spinnaker.io/application=testapp,spec.template.metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,spec.template.metadata.creationTimestamp=<invalid reflect.Value>,spec.template.metadata.labels.app.kubernetes.io/managed-by=spinnaker,spec.template.metadata.labels.app.kubernetes.io/name=testapp,spec.template.metadata.labels.app.kubernetes.io/version=1.0.918,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=kubectl.terraform.test/instance-class,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=In,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=cpu,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=2,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=topology.kubernetes.io/zone,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=25,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=kubernetes.io/hostname,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=100,spec.template.spec.containers.#=1,spec.template.spec.containers.0.envFrom.#=1,spec.template.spec.containers.0.envFrom.0.configMapRef.name=testapp-config-v001,spec.template.spec.containers.0.image=my-registry/application/testapp:1.0.918,spec.template.spec.containers.0.imagePullPolicy=Always,spec.template.spec.containers.0.name=application,spec.template.spec.containers.0.resources.limits.cpu=1,spec.template.spec.containers.0.resources.limits.memory=1Gi,spec.template.spec.containers.0.resources.requests.cpu=50m,spec.template.spec.containers.0.resources.requests.memory=256Mi,spec.template.spec.containers.0.terminationMessagePath=/dev/termination-log,spec.template.spec.containers.0.terminationMessagePolicy=File,spec.template.spec.containers.0.volumeMounts.#=1,spec.template.spec.containers.0.volumeMounts.0.mountPath=/share,spec.template.spec.containers.0.volumeMounts.0.name=shared-data,spec.template.spec.dnsPolicy=ClusterFirst,spec.template.spec.priorityClassName=spinnaker-managed-deployment,spec.template.spec.restartPolicy=Always,spec.template.spec.schedulerName=default-scheduler,spec.template.spec.securityContext.fsGroup=1100,spec.template.spec.securityContext.runAsUser=1100,spec.template.spec.serviceAccount=testapp,spec.template.spec.serviceAccountName=testapp,spec.template.spec.terminationGracePeriodSeconds=30,spec.template.spec.volumes.#=2,spec.template.spec.volumes.0.name=cert,spec.template.spec.volumes.0.secret.defaultMode=420,spec.template.spec.volumes.0.secret.secretName=testapp-cert,spec.template.spec.volumes.1.name=shared-data",
			expectedFingerprint: "1f1c6ba90e13b44ec07790933eef29604f8ad74be76bd38eff1bf7411f479eb5",
			expectedDrift:       false,
		},
		{
			description:         "Deployment manifest with kubectl annotation changes should not show drift",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        withAlteredField(loadRealDeploymentManifest(), "changed", "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration").Raw.Object,
			expectedFields:      "apiVersion=apps/v1,kind=Deployment,metadata.annotations.artifact.spinnaker.io/location=service-prod,metadata.annotations.artifact.spinnaker.io/name=testapp,metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,metadata.annotations.artifact.spinnaker.io/version=,metadata.annotations.deployment.kubernetes.io/revision=3,metadata.annotations.moniker.spinnaker.io/application=testapp,metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,metadata.labels.app.kubernetes.io/managed-by=spinnaker,metadata.labels.app.kubernetes.io/name=testapp,metadata.name=testapp,metadata.namespace=service-prod,spec.progressDeadlineSeconds=600,spec.replicas=1,spec.revisionHistoryLimit=3,spec.selector.matchLabels.app.kubernetes.io/name=testapp,spec.strategy.rollingUpdate.maxSurge=25%,spec.strategy.rollingUpdate.maxUnavailable=25%,spec.strategy.type=RollingUpdate,spec.template.metadata.annotations.artifact.spinnaker.io/location=service-prod,spec.template.metadata.annotations.artifact.spinnaker.io/name=testapp,spec.template.metadata.annotations.artifact.spinnaker.io/type=kubernetes/deployment,spec.template.metadata.annotations.artifact.spinnaker.io/version=,spec.template.metadata.annotations.kubectl.terraform.test/envoy=true,spec.template.metadata.annotations.kubectl.terraform.test/telegraf=statsd,spec.template.metadata.annotations.moniker.spinnaker.io/application=testapp,spec.template.metadata.annotations.moniker.spinnaker.io/cluster=deployment testapp,spec.template.metadata.creationTimestamp=<invalid reflect.Value>,spec.template.metadata.labels.app.kubernetes.io/managed-by=spinnaker,spec.template.metadata.labels.app.kubernetes.io/name=testapp,spec.template.metadata.labels.app.kubernetes.io/version=1.0.918,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=kubectl.terraform.test/instance-class,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=In,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=1,spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=cpu,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=2,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=topology.kubernetes.io/zone,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=25,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=app.kubernetes.io/name,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=In,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=1,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=testapp,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=kubernetes.io/hostname,spec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=100,spec.template.spec.containers.#=1,spec.template.spec.containers.0.envFrom.#=1,spec.template.spec.containers.0.envFrom.0.configMapRef.name=testapp-config-v001,spec.template.spec.containers.0.image=my-registry/application/testapp:1.0.918,spec.template.spec.containers.0.imagePullPolicy=Always,spec.template.spec.containers.0.name=application,spec.template.spec.containers.0.resources.limits.cpu=1,spec.template.spec.containers.0.resources.limits.memory=1Gi,spec.template.spec.containers.0.resources.requests.cpu=50m,spec.template.spec.containers.0.resources.requests.memory=256Mi,spec.template.spec.containers.0.terminationMessagePath=/dev/termination-log,spec.template.spec.containers.0.terminationMessagePolicy=File,spec.template.spec.containers.0.volumeMounts.#=1,spec.template.spec.containers.0.volumeMounts.0.mountPath=/share,spec.template.spec.containers.0.volumeMounts.0.name=shared-data,spec.template.spec.dnsPolicy=ClusterFirst,spec.template.spec.priorityClassName=spinnaker-managed-deployment,spec.template.spec.restartPolicy=Always,spec.template.spec.schedulerName=default-scheduler,spec.template.spec.securityContext.fsGroup=1100,spec.template.spec.securityContext.runAsUser=1100,spec.template.spec.serviceAccount=testapp,spec.template.spec.serviceAccountName=testapp,spec.template.spec.terminationGracePeriodSeconds=30,spec.template.spec.volumes.#=2,spec.template.spec.volumes.0.name=cert,spec.template.spec.volumes.0.secret.defaultMode=420,spec.template.spec.volumes.0.secret.secretName=testapp-cert,spec.template.spec.volumes.1.name=shared-data",
			expectedFingerprint: "1f1c6ba90e13b44ec07790933eef29604f8ad74be76bd38eff1bf7411f479eb5",
			expectedDrift:       false,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			userProvided := yaml.NewFromUnstructured(&unstructured.Unstructured{Object: tcase.userProvided})
			liveManifest := yaml.NewFromUnstructured(&unstructured.Unstructured{Object: tcase.liveManifest})

			var out bytes.Buffer
			log.SetOutput(&out)
			defer log.SetOutput(os.Stderr)

			fields := getLiveManifestFields_WithIgnoredFields(tcase.ignored, userProvided, liveManifest)
			assert.Equal(t, tcase.expectedFields, fields, "Expect the builder output to match")
			fingerprint := getFingerprint(fields)
			assert.Equal(t, tcase.expectedFingerprint, fingerprint, "Expect the builder output to match")

			if tcase.expectedDrift {
				assert.Contains(t, out.String(), "yaml drift", "Should have drift detected")
			} else {
				assert.NotContains(t, out.String(), "yaml drift", "Should not have drift detected")
			}
		})
	}
}

func TestAccKubectlServerSideValidationFailure(t *testing.T) {

	config := `
resource "kubectl_manifest" "test" {
  yaml_body = <<YAML
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress
spec:
  rules:
    - host: "test-a.proxypile.tk"
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: nginx.test-a.svc.cluster.local
                port:
                  number: 8080
YAML
}
`
	expectedError, _ := regexp.Compile(".*Invalid value: \"nginx.test-a.svc.cluster.local\": a DNS-1035 label must consist of lower case alphanumeric characters.*")
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

func withAlteredField(manifest *yaml.Manifest, value interface{}, fields ...string) *yaml.Manifest {
	_ = unstructured.SetNestedField(manifest.Raw.Object, value, fields...)
	return manifest
}

func loadRealDeploymentManifest() *yaml.Manifest {
	manifest, _ := yaml.ParseYAML(`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    artifact.spinnaker.io/location: service-prod
    artifact.spinnaker.io/name: testapp
    artifact.spinnaker.io/type: kubernetes/deployment
    artifact.spinnaker.io/version: ""
    deployment.kubernetes.io/revision: "3"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"something"}
    moniker.spinnaker.io/application: testapp
    moniker.spinnaker.io/cluster: deployment testapp
  creationTimestamp: "2021-08-11T23:34:34Z"
  generation: 3
  labels:
    app.kubernetes.io/managed-by: spinnaker
    app.kubernetes.io/name: testapp
  name: testapp
  namespace: service-prod
  resourceVersion: "5283884480"
  uid: cd198383-15da-4ec5-88d0-926e9bda484f
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: testapp
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      annotations:
        artifact.spinnaker.io/location: service-prod
        artifact.spinnaker.io/name: testapp
        artifact.spinnaker.io/type: kubernetes/deployment
        artifact.spinnaker.io/version: ""
        kubectl.terraform.test/envoy: "true"
        kubectl.terraform.test/telegraf: statsd
        moniker.spinnaker.io/application: testapp
        moniker.spinnaker.io/cluster: deployment testapp
      creationTimestamp: null
      labels:
        app.kubernetes.io/managed-by: spinnaker
        app.kubernetes.io/name: testapp
        app.kubernetes.io/version: 1.0.918
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubectl.terraform.test/instance-class
                operator: In
                values:
                - cpu
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - testapp
              topologyKey: topology.kubernetes.io/zone
            weight: 25
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - testapp
              topologyKey: kubernetes.io/hostname
            weight: 100
      containers:
      - envFrom:
        - configMapRef:
            name: testapp-config-v001
        image: my-registry/application/testapp:1.0.918
        imagePullPolicy: Always
        name: application
        resources:
          limits:
            cpu: "1"
            memory: 1Gi
          requests:
            cpu: 50m
            memory: 256Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /share
          name: shared-data
      dnsPolicy: ClusterFirst
      priorityClassName: spinnaker-managed-deployment
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext:
        fsGroup: 1100
        runAsUser: 1100
      serviceAccount: testapp
      serviceAccountName: testapp
      terminationGracePeriodSeconds: 30
      volumes:
      - name: cert
        secret:
          defaultMode: 420
          secretName: testapp-cert
      - emptyDir: {}
        name: shared-data
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2021-08-11T23:34:34Z"
    lastUpdateTime: "2021-08-18T20:05:00Z"
    message: ReplicaSet "testapp-59d5f58d75" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  - lastTransitionTime: "2021-09-30T13:10:54Z"
    lastUpdateTime: "2021-09-30T13:10:54Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  observedGeneration: 3
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
`)

	return manifest
}
