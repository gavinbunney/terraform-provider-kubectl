package main

import (
	kubernetes "github.com/gavinbunney/terraform-provider-kubectl/kubernetes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider})
}
