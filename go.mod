module github.com/gavinbunney/terraform-provider-kubectl

go 1.16

require (
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/hashicorp/go-plugin v1.4.0
	github.com/hashicorp/hcl/v2 v2.10.0
	github.com/hashicorp/terraform v0.12.29
	github.com/hashicorp/terraform-plugin-go v0.3.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.6.1
	github.com/icza/dyno v0.0.0-20200205103839-49cb13720835
	github.com/mitchellh/go-homedir v1.1.0
	github.com/stretchr/testify v1.7.0
	github.com/zclconf/go-cty v1.8.2
	github.com/zclconf/go-cty-yaml v1.0.2
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/cli-runtime v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/kube-aggregator v0.20.7
	k8s.io/kubectl v0.20.7
	sigs.k8s.io/yaml v1.2.0
)

//replace github.com/Azure/go-autorest v10.15.4+incompatible => github.com/Azure/go-autorest v13.0.0+incompatible
