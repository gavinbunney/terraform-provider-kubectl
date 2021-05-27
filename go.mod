module github.com/gavinbunney/terraform-provider-kubectl

go 1.16

require (
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/hashicorp/go-plugin v1.4.2
	github.com/hashicorp/hcl/v2 v2.10.1
	github.com/hashicorp/terraform v0.12.29
	github.com/hashicorp/terraform-plugin-go v0.3.1
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.7.0
	github.com/icza/dyno v0.0.0-20200205103839-49cb13720835
	github.com/mitchellh/go-homedir v1.1.0
	github.com/stretchr/testify v1.7.0
	github.com/zclconf/go-cty v1.8.4
	github.com/zclconf/go-cty-yaml v1.0.2
	google.golang.org/grpc v1.39.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/cli-runtime v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/kube-aggregator v0.21.3
	k8s.io/kubectl v0.21.3
	sigs.k8s.io/kustomize/api v0.8.10
	sigs.k8s.io/yaml v1.2.0
)
