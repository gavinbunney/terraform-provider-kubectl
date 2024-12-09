package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var testAccProviderFactories map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"kubectl": testAccProvider,
	}
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		"kubectl": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccCheckkubectlDestroy(s *terraform.State) error {
	return testAccCheckkubectlStatus(s, false)
}

func testAccCheckkubectlExists(s *terraform.State) error {
	return testAccCheckkubectlStatus(s, true)
}

func testAccCheckkubectlStatus(s *terraform.State, shouldExist bool) error {
	provider := testAccProvider.Meta().(*KubeProvider)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubectl_manifest" {
			continue
		}

		content, err := provider.MainClientset.RESTClient().Get().AbsPath(rs.Primary.ID).DoRaw(context.TODO())
		if (errors.IsNotFound(err) || errors.IsGone(err)) && shouldExist {
			return fmt.Errorf("Failed to find resource, likely a failure to create occured: %+v %v", err, string(content))
		}

	}

	return nil
}

func TestAccAuthExecPlugin(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}

	// load the kubeconfig for k3s and parse it manually
	// so we can invoke the exec plugin path in acc tests
	raw, err := os.ReadFile("../scripts/kubeconfig.yaml")
	require.NoErrorf(t, err, "failed to read k3s kubeconfig file: %v", err)

	config, err := clientcmd.Load(raw)
	require.NoErrorf(t, err, "failed to parse k3s kubeconfig file: %v", err)

	cc := config.Contexts[config.CurrentContext]
	cluster := config.Clusters[cc.Cluster]
	auth := config.AuthInfos[cc.AuthInfo]

	// double-escape the cert details so bash/echo will not interpret them
	authClientCert := strings.ReplaceAll(string(auth.ClientCertificateData), "\n", "\\\\n")
	authClientKey := strings.ReplaceAll(string(auth.ClientKeyData), "\n", "\\\\n")

	newProvider := Provider()
	newProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(map[string]interface{}{
		"host":                   cluster.Server,
		"cluster_ca_certificate": string(cluster.CertificateAuthorityData),
		"load_config_file":       false,
		"exec": []interface{}{
			map[string]interface{}{
				"api_version": "client.authentication.k8s.io/v1beta1",
				"command":     "/bin/sh",
				"args": []interface{}{
					"-c",
					fmt.Sprintf(`echo '{"apiVersion": "client.authentication.k8s.io/v1beta1","kind": "ExecCredential","status": {"clientCertificateData": "%s","clientKeyData": "%s"}}'`, authClientCert, authClientKey),
				},
			},
		},
	}))

	provider := newProvider.Meta().(*KubeProvider)
	discoveryClient, err := provider.ToDiscoveryClient()
	if err != nil {
		t.Errorf("failed to create discovery client: %v", err)
	}

	discoveryClient.Invalidate()
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		t.Errorf("failed to fetch server version: %v", err)
	}

	assert.NotNil(t, serverVersion.Major)
	assert.NotNil(t, serverVersion.Minor)
	assert.NotNil(t, serverVersion.Platform)
}

