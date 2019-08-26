package kubernetes

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/icza/dyno"
	yamlParser "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1_unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	meta_v1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	// "k8s.io/klog"
)

func resourceKubectlManifest() *schema.Resource {
	// klog.SetOutput(os.Stdout)
	return &schema.Resource{
		Create: func(d *schema.ResourceData, meta interface{}) error {
			return backoff.Retry(func() error {
				err := resourceKubectlManifestCreate(d, meta)
				if err != nil {
					return err
				}
				return err
			}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), kubectlCreateRetryCount))
		},
		Read:   resourceKubectlManifestRead,
		Exists: resourceKubectlManifestExists,
		Delete: resourceKubectlManifestDelete,
		Update: resourceKubectlManifestUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		CustomizeDiff: func(d *schema.ResourceDiff, meta interface{}) error {

			// trigger a recreation if the yaml-body has any pending changes
			if d.Get("force_new").(bool) {
				d.ForceNew("yaml_body")
			}

			parsedYaml, err := parseYaml(d.Get("yaml_body").(string))
			if err != nil {
				return err
			}

			d.SetNew("api_version", parsedYaml.GetAPIVersion())
			d.SetNew("kind", parsedYaml.GetKind())
			d.SetNew("namespace", parsedYaml.GetNamespace())
			d.SetNew("name", parsedYaml.GetName())

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
				log.Printf("[CUSTOMDIFF] DETECTED %s vs %s", UID, createdAtUID)
				d.SetNewComputed("uid")
				return nil
			}

			if resourceVersion != createdAtResourceVersion {
				log.Printf("[CUSTOMDIFF] DETECTED RESOURCE VERSION %s vs %s", resourceVersion, createdAtResourceVersion)
				// Check that the fields specified in our YAML for diff against cluster representation
				stateYaml := d.Get("yaml_incluster").(string)
				liveStateYaml := d.Get("live_manifest_incluster").(string)
				if stateYaml != liveStateYaml {
					log.Printf("[CUSTOMDIFF] DETECTED YAML STATE %s vs %s", stateYaml, liveStateYaml)
					d.SetNewComputed("yaml_incluster")

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
				Type:     schema.TypeString,
				Computed: true,
			},
			"live_manifest_incluster": {
				Type:     schema.TypeString,
				Computed: true,
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
				Type:     schema.TypeString,
				Required: true,
			},
			"force_new": {
				Type:        schema.TypeBool,
				Description: "Default to update in-place. Setting to true will delete and create the kubernetes instead.",
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func resourceKubectlManifestCreate(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kubernetes rest client for resource: %+v", err)
	}

	// Create the resource in Kubernetes
	response, err := client.Create(rawObj, meta_v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create resource in kubernetes: %+v", err)
	}

	d.SetId(response.GetSelfLink())

	// Capture the UID and Resource_version at time of creation
	// this allows us to diff these against the actual values
	// read in by the 'resourceKubectlManifestRead'
	d.Set("uid", response.GetUID())
	d.Set("resource_version", response.GetResourceVersion())
	comparisonString, err := compareMaps(rawObj.UnstructuredContent(), response.UnstructuredContent())
	if err != nil {
		return err
	}

	log.Printf("[COMPAREOUT] %+v\n", comparisonString)
	d.Set("yaml_incluster", comparisonString)

	return resourceKubectlManifestRead(d, meta)
}

func resourceKubectlManifestUpdate(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kubernetes rest client for resource: %+v", err)
	}

	// Update the resource in Kubernetes
	response, err := client.Update(rawObj, meta_v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update resource in kubernetes: %+v", err)
	}

	d.SetId(response.GetSelfLink())
	// Capture the UID and Resource_version at time of update
	// this allows us to diff these against the actual values
	// read in by the 'resourceKubectlManifestRead'
	d.Set("uid", response.GetUID())
	d.Set("resource_version", response.GetResourceVersion())
	comparisonString, err := compareMaps(rawObj.UnstructuredContent(), response.UnstructuredContent())
	if err != nil {
		return err
	}

	log.Printf("[COMPAREOUT] %+v\n", comparisonString)
	d.Set("yaml_incluster", comparisonString)

	return resourceKubectlManifestRead(d, meta)
}

