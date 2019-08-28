package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccKubectlDataSourceFilenameList_basic(t *testing.T) {
	path := "../_examples/crds"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceFilenameListConfig_basic(path + "/*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.#", "2"),
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.0", path+"/basic_crd.tf"),
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.1", path+"/couchbase.tf"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceFilenameListConfig_basic(path string) string {
	return fmt.Sprintf(`
data "kubectl_filename_list" "test" {
	pattern = "%s"
}
`, path)
}
