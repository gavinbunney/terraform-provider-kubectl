package kubernetes

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	// "github.com/hashicorp/terraform/helper/schema"
	// "github.com/hashicorp/terraform/terraform"
	// api "k8s.io/api/networking/v1"
	// meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// kubernetes "k8s.io/client-go/kubernetes"
)

func TestAccKubernetesNetworkPolicy_basic(t *testing.T) {
	// var conf api.NetworkPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() {},
		IDRefreshName: "kubernetes_network_policy.test",
		Providers:     testAccProviders,
		// CheckDestroy:  testAccCheckKubernetesNetworkPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testk8sRawYamlNetworking(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "1"),
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
	return string(dat)
}
