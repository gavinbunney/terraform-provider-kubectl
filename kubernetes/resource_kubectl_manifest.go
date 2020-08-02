package kubernetes

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/printers"
	"os"
	"time"

	"log"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sresource "k8s.io/cli-runtime/pkg/resource"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/kubectl/pkg/cmd/apply"
	k8sdelete "k8s.io/kubectl/pkg/cmd/delete"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/icza/dyno"

	yamlParser "gopkg.in/yaml.v2"
	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1_unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	yamlWriter "sigs.k8s.io/yaml"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

const (
	// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go#L93
	TimedOutReason = "ProgressDeadlineExceeded"
)

func resourceKubectlManifest() *schema.Resource {

	return &schema.Resource{
		Create: func(d *schema.ResourceData, meta interface{}) error {
			exponentialBackoffConfig := backoff.NewExponentialBackOff()
			exponentialBackoffConfig.InitialInterval = 3 * time.Second
			exponentialBackoffConfig.MaxInterval = 30 * time.Second

			if kubectlApplyRetryCount > 0 {
				retryConfig := backoff.WithMaxRetries(exponentialBackoffConfig, kubectlApplyRetryCount)
				return backoff.Retry(func() error {
					err := resourceKubectlManifestApply(d, meta)
					if err != nil {
						log.Printf("[ERROR] creating manifest failed: %+v", err)
					}

					return err
				}, retryConfig)
			} else {
				return resourceKubectlManifestApply(d, meta)
			}
		},
		Read:   resourceKubectlManifestRead,
		Delete: resourceKubectlManifestDelete,
		Update: func(d *schema.ResourceData, meta interface{}) error {
			exponentialBackoffConfig := backoff.NewExponentialBackOff()
			exponentialBackoffConfig.InitialInterval = 3 * time.Second
			exponentialBackoffConfig.MaxInterval = 30 * time.Second

			if kubectlApplyRetryCount > 0 {
				retryConfig := backoff.WithMaxRetries(exponentialBackoffConfig, kubectlApplyRetryCount)
				return backoff.Retry(func() error {
					err := resourceKubectlManifestApply(d, meta)
					if err != nil {
						log.Printf("[ERROR] updating manifest failed: %+v", err)
					}
					return err
				}, retryConfig)
			} else {
				return resourceKubectlManifestApply(d, meta)
			}
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "//")
				if len(idParts) != 3 && len(idParts) != 4 {
					return []*schema.ResourceData{}, fmt.Errorf("expected ID in format apiVersion//kind//name//namespace, received: %s", d.Id())
				}

				apiVersion := idParts[0]
				kind := idParts[1]
				name := idParts[2]

				var yaml = ""
				if len(idParts) == 4 {
					yaml = fmt.Sprintf(`
apiVersion: %s
kind: %s
metadata:
  namespace: %s
  name: %s
`, apiVersion, kind, idParts[3], name)
				} else {
					yaml = fmt.Sprintf(`
apiVersion: %s
kind: %s
metadata:
  name: %s
`, apiVersion, kind, name)
				}

				manifest, err := parseYaml(yaml)
				if err != nil {
					return nil, err
				}
				client, err := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))

				if err != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to create kubernetes rest client for import of resource: %s %s %s %+v %s %s", apiVersion, kind, name, err, yaml, manifest.unstruct)
				}

				// Get the resource from Kubernetes
				metaObjLive, err := client.Get(manifest.unstruct.GetName(), meta_v1.GetOptions{})
				if err != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to get resource %s %s %s from kubernetes: %+v", apiVersion, kind, name, err)
				}

				if metaObjLive.GetUID() == "" {
					return []*schema.ResourceData{}, fmt.Errorf("failed to parse item and get UUID: %+v", metaObjLive)
				}

				// Capture the UID and Resource_version from the cluster at the current time

				_ = d.Set("uid", metaObjLive.GetUID())
				_ = d.Set("live_uid", metaObjLive.GetUID())
				_ = d.Set("resource_version", metaObjLive.GetResourceVersion())
				_ = d.Set("live_resource_version", metaObjLive.GetResourceVersion())

				var ignoreFields []string = nil
				ignoreFieldsRaw, hasIgnoreFields := d.GetOk("ignore_fields")
				if hasIgnoreFields {
					ignoreFields = expandStringList(ignoreFieldsRaw.([]interface{}))
				}

				comparisonOutput, err := compareMaps(metaObjLive.UnstructuredContent(), metaObjLive.UnstructuredContent(), ignoreFields)
				if err != nil {
					return []*schema.ResourceData{}, err
				}

				_ = d.Set("yaml_incluster", comparisonOutput)
				_ = d.Set("live_manifest_incluster", comparisonOutput)

				// set fields captured normally during creation/updates
				d.SetId(metaObjLive.GetSelfLink())
				_ = d.Set("api_version", metaObjLive.GetAPIVersion())
				_ = d.Set("kind", metaObjLive.GetKind())
				_ = d.Set("namespace", metaObjLive.GetNamespace())
				_ = d.Set("name", metaObjLive.GetName())
				_ = d.Set("force_new", false)

				// clear out fields user can't set to try and get parity with yaml_body
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Object, "metadata", "creationTimestamp")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Object, "metadata", "resourceVersion")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Object, "metadata", "selfLink")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Object, "metadata", "uid")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Object, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")

				if len(metaObjLive.GetAnnotations()) == 0 {
					meta_v1_unstruct.RemoveNestedField(metaObjLive.Object, "metadata", "annotations")
				}

				yamlJson, err := metaObjLive.MarshalJSON()
				if err != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to convert object to json: %+v", err)
				}

				yamlParsed, err := yamlWriter.JSONToYAML(yamlJson)
				if err != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to convert json to yaml: %+v", err)
				}

				_ = d.Set("yaml_body", string(yamlParsed))
				_ = d.Set("yaml_body_parsed", string(yamlParsed))

				return []*schema.ResourceData{d}, nil
			},
		},
		CustomizeDiff: func(d *schema.ResourceDiff, meta interface{}) error {

			// trigger a recreation if the yaml-body has any pending changes
			if d.Get("force_new").(bool) {
				_ = d.ForceNew("yaml_body")
			}

			if !d.NewValueKnown("yaml_body") {
				log.Printf("[TRACE] yaml_body value interpolated, skipping customized diff")
				return nil
			}

			parsedYaml, err := parseYaml(d.Get("yaml_body").(string))
			if err != nil {
				return err
			}

			// set calculated fields based on parsed yaml values
			_ = d.SetNew("api_version", parsedYaml.unstruct.GetAPIVersion())
			_ = d.SetNew("kind", parsedYaml.unstruct.GetKind())
			_ = d.SetNew("namespace", parsedYaml.unstruct.GetNamespace())
			_ = d.SetNew("name", parsedYaml.unstruct.GetName())

			// set the yaml_body_parsed field to provided value and obfuscate the yaml_body values manually
			// this allows us to show a nice diff to the users with specific fields obfuscated, whilst storing the
			// real value to apply in yaml_body
			obfuscatedYaml, _ := parseYaml(d.Get("yaml_body").(string))
			if obfuscatedYaml.unstruct.Object == nil {
				obfuscatedYaml.unstruct.Object = make(map[string]interface{})
			}

			var sensitiveFields []string
			sensitiveFieldsRaw, hasSensitiveFields := d.GetOk("sensitive_fields")
			if hasSensitiveFields {
				sensitiveFields = expandStringList(sensitiveFieldsRaw.([]interface{}))
			} else if parsedYaml.unstruct.GetKind() == "Secret" && parsedYaml.unstruct.GetAPIVersion() == "v1" {
				sensitiveFields = []string{"data"}
			}

			for _, s := range sensitiveFields {
				fields := strings.Split(s, ".")
				err = meta_v1_unstruct.SetNestedField(obfuscatedYaml.unstruct.Object, "(sensitive value)", fields...)
				if err != nil {
					return fmt.Errorf("failed to obfuscate sensitive field '%s': %+v\nNote: only map values are supported!", s, err)
				}
			}

			obfuscatedYamlBytes, obfuscatedYamlBytesErr := yamlWriter.Marshal(obfuscatedYaml.unstruct.Object)
			if obfuscatedYamlBytesErr != nil {
				return fmt.Errorf("failed to serialized obfuscated yaml: %+v", obfuscatedYamlBytesErr)
			}

			_ = d.SetNew("yaml_body_parsed", string(obfuscatedYamlBytes))

			// Get the UID of the K8s resource as it was when the `resourceKubectlManifestCreate` func completed.
			createdAtUID := d.Get("uid").(string)
			// Get the UID of the K8s resource as it currently is in the cluster.
			UID, exists := d.Get("live_uid").(string)
			if !exists {
				return nil
			}

			// Get the ResourceVersion of the K8s resource as it was when the `resourceKubectlManifestCreate` func completed.
			createdAtResourceVersion := d.Get("resource_version").(string)
			// Get it as it currently is in the cluster
			resourceVersion, exists := d.Get("live_resource_version").(string)
			if !exists {
				return nil
			}

			// If either UID or ResourceVersion differ between the current state and the cluster
			// trigger an update on the resource to get back in sync
			if UID != createdAtUID {
				log.Printf("[TRACE] DETECTED %s vs %s", UID, createdAtUID)
				_ = d.SetNewComputed("uid")
				return nil
			}

			if resourceVersion != createdAtResourceVersion {
				log.Printf("[TRACE] DETECTED RESOURCE VERSION %s vs %s", resourceVersion, createdAtResourceVersion)
				// Check that the fields specified in our YAML for diff against cluster representation
				stateYaml := d.Get("yaml_incluster").(string)
				liveStateYaml := d.Get("live_manifest_incluster").(string)
				if stateYaml != liveStateYaml {
					log.Printf("[TRACE] DETECTED YAML STATE %s vs %s", stateYaml, liveStateYaml)
					_ = d.SetNewComputed("yaml_incluster")
				}
				return nil
			}

			return nil
		},
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"live_uid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"live_resource_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"yaml_incluster": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"live_manifest_incluster": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"api_version": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"kind": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"namespace": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"yaml_body": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"yaml_body_parsed": {
				Type:        schema.TypeString,
				Description: "Yaml body that is being applied, with sensitive values unobfuscated",
				Computed:    true,
			},
			"sensitive_fields": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of yaml keys with sensitive values. Set these for fields which you want obfuscated in the yaml_body output",
				Optional:    true,
			},
			"force_new": {
				Type:        schema.TypeBool,
				Description: "Default to update in-place. Setting to true will delete and create the kubernetes instead.",
				Optional:    true,
				Default:     false,
			},
			"ignore_fields": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of yaml keys to ignore changes to. Set these for fields set by Operators or other processes in kubernetes and as such you don't want to update.",
				Optional:    true,
			},
			"wait_for_rollout": {
				Type:        schema.TypeBool,
				Description: "Default to true (waiting). Set this flag to wait or not for Deployments and APIService to complete rollout",
				Optional:    true,
				Default:     true,
			},
		},
	}
}

