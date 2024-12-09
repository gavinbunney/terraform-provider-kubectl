package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccKubectlDataSourceFilenameList_basic(t *testing.T) {
	path := "../test/e2e/crds"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceFilenameListConfig_basic(path + "/*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.0", path+"/couchbase.tf"),
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