func resourceKubectlManifestRead(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kubernetes rest client for resource: %+v", err)
	}

	// Get the resource from Kubernetes
	metaObjLive, err := client.Get(rawObj.GetName(), meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get resource '%s' from kubernetes: %+v", metaObjLive.GetSelfLink(), err)
	}

	if metaObjLive.GetUID() == "" {
		return fmt.Errorf("Failed to parse item and get UUID: %+v", metaObjLive)
	}

	// Capture the UID and Resource_version from the cluster at the current time
	d.Set("live_uid", metaObjLive.GetUID())
	d.Set("live_resource_version", metaObjLive.GetResourceVersion())

	comparisonOutput, err := compareMaps(rawObj.UnstructuredContent(), metaObjLive.UnstructuredContent())
	if err != nil {
		return err
	}

	d.Set("live_manifest_incluster", comparisonOutput)

	return nil
}

func resourceKubectlManifestDelete(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	client, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kubernetes rest client for resource: %+v", err)
	}

	metaObj := &meta_v1beta1.PartialObjectMetadata{}
	err = client.Delete(rawObj.GetName(), &meta_v1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete kubernetes resource '%s': %+v", metaObj.SelfLink, err)
	}

	// Success remove it from state
	d.SetId("")

	return nil
}

func resourceKubectlManifestExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	yaml := d.Get("yaml_body").(string)

	client, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return false, fmt.Errorf("failed to create kubernetes rest client for resource: %+v", err)
	}

	metaObj, err := client.Get(rawObj.GetName(), meta_v1.GetOptions{})
	exists := !errors.IsGone(err) || !errors.IsNotFound(err)
	if err != nil && !exists {
		return false, fmt.Errorf("failed to get resource '%s' from kubernetes: %+v", metaObj.GetSelfLink(), err)
	}
	if exists {
		return true, nil
	}
	return false, nil
}

// To make things play nice we need the JSON representation of the object as the `RawObj`
// 1. UnMarshal YAML into map
// 2. Marshal map into JSON
// 3. UnMarshal JSON into the Unstructured type so we get some K8s checking
func parseYaml(yaml string) (*meta_v1_unstruct.Unstructured, error) {
	rawYamlParsed := &map[string]interface{}{}
	err := yamlParser.Unmarshal([]byte(yaml), rawYamlParsed)
	if err != nil {
		return nil, err
	}

	rawJSON, err := json.Marshal(dyno.ConvertMapI2MapS(*rawYamlParsed))
	if err != nil {
		return nil, err
	}

	unstrut := meta_v1_unstruct.Unstructured{}
	err = unstrut.UnmarshalJSON(rawJSON)
	if err != nil {
		return nil, err
	}

	unstructContent := unstrut.UnstructuredContent()
	log.Printf("[UNSTRUCT]: %+v\n", unstructContent)

	return &unstrut, nil
}

func getRestClientFromYaml(yaml string, provider KubeProvider) (dynamic.ResourceInterface, *meta_v1_unstruct.Unstructured, error) {
	unstrut, err := parseYaml(yaml)
	if err != nil {
		return nil, nil, err
	}

	// Use the k8s Discovery service to find all valid APIs for this cluster
	clientSet, config := provider()
	discoveryClient := clientSet.Discovery()
	resources, err := discoveryClient.ServerResources()
	// There is a partial failure mode here where not all groups are returned `GroupDiscoveryFailedError`
	// we'll try and continue in this condition as it's likely something we don't need
	// and if it is the `checkAPIResourceIsPresent` check will fail and stop the process
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, nil, err
	}

	// Validate that the APIVersion provided in the YAML is valid for this cluster
	apiResource, exists := checkAPIResourceIsPresent(resources, *unstrut)
	if !exists {
		return nil, nil, fmt.Errorf("resource provided in yaml isn't valid for cluster, check the APIVersion and Kind fields are valid")
	}

	resource := k8sschema.GroupVersionResource{Group: apiResource.Group, Version: apiResource.Version, Resource: apiResource.Name}
	// For core services (ServiceAccount, Service etc) the group is incorrectly parsed.
	// "v1" should be empty group and "v1" for verion
	if resource.Group == "v1" && resource.Version == "" {
		resource.Group = ""
		resource.Version = "v1"
	}
	client := dynamic.NewForConfigOrDie(&config).Resource(resource)

	if apiResource.Namespaced {
		namespace := unstrut.GetNamespace()
		if namespace == "" {
			namespace = "default"
		}
		return client.Namespace(namespace), unstrut, nil
	}

	return client, unstrut, nil
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
