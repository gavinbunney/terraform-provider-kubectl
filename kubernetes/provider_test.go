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
	return testAccCheckK8srawStatus(s, false)
}

func testAccCheckK8srawExists(s *terraform.State) error {
	return testAccCheckK8srawStatus(s, true)
}

func testAccCheckK8srawStatus(s *terraform.State, shouldExist bool) error {
	conn, _ := testAccProvider.Meta().(KubeProvider)()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "k8sraw_yaml" {
			continue
		}

		content, err := conn.RESTClient().Get().AbsPath(rs.Primary.ID).DoRaw()
		fmt.Println(string(content))
		if (errors.IsNotFound(err) || errors.IsGone(err)) && shouldExist {
			return fmt.Errorf("Failed to find resource, likely a failure to create occured: %+v %v", err, string(content))
		}

	}

	return nil
}
