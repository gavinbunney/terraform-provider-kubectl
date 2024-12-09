package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

var kustTargetUrl = "https://github.com/kubernetes-sigs/kustomize/examples/multibases?ref=v1.0.6"

func TestAccKubectlDataSourceKustomizeDocuments_url(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: kubectlKustomizeDocumentsConfig(kustTargetUrl),
				Check:  resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.#", "3"),
			},
		},
	})
}

func TestAccKubectlDataSourceKustomizeDocuments_localDir(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: kubectlKustomizeDocumentsConfig("../test/data/kustomize/helloWorld"),
				Check:  resource.TestCheckResourceAttr("data.kubectl_kustomize_documents.test", "documents.#", "3"),
			},
		},
	})
}

func kubectlKustomizeDocumentsConfig(target string) string {
	return fmt.Sprintf(`
data "kubectl_kustomize_documents" "test" {
	target = "%s"
}
	`, target)
}