type UnstructuredManifest struct {
	unstruct *meta_v1_unstruct.Unstructured
}

func (m *UnstructuredManifest) hasNamespace() bool {
	ns := m.unstruct.GetNamespace()
	return ns != ""
}

func (m *UnstructuredManifest) String() string {
	if m.hasNamespace() {
		return fmt.Sprintf("%s/%s", m.unstruct.GetNamespace(), m.unstruct.GetName())
	}

	return m.unstruct.GetName()
}

func resourceKubectlManifestApply(d *schema.ResourceData, meta interface{}) error {

	yaml := d.Get("yaml_body").(string)
	manifest, err := parseYaml(yaml)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes resource: %+v", err)
	}

	log.Printf("[DEBUG] %v apply kubernetes resource:\n%s", manifest, yaml)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, err := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))
	if err != nil {
		return fmt.Errorf("%v failed to create kubernetes rest client for update of resource: %+v", manifest, err)
	}

	// Update the resource in Kubernetes, using a temp file
	tmpfile, _ := ioutil.TempFile("", "*kubectl_manifest.yaml")
	_, _ = tmpfile.Write([]byte(yaml))
	_ = tmpfile.Close()

	applyOptions := apply.NewApplyOptions(genericclioptions.IOStreams{
		In:     strings.NewReader(yaml),
		Out:    log.Writer(),
		ErrOut: log.Writer(),
	})
	applyOptions.Builder = k8sresource.NewBuilder(k8sresource.RESTClientGetter(meta.(*KubeProvider)))
	applyOptions.DeleteOptions = &k8sdelete.DeleteOptions{
		FilenameOptions: k8sresource.FilenameOptions{
			Filenames: []string{tmpfile.Name()},
		},
	}

	applyOptions.ToPrinter = func(string) (printers.ResourcePrinter, error) {
		return printers.NewDiscardingPrinter(), nil
	}

	if manifest.hasNamespace() {
		applyOptions.Namespace = manifest.unstruct.GetNamespace()
	}

	log.Printf("[INFO] %s perform apply of manifest", manifest)

	err = applyOptions.Run()
	_ = os.Remove(tmpfile.Name())
	if err != nil {
		return fmt.Errorf("%v failed to run apply options: %+v", manifest, err)
	}

	log.Printf("[INFO] %v manifest applied, fetch resource from kubernetes", manifest)

	// get the resource from Kubernetes
	response, err := client.Get(manifest.unstruct.GetName(), meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("%v failed to fetch resource from kubernetes: %+v", manifest, err)
	}

	d.SetId(response.GetSelfLink())
	log.Printf("[DEBUG] %v fetched successfully, set id to: %v", manifest, d.Id())

	// Capture the UID and Resource_version at time of update
	// this allows us to diff these against the actual values
	// read in by the 'resourceKubectlManifestRead'
	_ = d.Set("uid", response.GetUID())
	_ = d.Set("resource_version", response.GetResourceVersion())

	var ignoreFields []string = nil
	ignoreFieldsRaw, hasIgnoreFields := d.GetOk("ignore_fields")
	if hasIgnoreFields {
		ignoreFields = expandStringList(ignoreFieldsRaw.([]interface{}))
	}

	comparisonString, err := compareMaps(manifest.unstruct.UnstructuredContent(), response.UnstructuredContent(), ignoreFields)
	if err != nil {
		return fmt.Errorf("%v failed to compare maps of manifest vs version in kubernetes: %+v", manifest, err)
	}

	_ = d.Set("yaml_incluster", comparisonString)

	if d.Get("wait_for_rollout").(bool) {
		timeout := d.Timeout(schema.TimeoutCreate)

		if manifest.unstruct.GetKind() == "Deployment" {
			log.Printf("[INFO] %v waiting for deployment rollout for %vmin", manifest, timeout.Minutes())
			err = resource.Retry(timeout,
				waitForDeploymentReplicasFunc(meta.(*KubeProvider), manifest.unstruct.GetNamespace(), manifest.unstruct.GetName()))
			if err != nil {
				return err
			}
		} else if manifest.unstruct.GetKind() == "APIService" && manifest.unstruct.GetAPIVersion() == "apiregistration.k8s.io/v1" {
			log.Printf("[INFO] %v waiting for APIService rollout for %vmin", manifest, timeout.Minutes())
			err = resource.Retry(timeout,
				waitForAPIServiceAvailableFunc(meta.(*KubeProvider), manifest.unstruct.GetName()))
			if err != nil {
				return err
			}
		}
	}

	return resourceKubectlManifestReadUsingClient(d, meta, client, manifest)
}

