package kubernetes

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
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

func TestAccK8srawYaml(t *testing.T) {
	var files []string
	root := "../_examples"
	err := filepath.Walk(root, visit(&files))
	if err != nil {
		panic(err)
	}

	for _, path := range files {
		t.Run("File: "+path, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test-service-%s", acctest.RandString(10))

			resource.Test(t, resource.TestCase{
				PreCheck:      func() {},
				IDRefreshName: "k8sraw_yaml.test",
				Providers:     testAccProviders,
				CheckDestroy:  testAccCheckK8srawDestroy,
				Steps: []resource.TestStep{
					{
						Config: testk8sRawYamlLoadTfExample(path, name),
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckK8srawExists,
							resource.TestCheckResourceAttrSet("k8sraw_yaml.test", "yaml_incluster"),
							resource.TestCheckResourceAttrSet("k8sraw_yaml.test", "live_yaml_incluster"),
						),
					},
				},
			})
		})
	}
}

func testk8sRawYamlLoadTfExample(path, name string) string {

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(dat), "name-here", name, -1)
}
