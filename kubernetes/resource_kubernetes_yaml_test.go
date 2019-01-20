package kubernetes

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccK8srawYamlService_basic(t *testing.T) {
	// var conf api.NetworkPolicy
	name := fmt.Sprintf("tf-acc-test-service-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() {},
		IDRefreshName: "k8sraw_yaml.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckK8srawDestroy,
		Steps: []resource.TestStep{
			{
				Config: testk8sRawYamlNetworking(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckK8srawExists,
					resource.TestCheckResourceAttrSet("k8sraw_yaml.test", "yaml_incluster"),
					resource.TestCheckResourceAttrSet("k8sraw_yaml.test", "live_yaml_incluster"),
				),
			},
		},
	})
}

func testk8sRawYamlNetworking(name string) string {

	dat, err := ioutil.ReadFile("./../_examples/services/basic_service.tf")
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(dat), "__NAME_HERE__", name, -1)
}