func resourceKubectlManifestRead(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)
	manifest, err := parseYaml(yaml)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes resource: %+v", err)
	}

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, err := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kubernetes rest client for read of resource: %+v", err)
	}

	return resourceKubectlManifestReadUsingClient(d, meta, client, manifest)
}

func resourceKubectlManifestReadUsingClient(d *schema.ResourceData, meta interface{}, client dynamic.ResourceInterface, manifest *UnstructuredManifest) error {

	log.Printf("[DEBUG] %v fetch from kubernetes", manifest)

	// Get the resource from Kubernetes
	metaObjLive, err := client.Get(manifest.unstruct.GetName(), meta_v1.GetOptions{})
	resourceGone := errors.IsGone(err) || errors.IsNotFound(err)
	if resourceGone {
		log.Printf("[WARN] kubernetes resource (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("%v failed to get resource from kubernetes: %+v", manifest, err)
	}

	if metaObjLive.GetUID() == "" {
		return fmt.Errorf("%v failed to parse item and get UUID: %+v", manifest, metaObjLive)
	}

	// Capture the UID and Resource_version from the cluster at the current time
	_ = d.Set("live_uid", metaObjLive.GetUID())
	_ = d.Set("live_resource_version", metaObjLive.GetResourceVersion())

	var ignoreFields []string = nil
	ignoreFieldsRaw, hasIgnoreFields := d.GetOk("ignore_fields")
	if hasIgnoreFields {
		ignoreFields = expandStringList(ignoreFieldsRaw.([]interface{}))
	}
	comparisonOutput, err := compareMaps(manifest.unstruct.UnstructuredContent(), metaObjLive.UnstructuredContent(), ignoreFields)
	if err != nil {
		return fmt.Errorf("%v failed to compare maps: %+v", manifest, err)
	}

	_ = d.Set("live_manifest_incluster", comparisonOutput)

	return nil
}

func resourceKubectlManifestDelete(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)
	manifest, err := parseYaml(yaml)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes resource: %+v", err)
	}

	log.Printf("[DEBUG] %v delete kubernetes resource:\n%s", manifest, yaml)

	client, err := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))
	if err != nil {
		return fmt.Errorf("%v failed to create kubernetes rest client for delete of resource: %+v", manifest, err)
	}

	log.Printf("[INFO] %s perform delete of manifest", manifest)

	deletePropagationBackground := meta_v1.DeletePropagationBackground
	err = client.Delete(manifest.unstruct.GetName(), &meta_v1.DeleteOptions{PropagationPolicy: &deletePropagationBackground})
	resourceGone := errors.IsGone(err) || errors.IsNotFound(err)
	if err != nil && !resourceGone {
		return fmt.Errorf("%v failed to delete kubernetes resource: %+v", manifest, err)
	}

	// Success remove it from state
	d.SetId("")
	return nil
}

