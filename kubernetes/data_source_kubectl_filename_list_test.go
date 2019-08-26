package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

//func TestAcckubectlFilenameList(t *testing.T) {
//
//	path := "../_examples/crds"
//	t.Run("Path: " + path, func(t *testing.T) {
//		name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
//
//		resource.Test(t, resource.TestCase{
//			PreCheck:      func() {},
//			IDRefreshName: "kubectl_manifest.test",
//			Providers:     testAccProviders,
//			CheckDestroy:  testAccCheckkubectlDestroy,
//			Steps: []resource.TestStep{
//				{
//					Config: testkubectlYamlLoadTfExample(path, name),
//					Check: resource.ComposeAggregateTestCheckFunc(
//						testAccCheckkubectlExists,
//						resource.TestCheckResourceAttrSet("kubectl_manifest.test", "yaml_incluster"),
//						resource.TestCheckResourceAttrSet("kubectl_manifest.test", "live_manifest_incluster"),
//					),
//				},
//			},
//		})
//	})
//}

func TestAccKubectlDataSourceFilenameList_basic(t *testing.T) {
	//name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	path := "../_examples/crds"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(path + "/*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.#", "2"),
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.0", path+"/basic_crd.tf"),
					resource.TestCheckResourceAttr("data.kubectl_filename_list.test", "matches.1", path+"/couchbase.tf"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceStorageClassConfig_basic(path string) string {
	return fmt.Sprintf(`
data "kubectl_filename_list" "test" {
	pattern = "%s"
}
`, path)
}
