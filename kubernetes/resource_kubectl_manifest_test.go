package kubernetes

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/alekc/terraform-provider-kubectl/yaml"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestKubectlManifest_RetryOnFailure(t *testing.T) {
	_ = os.Setenv("KUBECTL_PROVIDER_APPLY_RETRY_COUNT", "5")

	config := `
resource "kubectl_manifest" "test" {
	yaml_body = <<YAML
apiVersion: v1
kind: Ingress
YAML
}
	`

	expectedError, _ := regexp.Compile(".*failed to create kubernetes.*")
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				ExpectError: expectedError,
				Config:      config,
			},
		},
	})
}

func TestAccKubectl(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
  wait = true
	yaml_body = <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    readinessProbe:
      httpGet:
        path: "/"
        port: 80
      initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_Wait(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait = true
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_WaitForeground(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait = true
  delete_cascade = "Foreground"
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_WaitBackground(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait = true
  delete_cascade = "Background"
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_WaitForRolloutDeployment(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
  wait_for_rollout = true
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_WaitForRolloutDaemonSet(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
  wait_for_rollout = true
	yaml_body = <<YAML
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_WaitForRolloutStatefulSet(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
  wait_for_rollout = true
	yaml_body = <<YAML
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  updateStrategy:
    type: RollingUpdate
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
		},
	})
}

func TestAccKubectl_RequireWaitForFieldOrCondition(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait_for { }
	yaml_body = <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    readinessProbe:
      httpGet:
        path: "/"
        port: 80
      initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	expectedError, _ := regexp.Compile(".*at least one of `field` or `condition` must be provided in `wait_for` block.*")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: expectedError,
				//todo: improve checking
			},
		},
	})
}
func TestAccKubectl_WaitForNegativeField(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test_wait_for" {
  timeouts {
    create = "10s"
  }
  yaml_body = <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: test-wait-for
EOF

  wait_for {
    field {
      key = "status.phase"
      value = "Activez"
    }
  }
}` //start := time.Now()
	// atm the actual error is being hidden by the wait context being deleted. Fix this at some point
	//errorRegex, _ := regexp.Compile(".*failed to wait for resource*")
	errorRegex, _ := regexp.Compile(".*Wait returned an error*")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: errorRegex,
			},
		},
	})
	log.Println(config)
}

func TestAccKubectl_WaitForNegativeCondition(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	timeouts {
		create = "20s"
	}

	wait_for {
		condition {
			type = "ContainersReady"
			status = "Never"
		}
	}
	yaml_body = <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: busybox-sleep
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["sleep", "30"]
YAML
}` //start := time.Now()
	// atm the actual error is being hidden by the wait context being deleted. Fix this at some point
	//errorRegex, _ := regexp.Compile(".*failed to wait for resource*")
	errorRegex, _ := regexp.Compile(".*Wait returned an error*")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: errorRegex,
			},
		},
	})
	log.Println(config)
}

func TestAccKubectl_WaitForNS(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test_wait_for" {
  timeouts {
    create = "200s"
  }
  yaml_body = <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: test-wait-for
EOF

  wait_for {
    field {
      key = "status.phase"
      value = "Active"
    }
  }
}` //start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				//todo: improve checking
			},
		},
	})
	log.Println(config)
}

func TestAccKubectl_WaitForField(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait_for {
		field {
			key = "status.containerStatuses.[0].ready"
			value = "true"
		}
		field {
			key = "status.phase"
			value = "Running"
		}
		field {
			key = "status.podIP"
			value = "^(\\d+(\\.|$)){4}"
			value_type = "regex"
		}
	}
	yaml_body = <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    readinessProbe:
      httpGet:
        path: "/"
        port: 80
      initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				//todo: improve checking
			},
		},
	})
}

func TestAccKubectl_WaitForConditions(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait_for {
		condition {
			type = "ContainersReady"
			status = "True"
		}
		condition {
			type = "Ready"
			status = "True"
		}
	}
	yaml_body = <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    readinessProbe:
      httpGet:
        path: "/"
        port: 80
      initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				//todo: improve checking
			},
		},
	})
}

func TestAccKubectl_WaitForFieldAndCondition(t *testing.T) {
	//language=hcl
	config := `
resource "kubectl_manifest" "test" {
	wait_for {
		condition {
			type = "ContainersReady"
			status = "True"
		}
		condition {
			type = "Ready"
			status = "True"
		}
		field {
			key = "status.containerStatuses.[0].ready"
			value = "true"
		}
		field {
			key = "status.phase"
			value = "Running"
		}
		field {
			key = "status.podIP"
			value = "^(\\d+(\\.|$)){4}"
			value_type = "regex"
		}
	}
	yaml_body = <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    readinessProbe:
      httpGet:
        path: "/"
        port: 80
      initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				//todo: improve checking
			},
		},
	})
}

func TestAccKubectl_WaitForConditionUpdate(t *testing.T) {
	//language=hcl
	createConfig := `
resource "kubectl_manifest" "test" {
	wait_for_rollout = true
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.27.0
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	updateConfig := `
resource "kubectl_manifest" "test" {
  wait_for_rollout = false
	wait_for {
		condition {
			type = "Available"
			status = "True"
		}
	}
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.27.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
			},
			{
				Config: updateConfig,
			},
		},
	})
}

