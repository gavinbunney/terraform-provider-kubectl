package main

import (
	"github.com/hashicorp/terraform/plugin"
	kubernetes "github.com/lawrencegripper/terraform-provider-kubernetes-yaml/kubernetesyaml"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider})
}
