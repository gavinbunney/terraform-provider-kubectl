package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	upstream "github.com/terraform-providers/terraform-provider-kubernetes/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"kubectl":     testAccProvider,
		"kubernetes": upstream.Provider(),
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccCheckkubectlDestroy(s *terraform.State) error {
	return testAccCheckkubectlStatus(s, false)
}

func testAccCheckkubectlExists(s *terraform.State) error {
	return testAccCheckkubectlStatus(s, true)
}

func testAccCheckkubectlStatus(s *terraform.State, shouldExist bool) error {
	conn, _ := testAccProvider.Meta().(KubeProvider)()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubectl_manifest" {
			continue
		}

		content, err := conn.RESTClient().Get().AbsPath(rs.Primary.ID).DoRaw()
		if (errors.IsNotFound(err) || errors.IsGone(err)) && shouldExist {
			return fmt.Errorf("Failed to find resource, likely a failure to create occured: %+v %v", err, string(content))
		}

	}

	return nil
}
