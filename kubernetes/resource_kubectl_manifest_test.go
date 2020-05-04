package kubernetes

import (
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/stretchr/testify/assert"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestKubectlManifest_RetryOnFailure(t *testing.T) {
	_ = os.Setenv("KUBECTL_PROVIDER_APPLY_RETRY_COUNT", "3")

	config := `
resource "kubectl_manifest" "test" {
	yaml_body = <<YAML
apiVersion: v1
kind: Ingress
YAML
}
	`

	start := time.Now()
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

	assert.True(t, time.Since(start) > (3*time.Second))
}