// To make things play nice we need the JSON representation of the object as the `RawObj`
// 1. UnMarshal YAML into map
// 2. Marshal map into JSON
// 3. UnMarshal JSON into the Unstructured type so we get some K8s checking
func parseYaml(yaml string) (*UnstructuredManifest, error) {
	rawYamlParsed := &map[string]interface{}{}
	err := yamlParser.Unmarshal([]byte(yaml), rawYamlParsed)
	if err != nil {
		return nil, err
	}

	rawJSON, err := json.Marshal(dyno.ConvertMapI2MapS(*rawYamlParsed))
	if err != nil {
		return nil, err
	}

	unstruct := meta_v1_unstruct.Unstructured{}
	err = unstruct.UnmarshalJSON(rawJSON)
	if err != nil {
		return nil, err
	}

	manifest := &UnstructuredManifest{
		unstruct: &unstruct,
	}

	log.Printf("[DEBUG] %s Unstructed YAML: %+v\n", manifest, manifest.unstruct.UnstructuredContent())
	return manifest, nil
}

func getRestClientFromUnstructured(manifest *UnstructuredManifest, provider *KubeProvider) (dynamic.ResourceInterface, error) {

	type Result struct {
		ResourceInterface dynamic.ResourceInterface
		Error             error
	}

	doGetRestClientFromUnstructured := func(manifest *UnstructuredManifest, provider *KubeProvider) *Result {
		// Use the k8s Discovery service to find all valid APIs for this cluster
		discoveryClient, _ := provider.ToDiscoveryClient()
		var resources []*meta_v1.APIResourceList
		var err error
		_, resources, err = discoveryClient.ServerGroupsAndResources()

		// There is a partial failure mode here where not all groups are returned `GroupDiscoveryFailedError`
		// we'll try and continue in this condition as it's likely something we don't need
		// and if it is the `checkAPIResourceIsPresent` check will fail and stop the process
		if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
			return &Result{nil, err}
		}

		// Validate that the APIVersion provided in the YAML is valid for this cluster
		apiResource, exists := checkAPIResourceIsPresent(resources, *manifest.unstruct)
		if !exists {
			// api not found, invalidate the cache and try again
			// this handles the case when a CRD is being created by another kubectl_manifest resource run
			discoveryClient.Invalidate()
			_, resources, err = discoveryClient.ServerGroupsAndResources()

			if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
				return &Result{nil, err}
			}

			// check for resource again
			apiResource, exists = checkAPIResourceIsPresent(resources, *manifest.unstruct)
			if !exists {
				return &Result{nil, fmt.Errorf("resource [%s/%s] isn't valid for cluster, check the APIVersion and Kind fields are valid", manifest.unstruct.GroupVersionKind().GroupVersion().String(), manifest.unstruct.GetKind())}
			}
		}

		resourceStruct := k8sschema.GroupVersionResource{Group: apiResource.Group, Version: apiResource.Version, Resource: apiResource.Name}
		// For core services (ServiceAccount, Service etc) the group is incorrectly parsed.
		// "v1" should be empty group and "v1" for version
		if resourceStruct.Group == "v1" && resourceStruct.Version == "" {
			resourceStruct.Group = ""
			resourceStruct.Version = "v1"
		}
		client := dynamic.NewForConfigOrDie(&provider.RestConfig).Resource(resourceStruct)

		if apiResource.Namespaced {
			if manifest.unstruct.GetNamespace() == "" {
				manifest.unstruct.SetNamespace("default")
			}
			return &Result{client.Namespace(manifest.unstruct.GetNamespace()), nil}
		}

		return &Result{client, nil}
	}

	discoveryWithTimeout := func(manifest *UnstructuredManifest, provider *KubeProvider) <-chan *Result {
		ch := make(chan *Result)
		go func() {
			ch <- doGetRestClientFromUnstructured(manifest, provider)
		}()
		return ch
	}

	select {
	case res := <-discoveryWithTimeout(manifest, provider):
		return res.ResourceInterface, res.Error
	case <-time.After(60 * time.Second):
		log.Printf("[ERROR] %v timed out fetching resources from discovery client", manifest)
		return nil, fmt.Errorf("%v timed out fetching resources from discovery client", manifest)
	}
}

