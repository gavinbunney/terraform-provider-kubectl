package main

import (
	kubernetes "github.com/gavinbunney/terraform-provider-kubernetes-yaml/kubernetes"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider})
}
