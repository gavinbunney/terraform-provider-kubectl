module github.com/gavinbunney/terraform-provider-kubernetes-yaml

go 1.12

require (
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/google/btree v1.0.0 // indirect
	github.com/hashicorp/terraform v0.12.0
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/icza/dyno v0.0.0-20180601094105-0c96289f9585
	github.com/mitchellh/go-homedir v1.0.0
	github.com/stretchr/testify v1.3.0
	github.com/terraform-providers/terraform-provider-kubernetes v1.5.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/apimachinery v0.0.0-20190119020841-d41becfba9ee
	k8s.io/client-go v10.0.0+incompatible
)
