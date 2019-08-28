package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccKubectlDataSourceFileDocuments_single(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceFileDocumentsConfig_basic(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.0", "kind: Service1"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourceFileDocuments_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceFileDocumentsConfig_basic(2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.#", "2"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.0", "kind: Service1"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.1", "kind: Service2"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceFileDocumentsConfig_basic(docs int) string {
	var content = ""
	for i := 1; i <= docs; i++ {
		content += fmt.Sprintf("\nkind: Service%v\n---", i)
	}

	return fmt.Sprintf(`
data "kubectl_file_documents" "test" {
	content = <<YAML
%s
YAML
}
`, content)
}