func TestAccKubectl_WaitForFieldUpdate(t *testing.T) {
	//language=hcl
	createConfig := `
resource "kubectl_manifest" "test" {
	wait_for_rollout = true
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.27.0
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	updateConfig := `
resource "kubectl_manifest" "test" {
  wait_for_rollout = false
	wait_for {
		field {
			key = "status.observedGeneration"
			value = "2"
		}
	}
	yaml_body = <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.27.2
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: "/"
              port: 80
            initialDelaySeconds: 10
YAML
}
`

	//start := time.Now()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
			},
			{
				Config: updateConfig,
			},
		},
	})
}

//func TestAccKubect_Debug(t *testing.T) {
//	//language=hcl
//	config := `
//resource "kubectl_manifest" "test" {
//	yaml_body = <<YAML
//apiVersion: v1
//kind: Secret
//metadata:
//  name: test-secret
//stringData:
//  var: "${formatdate("YYYYMMDDhhmmss", timestamp())}"
//YAML
//}
//`
//
//	//start := time.Now()
//	resource.Test(t, resource.TestCase{
//		PreCheck:     func() { testAccPreCheck(t) },
//		Providers:    testAccProviders,
//		CheckDestroy: testAccCheckkubectlDestroy,
//		Steps: []resource.TestStep{
//			{
//				Config: config,
//				//todo: improve checking
//			},
//		},
//	})
//}

func TestAccInconsistentPlanning(t *testing.T) {
	//See https://github.com/alekc/terraform-provider-kubectl/pull/46
	config := `
resource "kubectl_manifest" "secret" {
  yaml_body = <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
stringData:
  var: "${formatdate("YYYYMMDDhhmmss", timestamp())}"
EOF
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,

		Steps: []resource.TestStep{
			{
				Config:             config,
				ExpectNonEmptyPlan: true, // yaml_incluster is going to be constantly different
			},
			{
				// used to crash out on the second run
				Config:             config,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccKubectlUnknownNamespace(t *testing.T) {
	config := `
resource "kubectl_manifest" "test" {
	yaml_body = <<EOT
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
  namespace: this-doesnt-exist
spec:
  ingressClassName: "nginx"
  rules:
  - host: "*.example.com"
    http:
      paths:
      - path: "/testpath"
        pathType: "Prefix"
        backend:
          service:
            name: test
            port:
              number: 80
	EOT
		}
`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("\"this-doesnt-exist\" not found"),
			},
		},
	})
}

func TestAccKubectlOverrideNamespace(t *testing.T) {

	namespace := "dev-" + acctest.RandString(10)
	yaml_body := `
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  namespace: prod
type: Opaque
data:
`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "ns" {
	yaml_body = <<EOT
apiVersion: v1
kind: Namespace
metadata:
  name: %s
    EOT
}

resource "kubectl_manifest" "test" {
	depends_on = [kubectl_manifest.ns]
    override_namespace = "%s"
	yaml_body = <<EOT
%s
	EOT
		}
`, namespace, namespace, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "override_namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", fmt.Sprintf(`apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
  namespace: %s
type: Opaque
`, namespace)),
				),
			},
		},
	})
}

func TestAccKubectlSetNamespace(t *testing.T) {

	namespace := "dev-" + acctest.RandString(10)
	yaml_body := `
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
type: Opaque
data:
`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "ns" {
	yaml_body = <<EOT
apiVersion: v1
kind: Namespace
metadata:
  name: %s
    EOT
}

resource "kubectl_manifest" "test" {
    depends_on = [kubectl_manifest.ns]
    override_namespace = "%s"
	yaml_body = <<EOT
%s
	EOT
		}
`, namespace, namespace, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "id", "/api/v1/namespaces/"+namespace+"/secrets/mysecret"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "override_namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", fmt.Sprintf(`apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
  namespace: %s
type: Opaque
`, namespace)),
				),
			},
		},
	})
}

func TestAccKubectlSetNamespace_nonnamespaced_resource(t *testing.T) {

	namespace := "dev-" + acctest.RandString(10)
	yaml_body := fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mysuperrole-%s
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "watch", "list"]
`, namespace)

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    override_namespace = "%s"
	yaml_body = <<EOT
%s
	EOT
		}
`, namespace, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "override_namespace", namespace),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mysuperrole-%s
  namespace: %s
rules:
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - get
  - watch
  - list
`, namespace, namespace)),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_secret(t *testing.T) {

	yaml_body := `
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  namespace: default
type: Opaque
data:
  USER_NAME: YWRtaW4=
  PASSWORD: MWYyZDFlMmU2N2Rm
`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "namespace", "default"),
					resource.TestCheckNoResourceAttr("kubectl_manifest.test", "override_namespace"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: v1
data: (sensitive value)
kind: Secret
metadata:
  name: mysecret
  namespace: default
type: Opaque
`),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_slice(t *testing.T) {

	yaml_body := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: "nginx"
  rules:
  - host: "*.example.com"
    http:
      paths:
      - path: "/testpath"
        pathType: "Prefix"
        backend:
          service:
            name: test
            port:
              number: 80`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    sensitive_fields = [
      "spec.rules",
    ]

	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: nginx
  rules: (sensitive value)
`),
				),
			},
		},
	})
}

func TestAccKubectlSensitiveFields_unknown_field(t *testing.T) {

	yaml_body := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: "nginx"
  rules:
  - host: "*.example.com"
    http:
      paths:
      - path: "/testpath"
        pathType: "Prefix"
        backend:
          service:
            name: test
            port:
              number: 80`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    sensitive_fields = [
      "spec.field.missing",
    ]

	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body_parsed", `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: nginx
  rules:
  - host: '*.example.com'
    http:
      paths:
      - backend:
          service:
            name: test
            port:
              number: 80
        path: /testpath
        pathType: Prefix
`),
				),
			},
		},
	})
}

func TestAccKubectlWithoutValidation(t *testing.T) {

	yaml_body := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: name-here
spec:
  ingressClassName: "nginx"
  rules:
  - host: "*.example.com"
    http:
      paths:
      - path: "/testpath"
        pathType: "Prefix"
        backend:
          service:
            name: test
            port:
              number: 80`

	config := fmt.Sprintf(`
resource "kubectl_manifest" "test" {
    validate_schema = false

	yaml_body = <<EOT
%s
	EOT
		}
`, yaml_body)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckkubectlDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubectl_manifest.test", "yaml_body", yaml_body+"\n"),
					resource.TestCheckResourceAttr("kubectl_manifest.test", "validate_schema", "false"),
				),
			},
		},
	})
}

