package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccKubectlDataSourcePathDocuments_single(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePathDocumentsConfig_basic(path + "/single.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "apiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_multiple(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePathDocumentsConfig_basic(path + "/multiple.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "2"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "---\napiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", "apiVersion: apiextensions.k8s.io/v1beta1\nkind: CustomResourceDefinition\nmetadata:\n  name: name-here-crontabs.stable.example.com\nspec:\n  group: stable.example.com\n  conversion:\n    strategy: None\n  scope: Namespaced\n  names:\n    plural: name-here-crontabs\n    singular: crontab\n    kind: CronTab\n    shortNames:\n      - ct\n  version: v1\n  versions:\n    - name: v1\n      served: true\n      storage: true"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_multiple_files(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePathDocumentsConfig_basic(path + "/*.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "3"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "---\napiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", "apiVersion: apiextensions.k8s.io/v1beta1\nkind: CustomResourceDefinition\nmetadata:\n  name: name-here-crontabs.stable.example.com\nspec:\n  group: stable.example.com\n  conversion:\n    strategy: None\n  scope: Namespaced\n  names:\n    plural: name-here-crontabs\n    singular: crontab\n    kind: CronTab\n    shortNames:\n      - ct\n  version: v1\n  versions:\n    - name: v1\n      served: true\n      storage: true"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.2", "apiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePathDocumentsConfig_basic(path string) string {
	return fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
}
`, path)
}
