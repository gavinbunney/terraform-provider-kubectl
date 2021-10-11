package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests.%", "1"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests./apis/service1s", "kind: Service1\n"),
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
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests.%", "2"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests./apis/service1s", "kind: Service1\n"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests./apis/service2s", "kind: Service2\n"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourceFileDocuments_basicMultipleEmpty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "kubectl_file_documents" "test" {
	content = <<YAML
kind: Service1
---
# just a comment
---
kind: Service2
---
YAML
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.#", "2"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.0", "kind: Service1"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "documents.1", "kind: Service2"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests.%", "2"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests./apis/service1s", "kind: Service1\n"),
					resource.TestCheckResourceAttr("data.kubectl_file_documents.test", "manifests./apis/service2s", "kind: Service2\n"),
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