func TestGetLiveManifestFilteredForUserProvidedOnly(t *testing.T) {
	testCases := []struct {
		description         string
		expectedFields      string
		expectedFingerprint string
		userProvided        map[string]interface{}
		liveManifest        map[string]interface{}
		ignored             []string
		expectedDrift       bool
	}{
		{
			description: "Simple map with string value",
			userProvided: map[string]interface{}{
				"test1": "test2",
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
			},
			expectedFields:      "test1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "f9ed95ba61889f8e13cd1189f86c5901fb717ac60be6f4d7a65d17bf07326700",
			expectedDrift:       false,
		},
		{
			// Ensure skippable fields are skipped
			description: "Simple map with string value and Skippable fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"metadata": map[string]interface{}{
					"resourceVersion": "1245",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"metadata": map[string]interface{}{
					"resourceVersion": "1245",
				},
			},
			expectedFields:      "test1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "f9ed95ba61889f8e13cd1189f86c5901fb717ac60be6f4d7a65d17bf07326700",
			expectedDrift:       false,
		},
		{
			// Ensure ignored fields are skipped
			description: "Simple map with string value and ignored fields",
			userProvided: map[string]interface{}{
				"test1":      "test2",
				"ignoreThis": "1245",
			},
			liveManifest: map[string]interface{}{
				"test1":      "test2",
				"ignoreThis": "1245",
			},
			expectedFields:      "test1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "f9ed95ba61889f8e13cd1189f86c5901fb717ac60be6f4d7a65d17bf07326700",
			ignored:             []string{"ignoreThis"},
			expectedDrift:       false,
		},
		{
			// Ensure ignored sub fields are skipped
			description: "Simple map with string value and ignored fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "5432",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "1245",
				},
			},
			expectedFields:      "test1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "f9ed95ba61889f8e13cd1189f86c5901fb717ac60be6f4d7a65d17bf07326700",
			ignored:             []string{"ignore.this"},
			expectedDrift:       false,
		},
		{
			// Ensure ignored sub fields are skipped
			description: "Simple map with string ignore nested fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "5432",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "1245",
				},
			},
			expectedFields:      "test1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "f9ed95ba61889f8e13cd1189f86c5901fb717ac60be6f4d7a65d17bf07326700",
			ignored:             []string{"ignore"},
			expectedDrift:       false,
		},
		{
			// Ensure ignored sub fields are skipped
			description: "Simple map with string ignore highly nested fields",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]string{
					"this": "5432",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"ignore": map[string]interface{}{
					"this": "1245",
					"also": map[string]string{
						"these": "9876",
					},
				},
			},
			expectedFields:      "test1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "f9ed95ba61889f8e13cd1189f86c5901fb717ac60be6f4d7a65d17bf07326700",
			ignored:             []string{"ignore"},
			expectedDrift:       false,
		},
		{
			// Ensure nested `map[string]string` are supported
			description: "Map with nested map[string]string",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob": "bill",
				},
			},
			expectedFields:      "nest.bob=623210167553939c87ed8c5f2bfe0b3e0684e12c3a3dd2513613c4e67263b5a1\ntest1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "40bf1763c91a63191148522b44cf86d80fe76bb110f772eab7d9c3ecaef5259f",
			expectedDrift:       false,
		},
		{
			// Ensure nested `map[string]string` with different ordering are supported
			description: "Map with nested map[string]string with different ordering",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob1": "bill",
					"bob2": "bill",
					"bob3": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob2": "bill",
					"bob1": "bill",
					"bob3": "bill",
				},
			},
			expectedFields:      "nest.bob1=623210167553939c87ed8c5f2bfe0b3e0684e12c3a3dd2513613c4e67263b5a1\nnest.bob2=623210167553939c87ed8c5f2bfe0b3e0684e12c3a3dd2513613c4e67263b5a1\nnest.bob3=623210167553939c87ed8c5f2bfe0b3e0684e12c3a3dd2513613c4e67263b5a1\ntest1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "74762989e04b2a4c76111ef52e28571447ea2da66b5ec170d3a2aa8c87d36953",
			expectedDrift:       false,
		},
		{
			description: "Map with nested map[string]string with nested slice",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						"a",
						"b",
						"c",
					},
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						"c",
						"b",
						"a",
					},
				},
			},
			expectedFields:      "nest.bob1.#=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nnest.bob1.0=2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6\nnest.bob1.1=3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d\nnest.bob1.2=ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb\ntest1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "ac14f80b8067dfacbdfb9f6ad728ea09eadcf879bd9171500cc6adf02e84eb9d",
			expectedDrift:       true,
		},
		{
			description: "Map with nested map[string]string with nested array and nested map",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						map[string]string{
							"1": "1",
							"2": "2",
							"3": "3",
						},
						map[string]interface{}{
							"1": 1,
							"2": 2,
							"3": 3,
						},
					},
				},
			},
			liveManifest: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						map[string]string{
							"2": "2",
							"1": "1",
							"3": "3",
						},
						map[string]interface{}{
							"2": 2,
							"1": 1,
							"3": 3,
						},
					},
				},
			},
			expectedFields:      "nest.bob1.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nnest.bob1.0.1=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nnest.bob1.0.2=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nnest.bob1.0.3=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nnest.bob1.1.1=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nnest.bob1.1.2=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nnest.bob1.1.3=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\ntest1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "854345b04fc20f9bd5bfacb6179fd7ec38832f3927d0ff2de86cb96383cdb5f1",
			expectedDrift:       false,
		},
		{
			// Ensure ordering of the fields doesn't affect matching
			description: "Different Ordering",
			userProvided: map[string]interface{}{
				"ztest1": "test2",
				"afield": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
				"ztest1": "test2",
			},
			expectedFields:      "afield=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752\nztest1=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "72a914d79cbd09f028a179229f8a8695f1e733f89819c3f951e2045bbf1ab49f",
			expectedDrift:       false,
		},
		{
			// Ensure nested arrays are handled
			description: "Nested Array",
			userProvided: map[string]interface{}{
				"ztest1": []string{
					"1", "2",
				},
				"afield": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
				"ztest1": []string{
					"1", "2",
				},
			},
			expectedFields:      "afield=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752\nztest1.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nztest1.0=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nztest1.1=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35",
			expectedFingerprint: "60fec4d049ffe32998a8b98bafb4759d4c64d1e7d948f9ffd9af2ec3a8382892",
			expectedDrift:       false,
		},
		{
			// Ensure fields added to the `liveManifest` which aren't present in the `originl` are ignored
			description: "Ignore additional fields",
			userProvided: map[string]interface{}{
				"afield": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
				"ztest1": []string{
					"1", "2",
				},
			},
			expectedFields:      "afield=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "abd6da5468cf9cc65252371afe556be35b40903ec9e151a03945d20a41891423",
			expectedDrift:       false,
		},
		{
			// Ensure that fields present in the `userProvided` but missing in the `liveManifest` are skipped
			description: "Handle removed fields",
			userProvided: map[string]interface{}{
				"afield":   "test2",
				"igetlost": "test2",
			},
			liveManifest: map[string]interface{}{
				"afield": "test2",
			},
			expectedFields:      "afield=60303ae22b998861bce3b28f33eec1be758a213c86c93c076dbe9f558c11c752",
			expectedFingerprint: "abd6da5468cf9cc65252371afe556be35b40903ec9e151a03945d20a41891423",
			expectedDrift:       true,
		},
		{
			description: "Handle integers",
			userProvided: map[string]interface{}{
				"afield": 1,
			},
			liveManifest: map[string]interface{}{
				"afield": 1,
			},
			expectedFields:      "afield=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b",
			expectedFingerprint: "2d86f5deef362ae6a74fadb8cdb95df4386ee605a0c75c6226f0a810f49c9184",
			expectedDrift:       false,
		},
		{
			// Ensure that the updated value for `afield` on the `liveManifest` object is taken
			description: "Handle updated field. Expect liveManifest value to be shown",
			userProvided: map[string]interface{}{
				"afield": 1,
			},
			liveManifest: map[string]interface{}{
				"afield": 2,
			},
			expectedFields:      "afield=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35",
			expectedFingerprint: "cce738ca1a3ca9fa9069b6ed402b61f6021b9aa2826efb58bffb4c86948ae043",
			expectedDrift:       true,
		},
		{
			// Ensure that the updated value fo the `liveManifest` object is taken for the `willchange` field
			description: "Map with nested map[string]string with updated field",
			userProvided: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"willchange": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"nest": map[string]string{
					"willchange": "updatedbill",
				},
			},
			expectedFields:      "nest.willchange=aaab42be650931c48b99f6409a7c6990c62bcd55f827b0c4e76941d76490d8ca",
			expectedFingerprint: "42d86c0cee389ab367b2b8df0f2454d9677c15da425a6694b8208cc629e13c56",
			expectedDrift:       true,
		},
		{
			// Ensure that both fields are tracked in the output
			description: "Handle duplicate name fields in nested maps",
			userProvided: map[string]interface{}{
				"atest": "test",
				"nest": map[string]string{
					"atest": "bill",
				},
			},
			liveManifest: map[string]interface{}{
				"atest": "test",
				"nest": map[string]string{
					"atest": "bill",
				},
			},
			expectedFields:      "atest=9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\nnest.atest=623210167553939c87ed8c5f2bfe0b3e0684e12c3a3dd2513613c4e67263b5a1",
			expectedFingerprint: "1b8bf74179b9e91bf8c2ab43a3c5100f4261dba6f9409ef13359df55bb738c86",
			expectedDrift:       false,
		},
		{
			description: "Map with nested map[string]string with annotations",
			userProvided: map[string]interface{}{
				"atest": "test",
				"meta": map[string]interface{}{
					"annotations": map[string]string{
						"helm.sh/hook": "crd-install",
					},
				},
			},
			liveManifest: map[string]interface{}{
				"atest": "test",
				"meta": map[string]interface{}{
					"annotations": map[string]string{
						"helm.sh/hook": "crd-install",
					},
				},
			},
			expectedFields:      "atest=9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\nmeta.annotations.helm.sh/hook=7e53588c2c8cd5a8f0d7e59f4d22977f1fb7a1a5166dde9456795d80f683edb5",
			expectedFingerprint: "6f62fad1a4f0029840e291bee701c3f003951c10a5f9304028d44667f0db7ff1",
			expectedDrift:       false,
		},
		{
			description: "Map with empty annotations in user manifest",
			userProvided: map[string]interface{}{
				"atest": "test",
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{},
				},
			},
			liveManifest: map[string]interface{}{
				"atest": "test",
				"metadata": map[string]interface{}{
					"annotations": map[string]string{
						"kubectl.kubernetes.io/last-applied-configuration": "{\"should-be-ignored\"}",
					},
				},
			},
			expectedFields:      "atest=9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			expectedFingerprint: "522627f1e0ced24f72e61b1e93b2c9c82b7ea0516f444b57ab8b7697ef846b2a",
			expectedDrift:       false,
		},
		{
			description:         "Deployment manifest without changes",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        loadRealDeploymentManifest().Raw.Object,
			expectedFields:      "apiVersion=024614bbf9753e35bd0e7e47cf2f1d05243368e9b54ae6f53f8c80e152530aed\nkind=870a8ffd98f4f2bd5041ee4cebde82de4bdeb253fac88c5469a2f15f15614186\nmetadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nmetadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nmetadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nmetadata.annotations.deployment.kubernetes.io/revision=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nmetadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nmetadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nmetadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.namespace=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.progressDeadlineSeconds=284b7e6d788f363f910f7beb1910473e23ce9d6c871f1ce0f31f22a982d48ad4\nspec.replicas=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.revisionHistoryLimit=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nspec.selector.matchLabels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.strategy.rollingUpdate.maxSurge=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.rollingUpdate.maxUnavailable=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.type=8f434f8bcd785d91c7e6c0394b1a3ddd503ece214c8aea23b8446ee389456f96\nspec.template.metadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.template.metadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nspec.template.metadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nspec.template.metadata.annotations.kubectl.terraform.test/envoy=b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b\nspec.template.metadata.annotations.kubectl.terraform.test/telegraf=22dcc997cd3a5311709bb7fd75ea4e92d6d76daa87ff6a0e4b2eef9b85129736\nspec.template.metadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nspec.template.metadata.creationTimestamp=426b5dfece37e413f559015825ebc7c5ba251a13028e2fcd5ed36df57be00b6c\nspec.template.metadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nspec.template.metadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.labels.app.kubernetes.io/version=da7965ad0f9dddb433712a4ee3b014e42c70a4c1d560187dcf446cba4cdd4860\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=b2c21b8c6d5bc3c78a90d2e5974ba05916e2f777d1a85cf3d2832cdc9c487093\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=68ab84f7c6d0f5781585eb1b5289499fb29081b918f71ebddb5f72021c9ef9c5\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=d26ba3a6c4fcf2b72755b5a23fee0a5994a39b71bdc9db6ef6d083a22a001353\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=b7a56873cd771f2c446d369b649430b65a756ba278ff97ec81bb6f55b2e73569\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=93966ab96ee1d305f0346ca2acf3b7e16c004a07bf322671293db9f19c18b079\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=ad57366865126e55649ecb23ae1d48887544976efea46a48eb5d85a6eeb4d306\nspec.template.spec.containers.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.0.configMapRef.name=5fca260b630569e695d3345f5ed80bf350446b6e85014b8365c45fe9e139d20c\nspec.template.spec.containers.0.image=5f67805ae6c365cf598b82eb292e333e6f64f85869ccf89b0c0f6cae61c1ec5b\nspec.template.spec.containers.0.imagePullPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.containers.0.name=1fe289205936c3fdb61158223892c7a8bee6ff4dfa085ea1c094ce0294e32114\nspec.template.spec.containers.0.resources.limits.cpu=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.resources.limits.memory=861c482be953844b2a4cc7f3b6e237b2dd6d59a22bc473acdb67af5886c39da2\nspec.template.spec.containers.0.resources.requests.cpu=faffa5ac848811d8696883abb6cc8a3fb969f5e8fd0d01ba05f5548239021783\nspec.template.spec.containers.0.resources.requests.memory=1306e550ae337d714509c29593c3206953a97c28c691de5fd076aa0f0fb8e180\nspec.template.spec.containers.0.terminationMessagePath=b4233eab819d8ac0fcf88d898f421811a69431b589b99b0566fc5d2e93f8d51b\nspec.template.spec.containers.0.terminationMessagePolicy=50009ce1da4d15e1c4a04024df691eed5f0d598e2c4c67092f205366d0adf99e\nspec.template.spec.containers.0.volumeMounts.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.volumeMounts.0.mountPath=f2faf809c8aa079f24a51a3652d62801b3316c01777f034cb64de66b8ab297b3\nspec.template.spec.containers.0.volumeMounts.0.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336\nspec.template.spec.dnsPolicy=a6fa189cbc86bdda65887ed55da47e8c1e09bb263e1a2c978d7f9aaede2d7ec9\nspec.template.spec.priorityClassName=531bc7e09f78453c899c5193bdf009f12236bf0eb9b317c222aa0b2569722f02\nspec.template.spec.restartPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.schedulerName=6a1fba091ce95fd821cfa7b9d45e24391aa1902cccef6c5807c56cafbb324851\nspec.template.spec.securityContext.fsGroup=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.securityContext.runAsUser=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.serviceAccount=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.serviceAccountName=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.terminationGracePeriodSeconds=624b60c58c9d8bfb6ff1886c2fd605d2adeb6ea4da576068201b6c6958ce93f4\nspec.template.spec.volumes.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.volumes.0.name=06298432e8066b29e2223bcc23aa9504b56ae508fabf3435508869b9c3190e22\nspec.template.spec.volumes.0.secret.defaultMode=db55da3fc3098e9c42311c6013304ff36b19ef73d12ea932054b5ad51df4f49d\nspec.template.spec.volumes.0.secret.secretName=21e72c0fdfbf18403978b00e2ccc30c8c29480ebb29087fe09866c90333f4d78\nspec.template.spec.volumes.1.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336",
			expectedFingerprint: "5c0b0545c1b6fffe489d1cb3f9fe96a577e0cff37334cafc00af8ae4d7dbdf9e",
			expectedDrift:       false,
		},
		{
			description:         "Deployment manifest with changes should show drift",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        withAlteredField(loadRealDeploymentManifest(), "name-changed", "metadata", "name").Raw.Object,
			expectedFields:      "apiVersion=024614bbf9753e35bd0e7e47cf2f1d05243368e9b54ae6f53f8c80e152530aed\nkind=870a8ffd98f4f2bd5041ee4cebde82de4bdeb253fac88c5469a2f15f15614186\nmetadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nmetadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nmetadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nmetadata.annotations.deployment.kubernetes.io/revision=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nmetadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nmetadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nmetadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.name=194e285f328d7f2a1ea257da138d1e670b142dc01df00bacb7f915f4e90533b0\nmetadata.namespace=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.progressDeadlineSeconds=284b7e6d788f363f910f7beb1910473e23ce9d6c871f1ce0f31f22a982d48ad4\nspec.replicas=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.revisionHistoryLimit=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nspec.selector.matchLabels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.strategy.rollingUpdate.maxSurge=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.rollingUpdate.maxUnavailable=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.type=8f434f8bcd785d91c7e6c0394b1a3ddd503ece214c8aea23b8446ee389456f96\nspec.template.metadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.template.metadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nspec.template.metadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nspec.template.metadata.annotations.kubectl.terraform.test/envoy=b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b\nspec.template.metadata.annotations.kubectl.terraform.test/telegraf=22dcc997cd3a5311709bb7fd75ea4e92d6d76daa87ff6a0e4b2eef9b85129736\nspec.template.metadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nspec.template.metadata.creationTimestamp=426b5dfece37e413f559015825ebc7c5ba251a13028e2fcd5ed36df57be00b6c\nspec.template.metadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nspec.template.metadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.labels.app.kubernetes.io/version=da7965ad0f9dddb433712a4ee3b014e42c70a4c1d560187dcf446cba4cdd4860\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=b2c21b8c6d5bc3c78a90d2e5974ba05916e2f777d1a85cf3d2832cdc9c487093\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=68ab84f7c6d0f5781585eb1b5289499fb29081b918f71ebddb5f72021c9ef9c5\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=d26ba3a6c4fcf2b72755b5a23fee0a5994a39b71bdc9db6ef6d083a22a001353\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=b7a56873cd771f2c446d369b649430b65a756ba278ff97ec81bb6f55b2e73569\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=93966ab96ee1d305f0346ca2acf3b7e16c004a07bf322671293db9f19c18b079\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=ad57366865126e55649ecb23ae1d48887544976efea46a48eb5d85a6eeb4d306\nspec.template.spec.containers.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.0.configMapRef.name=5fca260b630569e695d3345f5ed80bf350446b6e85014b8365c45fe9e139d20c\nspec.template.spec.containers.0.image=5f67805ae6c365cf598b82eb292e333e6f64f85869ccf89b0c0f6cae61c1ec5b\nspec.template.spec.containers.0.imagePullPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.containers.0.name=1fe289205936c3fdb61158223892c7a8bee6ff4dfa085ea1c094ce0294e32114\nspec.template.spec.containers.0.resources.limits.cpu=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.resources.limits.memory=861c482be953844b2a4cc7f3b6e237b2dd6d59a22bc473acdb67af5886c39da2\nspec.template.spec.containers.0.resources.requests.cpu=faffa5ac848811d8696883abb6cc8a3fb969f5e8fd0d01ba05f5548239021783\nspec.template.spec.containers.0.resources.requests.memory=1306e550ae337d714509c29593c3206953a97c28c691de5fd076aa0f0fb8e180\nspec.template.spec.containers.0.terminationMessagePath=b4233eab819d8ac0fcf88d898f421811a69431b589b99b0566fc5d2e93f8d51b\nspec.template.spec.containers.0.terminationMessagePolicy=50009ce1da4d15e1c4a04024df691eed5f0d598e2c4c67092f205366d0adf99e\nspec.template.spec.containers.0.volumeMounts.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.volumeMounts.0.mountPath=f2faf809c8aa079f24a51a3652d62801b3316c01777f034cb64de66b8ab297b3\nspec.template.spec.containers.0.volumeMounts.0.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336\nspec.template.spec.dnsPolicy=a6fa189cbc86bdda65887ed55da47e8c1e09bb263e1a2c978d7f9aaede2d7ec9\nspec.template.spec.priorityClassName=531bc7e09f78453c899c5193bdf009f12236bf0eb9b317c222aa0b2569722f02\nspec.template.spec.restartPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.schedulerName=6a1fba091ce95fd821cfa7b9d45e24391aa1902cccef6c5807c56cafbb324851\nspec.template.spec.securityContext.fsGroup=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.securityContext.runAsUser=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.serviceAccount=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.serviceAccountName=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.terminationGracePeriodSeconds=624b60c58c9d8bfb6ff1886c2fd605d2adeb6ea4da576068201b6c6958ce93f4\nspec.template.spec.volumes.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.volumes.0.name=06298432e8066b29e2223bcc23aa9504b56ae508fabf3435508869b9c3190e22\nspec.template.spec.volumes.0.secret.defaultMode=db55da3fc3098e9c42311c6013304ff36b19ef73d12ea932054b5ad51df4f49d\nspec.template.spec.volumes.0.secret.secretName=21e72c0fdfbf18403978b00e2ccc30c8c29480ebb29087fe09866c90333f4d78\nspec.template.spec.volumes.1.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336",
			expectedFingerprint: "438901c57e3fe7c1cdc5280556b006f76d9d5da28d0ac32306401aa6bc481699",
			expectedDrift:       true,
		},
		{
			description:         "Deployment manifest with generation changes should not show drift",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        withAlteredField(loadRealDeploymentManifest(), "123", "metadata", "generation").Raw.Object,
			expectedFields:      "apiVersion=024614bbf9753e35bd0e7e47cf2f1d05243368e9b54ae6f53f8c80e152530aed\nkind=870a8ffd98f4f2bd5041ee4cebde82de4bdeb253fac88c5469a2f15f15614186\nmetadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nmetadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nmetadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nmetadata.annotations.deployment.kubernetes.io/revision=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nmetadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nmetadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nmetadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.namespace=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.progressDeadlineSeconds=284b7e6d788f363f910f7beb1910473e23ce9d6c871f1ce0f31f22a982d48ad4\nspec.replicas=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.revisionHistoryLimit=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nspec.selector.matchLabels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.strategy.rollingUpdate.maxSurge=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.rollingUpdate.maxUnavailable=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.type=8f434f8bcd785d91c7e6c0394b1a3ddd503ece214c8aea23b8446ee389456f96\nspec.template.metadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.template.metadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nspec.template.metadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nspec.template.metadata.annotations.kubectl.terraform.test/envoy=b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b\nspec.template.metadata.annotations.kubectl.terraform.test/telegraf=22dcc997cd3a5311709bb7fd75ea4e92d6d76daa87ff6a0e4b2eef9b85129736\nspec.template.metadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nspec.template.metadata.creationTimestamp=426b5dfece37e413f559015825ebc7c5ba251a13028e2fcd5ed36df57be00b6c\nspec.template.metadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nspec.template.metadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.labels.app.kubernetes.io/version=da7965ad0f9dddb433712a4ee3b014e42c70a4c1d560187dcf446cba4cdd4860\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=b2c21b8c6d5bc3c78a90d2e5974ba05916e2f777d1a85cf3d2832cdc9c487093\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=68ab84f7c6d0f5781585eb1b5289499fb29081b918f71ebddb5f72021c9ef9c5\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=d26ba3a6c4fcf2b72755b5a23fee0a5994a39b71bdc9db6ef6d083a22a001353\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=b7a56873cd771f2c446d369b649430b65a756ba278ff97ec81bb6f55b2e73569\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=93966ab96ee1d305f0346ca2acf3b7e16c004a07bf322671293db9f19c18b079\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=ad57366865126e55649ecb23ae1d48887544976efea46a48eb5d85a6eeb4d306\nspec.template.spec.containers.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.0.configMapRef.name=5fca260b630569e695d3345f5ed80bf350446b6e85014b8365c45fe9e139d20c\nspec.template.spec.containers.0.image=5f67805ae6c365cf598b82eb292e333e6f64f85869ccf89b0c0f6cae61c1ec5b\nspec.template.spec.containers.0.imagePullPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.containers.0.name=1fe289205936c3fdb61158223892c7a8bee6ff4dfa085ea1c094ce0294e32114\nspec.template.spec.containers.0.resources.limits.cpu=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.resources.limits.memory=861c482be953844b2a4cc7f3b6e237b2dd6d59a22bc473acdb67af5886c39da2\nspec.template.spec.containers.0.resources.requests.cpu=faffa5ac848811d8696883abb6cc8a3fb969f5e8fd0d01ba05f5548239021783\nspec.template.spec.containers.0.resources.requests.memory=1306e550ae337d714509c29593c3206953a97c28c691de5fd076aa0f0fb8e180\nspec.template.spec.containers.0.terminationMessagePath=b4233eab819d8ac0fcf88d898f421811a69431b589b99b0566fc5d2e93f8d51b\nspec.template.spec.containers.0.terminationMessagePolicy=50009ce1da4d15e1c4a04024df691eed5f0d598e2c4c67092f205366d0adf99e\nspec.template.spec.containers.0.volumeMounts.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.volumeMounts.0.mountPath=f2faf809c8aa079f24a51a3652d62801b3316c01777f034cb64de66b8ab297b3\nspec.template.spec.containers.0.volumeMounts.0.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336\nspec.template.spec.dnsPolicy=a6fa189cbc86bdda65887ed55da47e8c1e09bb263e1a2c978d7f9aaede2d7ec9\nspec.template.spec.priorityClassName=531bc7e09f78453c899c5193bdf009f12236bf0eb9b317c222aa0b2569722f02\nspec.template.spec.restartPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.schedulerName=6a1fba091ce95fd821cfa7b9d45e24391aa1902cccef6c5807c56cafbb324851\nspec.template.spec.securityContext.fsGroup=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.securityContext.runAsUser=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.serviceAccount=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.serviceAccountName=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.terminationGracePeriodSeconds=624b60c58c9d8bfb6ff1886c2fd605d2adeb6ea4da576068201b6c6958ce93f4\nspec.template.spec.volumes.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.volumes.0.name=06298432e8066b29e2223bcc23aa9504b56ae508fabf3435508869b9c3190e22\nspec.template.spec.volumes.0.secret.defaultMode=db55da3fc3098e9c42311c6013304ff36b19ef73d12ea932054b5ad51df4f49d\nspec.template.spec.volumes.0.secret.secretName=21e72c0fdfbf18403978b00e2ccc30c8c29480ebb29087fe09866c90333f4d78\nspec.template.spec.volumes.1.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336",
			expectedFingerprint: "5c0b0545c1b6fffe489d1cb3f9fe96a577e0cff37334cafc00af8ae4d7dbdf9e",
			expectedDrift:       false,
		},
		{
			description:         "Deployment manifest with kubectl annotation changes should not show drift",
			userProvided:        loadRealDeploymentManifest().Raw.Object,
			liveManifest:        withAlteredField(loadRealDeploymentManifest(), "changed", "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration").Raw.Object,
			expectedFields:      "apiVersion=024614bbf9753e35bd0e7e47cf2f1d05243368e9b54ae6f53f8c80e152530aed\nkind=870a8ffd98f4f2bd5041ee4cebde82de4bdeb253fac88c5469a2f15f15614186\nmetadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nmetadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nmetadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nmetadata.annotations.deployment.kubernetes.io/revision=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nmetadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nmetadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nmetadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nmetadata.namespace=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.progressDeadlineSeconds=284b7e6d788f363f910f7beb1910473e23ce9d6c871f1ce0f31f22a982d48ad4\nspec.replicas=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.revisionHistoryLimit=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce\nspec.selector.matchLabels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.strategy.rollingUpdate.maxSurge=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.rollingUpdate.maxUnavailable=72da55d317fd997b93138b8646a2238e806a4c4566d1e854277b5a583d8aef23\nspec.strategy.type=8f434f8bcd785d91c7e6c0394b1a3ddd503ece214c8aea23b8446ee389456f96\nspec.template.metadata.annotations.artifact.spinnaker.io/location=bb7a7a27e307fbcae3a498b5fa18cf6d9f9ec756497cd98c12255a1639eb8f87\nspec.template.metadata.annotations.artifact.spinnaker.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.artifact.spinnaker.io/type=216ded8192cde024f33a1fc8fe9594cd48bca91a637f37793fabc402d07ce4fd\nspec.template.metadata.annotations.artifact.spinnaker.io/version=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nspec.template.metadata.annotations.kubectl.terraform.test/envoy=b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b\nspec.template.metadata.annotations.kubectl.terraform.test/telegraf=22dcc997cd3a5311709bb7fd75ea4e92d6d76daa87ff6a0e4b2eef9b85129736\nspec.template.metadata.annotations.moniker.spinnaker.io/application=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.annotations.moniker.spinnaker.io/cluster=6eb28159e0032bc1dcceb48d2024fd0851486cec17830f332d9d6d3ea116d872\nspec.template.metadata.creationTimestamp=426b5dfece37e413f559015825ebc7c5ba251a13028e2fcd5ed36df57be00b6c\nspec.template.metadata.labels.app.kubernetes.io/managed-by=16bddce66b69a75e784de97b0d09bfb6f7b6288e721146ac771d90ea64923a96\nspec.template.metadata.labels.app.kubernetes.io/name=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.metadata.labels.app.kubernetes.io/version=da7965ad0f9dddb433712a4ee3b014e42c70a4c1d560187dcf446cba4cdd4860\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.key=b2c21b8c6d5bc3c78a90d2e5974ba05916e2f777d1a85cf3d2832cdc9c487093\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.0.matchExpressions.0.values.0=68ab84f7c6d0f5781585eb1b5289499fb29081b918f71ebddb5f72021c9ef9c5\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.podAffinityTerm.topologyKey=d26ba3a6c4fcf2b72755b5a23fee0a5994a39b71bdc9db6ef6d083a22a001353\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.0.weight=b7a56873cd771f2c446d369b649430b65a756ba278ff97ec81bb6f55b2e73569\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.key=71dafa318ec2911c798e83bed9911c4a3579d9f70c24e2a9b0f7af9445fca167\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.operator=8bc1d53cc57c24b79bf7c260b1f3b29973caab7b8f501c33016b321ebfc274f1\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.labelSelector.matchExpressions.0.values.0=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.podAffinityTerm.topologyKey=93966ab96ee1d305f0346ca2acf3b7e16c004a07bf322671293db9f19c18b079\nspec.template.spec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.1.weight=ad57366865126e55649ecb23ae1d48887544976efea46a48eb5d85a6eeb4d306\nspec.template.spec.containers.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.envFrom.0.configMapRef.name=5fca260b630569e695d3345f5ed80bf350446b6e85014b8365c45fe9e139d20c\nspec.template.spec.containers.0.image=5f67805ae6c365cf598b82eb292e333e6f64f85869ccf89b0c0f6cae61c1ec5b\nspec.template.spec.containers.0.imagePullPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.containers.0.name=1fe289205936c3fdb61158223892c7a8bee6ff4dfa085ea1c094ce0294e32114\nspec.template.spec.containers.0.resources.limits.cpu=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.resources.limits.memory=861c482be953844b2a4cc7f3b6e237b2dd6d59a22bc473acdb67af5886c39da2\nspec.template.spec.containers.0.resources.requests.cpu=faffa5ac848811d8696883abb6cc8a3fb969f5e8fd0d01ba05f5548239021783\nspec.template.spec.containers.0.resources.requests.memory=1306e550ae337d714509c29593c3206953a97c28c691de5fd076aa0f0fb8e180\nspec.template.spec.containers.0.terminationMessagePath=b4233eab819d8ac0fcf88d898f421811a69431b589b99b0566fc5d2e93f8d51b\nspec.template.spec.containers.0.terminationMessagePolicy=50009ce1da4d15e1c4a04024df691eed5f0d598e2c4c67092f205366d0adf99e\nspec.template.spec.containers.0.volumeMounts.#=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b\nspec.template.spec.containers.0.volumeMounts.0.mountPath=f2faf809c8aa079f24a51a3652d62801b3316c01777f034cb64de66b8ab297b3\nspec.template.spec.containers.0.volumeMounts.0.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336\nspec.template.spec.dnsPolicy=a6fa189cbc86bdda65887ed55da47e8c1e09bb263e1a2c978d7f9aaede2d7ec9\nspec.template.spec.priorityClassName=531bc7e09f78453c899c5193bdf009f12236bf0eb9b317c222aa0b2569722f02\nspec.template.spec.restartPolicy=de9f057a471cdb8d3b082719bdc7ad2031788d042947349723fa83c9d13a517a\nspec.template.spec.schedulerName=6a1fba091ce95fd821cfa7b9d45e24391aa1902cccef6c5807c56cafbb324851\nspec.template.spec.securityContext.fsGroup=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.securityContext.runAsUser=ab9828ca390581b72629069049793ba3c99bb8e5e9e7b97a55c71957e04df9a3\nspec.template.spec.serviceAccount=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.serviceAccountName=4ff6303b0c4a88c0beb95a1ac1f3ab1bfefa326f74992cf42f7b9a04a5d8703d\nspec.template.spec.terminationGracePeriodSeconds=624b60c58c9d8bfb6ff1886c2fd605d2adeb6ea4da576068201b6c6958ce93f4\nspec.template.spec.volumes.#=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35\nspec.template.spec.volumes.0.name=06298432e8066b29e2223bcc23aa9504b56ae508fabf3435508869b9c3190e22\nspec.template.spec.volumes.0.secret.defaultMode=db55da3fc3098e9c42311c6013304ff36b19ef73d12ea932054b5ad51df4f49d\nspec.template.spec.volumes.0.secret.secretName=21e72c0fdfbf18403978b00e2ccc30c8c29480ebb29087fe09866c90333f4d78\nspec.template.spec.volumes.1.name=577375a8b9495741fb95e9d46a88c5f5b1ae99a863c59d0ae508596be0834336",
			expectedFingerprint: "5c0b0545c1b6fffe489d1cb3f9fe96a577e0cff37334cafc00af8ae4d7dbdf9e",
			expectedDrift:       false,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			userProvided := yaml.NewFromUnstructured(&unstructured.Unstructured{Object: tcase.userProvided})
			liveManifest := yaml.NewFromUnstructured(&unstructured.Unstructured{Object: tcase.liveManifest})

			var out bytes.Buffer
			log.SetOutput(&out)
			defer log.SetOutput(os.Stderr)

			fields := getLiveManifestFields_WithIgnoredFields(tcase.ignored, userProvided, liveManifest)
			assert.Equal(t, tcase.expectedFields, fields, "Expect the builder output to match")
			fingerprint := getFingerprint(fields)
			assert.Equal(t, tcase.expectedFingerprint, fingerprint, "Expect the builder output to match")

			if tcase.expectedDrift {
				assert.Contains(t, out.String(), "yaml drift", "Should have drift detected")
			} else {
				assert.NotContains(t, out.String(), "yaml drift", "Should not have drift detected")
			}
		})
	}
}

