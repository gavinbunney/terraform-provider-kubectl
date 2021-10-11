package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nmetadata:\n  name: name-here-crontabs.stable.example.com\nspec:\n  group: stable.example.com\n  conversion:\n    strategy: None\n  scope: Namespaced\n  names:\n    plural: name-here-crontabs\n    singular: crontab\n    kind: CronTab\n    shortNames:\n      - ct\n  version: v1\n  versions:\n    - name: v1\n      served: true\n      storage: true"),
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
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		the_kind           = "MyAwesomeCRD"
		crd_kind           = "MyAwesomeCRD"
		name               = "Malcolm"
		namespaces         = "dev"
		hyperscale_enabled = "false"
	}
}
`, path+"/*.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "8"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "kind: MyAwesomeCRD\nMyYaml: Hello, Malcolm!"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", "---\napiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.2", "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nmetadata:\n  name: name-here-crontabs.stable.example.com\nspec:\n  group: stable.example.com\n  conversion:\n    strategy: None\n  scope: Namespaced\n  names:\n    plural: name-here-crontabs\n    singular: crontab\n    kind: MyAwesomeCRD\n    shortNames:\n      - ct\n  version: v1\n  versions:\n    - name: v1\n      served: true\n      storage: true"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.3", "---\napiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.4", "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nmetadata:\n  name: name-here-crontabs.stable.example.com\nspec:\n  group: stable.example.com\n  conversion:\n    strategy: None\n  scope: Namespaced\n  names:\n    plural: name-here-crontabs\n    singular: crontab\n    kind: CronTab\n    shortNames:\n      - ct\n  version: v1\n  versions:\n    - name: v1\n      served: true\n      storage: true"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.5", "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: dev\n  labels:\n    name: dev"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.6", "apiVersion: \"stable.example.com/v1\"\nkind: MyAwesomeCRD\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.7", "apiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
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

func TestAccKubectlDataSourcePathDocuments_single_templated(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		the_kind = "MyAwesomeCRD"
	}
}
`, path+"/single-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "apiVersion: \"stable.example.com/v1\"\nkind: MyAwesomeCRD\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
				),
			},
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	sensitive_vars = {
		the_kind = "MyAwesomeCRD"
	}
}
`, path+"/single-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "apiVersion: \"stable.example.com/v1\"\nkind: MyAwesomeCRD\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
				),
			},
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		the_kind = "DefaultValue"
	}
	sensitive_vars = {
		the_kind = "MyAwesomeCRD"
	}
}
`, path+"/single-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "apiVersion: \"stable.example.com/v1\"\nkind: MyAwesomeCRD\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_multiple_templated(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		crd_kind = "MyAwesomeCRD"
	}
}
`, path+"/multiple-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "2"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "---\napiVersion: \"stable.example.com/v1\"\nkind: CronTab\nmetadata:\n  name: name-here-crd\nspec:\n  cronSpec: \"* * * * /5\"\n  image: my-awesome-cron-image"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nmetadata:\n  name: name-here-crontabs.stable.example.com\nspec:\n  group: stable.example.com\n  conversion:\n    strategy: None\n  scope: Namespaced\n  names:\n    plural: name-here-crontabs\n    singular: crontab\n    kind: MyAwesomeCRD\n    shortNames:\n      - ct\n  version: v1\n  versions:\n    - name: v1\n      served: true\n      storage: true"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_directives(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		name = "Malcolm"
	}
}
`, path+"/directives-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "kind: MyAwesomeCRD\nMyYaml: Hello, Malcolm!"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_directives_without_var(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		name = ""
	}
}
`, path+"/directives-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "kind: MyAwesomeCRD\nMyYaml: Hello, unnamed!"),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_namespaces_looped(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		namespaces = "dev,test,prod"
		hyperscale_enabled = false
	}
}
`, path+"/namespaces.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "3"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", `apiVersion: v1
kind: Namespace
metadata:
  name: dev
  labels:
    name: dev`),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", `apiVersion: v1
kind: Namespace
metadata:
  name: test
  labels:
    name: test`),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.2", `apiVersion: v1
kind: Namespace
metadata:
  name: prod
  labels:
    name: prod`),
				),
			},
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern = "%s"
	vars = {
		namespaces = "dev,test,prod"
		hyperscale_enabled = true
	}
}
`, path+"/namespaces.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "3"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", `apiVersion: v1
kind: Namespace
metadata:
  name: dev
  labels:
    name: dev
    hyperscale: enabled`),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.1", `apiVersion: v1
kind: Namespace
metadata:
  name: test
  labels:
    name: test
    hyperscale: enabled`),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.2", `apiVersion: v1
kind: Namespace
metadata:
  name: prod
  labels:
    name: prod
    hyperscale: enabled`),
				),
			},
		},
	})
}

func TestAccKubectlDataSourcePathDocuments_disable_template(t *testing.T) {
	path := "../_examples/manifests"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubectl_path_documents" "test" {
	pattern          = "%s"
    disable_template = true
}
`, path+"/directives-templated.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.#", "1"),
					resource.TestCheckResourceAttr("data.kubectl_path_documents.test", "documents.0", "kind: MyAwesomeCRD\nMyYaml: Hello, %{ if name != \"\" }${name}%{ else }unnamed%{ endif }!"),
				),
			},
		},
	})
}
