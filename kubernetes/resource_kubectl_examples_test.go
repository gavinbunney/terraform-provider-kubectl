package kubernetes

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func visit(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if filepath.Ext(path) == ".tf" {
			*files = append(*files, path)
		}
		return nil
	}
}

func TestAcckubectlYaml(t *testing.T) {
	var files []string
	root := "../_examples"
	err := filepath.Walk(root, visit(&files))
	if err != nil {
		panic(err)
	}

	for _, path := range files {
		t.Run("File: "+path, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

			resource.Test(t, resource.TestCase{
				PreCheck:      func() {},
				IDRefreshName: "kubectl_manifest.test",
				Providers:     testAccProviders,
				CheckDestroy:  testAccCheckkubectlDestroy,
				Steps: []resource.TestStep{
					{
						Config: testkubectlYamlLoadTfExample(path, name),
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckkubectlExists,
							resource.TestCheckResourceAttrSet("kubectl_manifest.test", "yaml_incluster"),
							resource.TestCheckResourceAttrSet("kubectl_manifest.test", "live_manifest_incluster"),
						),
					},
				},
			})
		})
	}
}

func testkubectlYamlLoadTfExample(path, name string) string {

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(dat), "name-here", name, -1)
}
