package main

import (
	kubernetes "github.com/gavinbunney/terraform-provider-kubectl/kubernetes"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider})
}