func unsetEnv(t *testing.T) func() {
	e := getEnv()

	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Error unsetting env var KUBECONFIG: %s", err)
	}
	if err := os.Unsetenv("KUBE_CONFIG"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CONFIG: %s", err)
	}
	if err := os.Unsetenv("KUBE_CTX"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CTX: %s", err)
	}
	if err := os.Unsetenv("KUBE_CTX_AUTH_INFO"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CTX_AUTH_INFO: %s", err)
	}
	if err := os.Unsetenv("KUBE_CTX_CLUSTER"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CTX_CLUSTER: %s", err)
	}
	if err := os.Unsetenv("KUBE_HOST"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_HOST: %s", err)
	}
	if err := os.Unsetenv("KUBE_USER"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_USER: %s", err)
	}
	if err := os.Unsetenv("KUBE_PASSWORD"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_PASSWORD: %s", err)
	}
	if err := os.Unsetenv("KUBE_CLIENT_CERT_DATA"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CLIENT_CERT_DATA: %s", err)
	}
	if err := os.Unsetenv("KUBE_CLIENT_KEY_DATA"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CLIENT_KEY_DATA: %s", err)
	}
	if err := os.Unsetenv("KUBE_CLUSTER_CA_CERT_DATA"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CLUSTER_CA_CERT_DATA: %s", err)
	}

	return func() {
		if err := os.Setenv("KUBE_CONFIG", e.Config); err != nil {
			t.Fatalf("Error resetting env var KUBE_CONFIG: %s", err)
		}
		if err := os.Setenv("KUBECONFIG", e.Config); err != nil {
			t.Fatalf("Error resetting env var KUBECONFIG: %s", err)
		}
		if err := os.Setenv("KUBE_CTX", e.Config); err != nil {
			t.Fatalf("Error resetting env var KUBE_CTX: %s", err)
		}
		if err := os.Setenv("KUBE_CTX_AUTH_INFO", e.CtxAuthInfo); err != nil {
			t.Fatalf("Error resetting env var KUBE_CTX_AUTH_INFO: %s", err)
		}
		if err := os.Setenv("KUBE_CTX_CLUSTER", e.CtxCluster); err != nil {
			t.Fatalf("Error resetting env var KUBE_CTX_CLUSTER: %s", err)
		}
		if err := os.Setenv("KUBE_HOST", e.Host); err != nil {
			t.Fatalf("Error resetting env var KUBE_HOST: %s", err)
		}
		if err := os.Setenv("KUBE_USER", e.User); err != nil {
			t.Fatalf("Error resetting env var KUBE_USER: %s", err)
		}
		if err := os.Setenv("KUBE_PASSWORD", e.Password); err != nil {
			t.Fatalf("Error resetting env var KUBE_PASSWORD: %s", err)
		}
		if err := os.Setenv("KUBE_CLIENT_CERT_DATA", e.ClientCertData); err != nil {
			t.Fatalf("Error resetting env var KUBE_CLIENT_CERT_DATA: %s", err)
		}
		if err := os.Setenv("KUBE_CLIENT_KEY_DATA", e.ClientKeyData); err != nil {
			t.Fatalf("Error resetting env var KUBE_CLIENT_KEY_DATA: %s", err)
		}
		if err := os.Setenv("KUBE_CLUSTER_CA_CERT_DATA", e.ClusterCACertData); err != nil {
			t.Fatalf("Error resetting env var KUBE_CLUSTER_CA_CERT_DATA: %s", err)
		}
	}
}

func getEnv() *currentEnv {
	e := &currentEnv{
		Ctx:               os.Getenv("KUBE_CTX"),
		CtxAuthInfo:       os.Getenv("KUBE_CTX_AUTH_INFO"),
		CtxCluster:        os.Getenv("KUBE_CTX_CLUSTER"),
		Host:              os.Getenv("KUBE_HOST"),
		User:              os.Getenv("KUBE_USER"),
		Password:          os.Getenv("KUBE_PASSWORD"),
		ClientCertData:    os.Getenv("KUBE_CLIENT_CERT_DATA"),
		ClientKeyData:     os.Getenv("KUBE_CLIENT_KEY_DATA"),
		ClusterCACertData: os.Getenv("KUBE_CLUSTER_CA_CERT_DATA"),
	}
	if cfg := os.Getenv("KUBE_CONFIG"); cfg != "" {
		e.Config = cfg
	}
	if cfg := os.Getenv("KUBECONFIG"); cfg != "" {
		e.Config = cfg
	}
	return e
}

type currentEnv struct {
	Config            string
	Ctx               string
	CtxAuthInfo       string
	CtxCluster        string
	Host              string
	User              string
	Password          string
	ClientCertData    string
	ClientKeyData     string
	ClusterCACertData string
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("KUBECONFIG"); v == "" {
		t.Fatal("KUBECONFIG must be set for acceptance tests")
	}
}