// checkAPIResourceIsPresent Loops through a list of available APIResources and
// checks there is a resource for the APIVersion and Kind defined in the 'resource'
// if found it returns true and the APIResource which matched
func checkAPIResourceIsPresent(available []*meta_v1.APIResourceList, resource meta_v1_unstruct.Unstructured) (*meta_v1.APIResource, bool) {
	for _, rList := range available {
		if rList == nil {
			continue
		}
		group := rList.GroupVersion
		for _, r := range rList.APIResources {
			if group == resource.GroupVersionKind().GroupVersion().String() && r.Kind == resource.GetKind() {
				r.Group = rList.GroupVersion
				r.Kind = rList.Kind
				return &r, true
			}
		}
	}
	return nil, false
}

// GetDeploymentConditionInternal returns the condition with the provided type.
// Borrowed from: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go#L135
func GetDeploymentCondition(status apps_v1.DeploymentStatus, condType apps_v1.DeploymentConditionType) *apps_v1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func waitForDeploymentReplicasFunc(provider *KubeProvider, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {

		// Query the deployment to get a status update.
		dply, err := provider.MainClientset.AppsV1().Deployments(ns).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if dply.Generation <= dply.Status.ObservedGeneration {
			cond := GetDeploymentCondition(dply.Status, apps_v1.DeploymentProgressing)
			if cond != nil && cond.Reason == TimedOutReason {
				err := fmt.Errorf("Deployment exceeded its progress deadline: %v\nDeployment details:\n%v", cond.String(), dply.String())
				return resource.NonRetryableError(err)
			}

			if dply.Status.UpdatedReplicas < *dply.Spec.Replicas {
				return resource.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d out of %d new replicas have been updated...", dply.Status.UpdatedReplicas, dply.Spec.Replicas))
			}

			if dply.Status.Replicas > dply.Status.UpdatedReplicas {
				return resource.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d old replicas are pending termination...", dply.Status.Replicas-dply.Status.UpdatedReplicas))
			}

			if dply.Status.AvailableReplicas < dply.Status.UpdatedReplicas {
				return resource.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d of %d updated replicas are available...", dply.Status.AvailableReplicas, dply.Status.UpdatedReplicas))
			}
		} else if dply.Status.ObservedGeneration == 0 {
			return resource.RetryableError(fmt.Errorf("Waiting for rollout to start"))
		}
		return nil
	}
}

func waitForAPIServiceAvailableFunc(provider *KubeProvider, name string) resource.RetryFunc {
	return func() *resource.RetryError {

		apiService, err := provider.AggregatorClientset.ApiregistrationV1().APIServices().Get(name, meta_v1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		for i := range apiService.Status.Conditions {
			if apiService.Status.Conditions[i].Type == apiregistration.Available {
				return nil
			}
		}

		return resource.RetryableError(fmt.Errorf("Waiting for APIService %v to be Available", name))
	}
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []*string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, val)
		}
	}
	return vs
}
