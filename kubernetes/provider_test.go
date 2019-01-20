package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	upstream "github.com/terraform-providers/terraform-provider-kubernetes"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"k8sraw":     testAccProvider,
		"kubernetes": upstream.Provider(),
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccCheckK8srawDestroy(s *terraform.State) error {
	return testAccCheckK8srawStatus(s, 404)
}

func testAccCheckK8srawExists(s *terraform.State) error {
	return testAccCheckK8srawStatus(s, 200)
}

func testAccCheckK8srawStatus(s *terraform.State, expectedCode int) error {
	conn, _ := testAccProvider.Meta().(KubeProvider)()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "k8sraw_yaml" {
			continue
		}

		result := conn.RESTClient().Get().AbsPath(rs.Primary.ID).Do()
		var statusCode int
		result.StatusCode(&statusCode)
		// Did we get the status code we expected?
		if statusCode == expectedCode {
			continue
		}

		// Another error occured
		response, err := result.Get()
		if err != nil {
			return fmt.Errorf("Failed to get 404 for resource, likely a failure to delete occured: %+v", err)
		}
		return fmt.Errorf("Response: %+v", response)

	}

	return nil
}