func TestAccKubectlServerSideValidationFailure(t *testing.T) {

	config := `
resource "kubectl_manifest" "test" {
  yaml_body = <<YAML
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress
spec:
  rules:
    - host: "test-a.proxypile.tk"
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: nginx.test-a.svc.cluster.local
                port:
                  number: 8080
YAML
}
`
	expectedError, _ := regexp.Compile(".*Invalid value: \"nginx.test-a.svc.cluster.local\": a DNS-1035 label must consist of lower case alphanumeric characters.*")
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				ExpectError: expectedError,
				Config:      config,
			},
		},
	})
}

func withAlteredField(manifest *yaml.Manifest, value interface{}, fields ...string) *yaml.Manifest {
	_ = unstructured.SetNestedField(manifest.Raw.Object, value, fields...)
	return manifest
}

func loadRealDeploymentManifest() *yaml.Manifest {
	manifest, _ := yaml.ParseYAML(`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    artifact.spinnaker.io/location: service-prod
    artifact.spinnaker.io/name: testapp
    artifact.spinnaker.io/type: kubernetes/deployment
    artifact.spinnaker.io/version: ""
    deployment.kubernetes.io/revision: "3"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"something"}
    moniker.spinnaker.io/application: testapp
    moniker.spinnaker.io/cluster: deployment testapp
  creationTimestamp: "2021-08-11T23:34:34Z"
  generation: 3
  labels:
    app.kubernetes.io/managed-by: spinnaker
    app.kubernetes.io/name: testapp
  name: testapp
  namespace: service-prod
  resourceVersion: "5283884480"
  uid: cd198383-15da-4ec5-88d0-926e9bda484f
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: testapp
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      annotations:
        artifact.spinnaker.io/location: service-prod
        artifact.spinnaker.io/name: testapp
        artifact.spinnaker.io/type: kubernetes/deployment
        artifact.spinnaker.io/version: ""
        kubectl.terraform.test/envoy: "true"
        kubectl.terraform.test/telegraf: statsd
        moniker.spinnaker.io/application: testapp
        moniker.spinnaker.io/cluster: deployment testapp
      creationTimestamp: null
      labels:
        app.kubernetes.io/managed-by: spinnaker
        app.kubernetes.io/name: testapp
        app.kubernetes.io/version: 1.0.918
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubectl.terraform.test/instance-class
                operator: In
                values:
                - cpu
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - testapp
              topologyKey: topology.kubernetes.io/zone
            weight: 25
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - testapp
              topologyKey: kubernetes.io/hostname
            weight: 100
      containers:
      - envFrom:
        - configMapRef:
            name: testapp-config-v001
        image: my-registry/application/testapp:1.0.918
        imagePullPolicy: Always
        name: application
        resources:
          limits:
            cpu: "1"
            memory: 1Gi
          requests:
            cpu: 50m
            memory: 256Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /share
          name: shared-data
      dnsPolicy: ClusterFirst
      priorityClassName: spinnaker-managed-deployment
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext:
        fsGroup: 1100
        runAsUser: 1100
      serviceAccount: testapp
      serviceAccountName: testapp
      terminationGracePeriodSeconds: 30
      volumes:
      - name: cert
        secret:
          defaultMode: 420
          secretName: testapp-cert
      - emptyDir: {}
        name: shared-data
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2021-08-11T23:34:34Z"
    lastUpdateTime: "2021-08-18T20:05:00Z"
    message: ReplicaSet "testapp-59d5f58d75" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  - lastTransitionTime: "2021-09-30T13:10:54Z"
    lastUpdateTime: "2021-09-30T13:10:54Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  observedGeneration: 3
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
`)

	return manifest
}
