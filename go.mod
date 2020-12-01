module github.com/gavinbunney/terraform-provider-kubectl

go 1.14

require (
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/hashicorp/go-plugin v1.3.0
	github.com/hashicorp/hcl/v2 v2.6.0
	github.com/hashicorp/terraform v0.12.29
	github.com/hashicorp/terraform-plugin-go v0.1.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.3.0
	github.com/icza/dyno v0.0.0-20180601094105-0c96289f9585
	github.com/mitchellh/go-homedir v1.1.0
	github.com/stretchr/testify v1.5.1
	github.com/zclconf/go-cty v1.2.1
	github.com/zclconf/go-cty-yaml v1.0.1
	google.golang.org/grpc v1.32.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.18.12
	k8s.io/cli-runtime v0.18.12
	k8s.io/client-go v0.18.12
	k8s.io/kube-aggregator v0.18.12
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect
	k8s.io/kubectl v0.18.12
	sigs.k8s.io/yaml v1.2.0
)

//replace github.com/Azure/go-autorest v10.15.4+incompatible => github.com/Azure/go-autorest v13.0.0+incompatible
