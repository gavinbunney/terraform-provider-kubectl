package kubernetes

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/icza/dyno"
	yamlParser "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	k8meta "k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1_unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	meta_v1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	// serializer "k8s.io/apimachinery/pkg/runtime/serializer"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"os"
)

func resourceKubernetesYAML() *schema.Resource {
	klog.SetOutput(os.Stdout)
	return &schema.Resource{
		Create: resourceKubernetesYAMLCreate,
		Read:   resourceKubernetesYAMLRead,
		Exists: resourceKubernetesYAMLExists,
		Delete: resourceKubernetesYAMLDelete,
		Update: resourceKubernetesYAMLUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		CustomizeDiff: func(d *schema.ResourceDiff, meta interface{}) error {
			// Enable force new on yaml_body field.
			// This can't be done in the schema as it will fail internal validation
			// as all fields would be 'ForceNew' so no 'Update' func is needed.
			// but as we manually trigger an update in this compare function
			// we need the update function specified.
			d.ForceNew("yaml_body")

			// Get the UID of the K8s resource as it was when the `resourceKubernetesYAMLCreate` func completed.
			createdAtUID := d.Get("uid").(string)
			// Get the UID of the K8s resource as it currently is in the cluster.
			UID, exists := d.Get("live_uid").(string)
			if !exists {
				return nil
			}

			// Get the ResourceVersion of the K8s resource as it was when the `resourceKubernetesYAMLCreate` func completed.
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
				liveStateYaml := d.Get("live_yaml_incluster").(string)
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
			"live_yaml_incluster": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"yaml_body": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceKubernetesYAMLCreate(d *schema.ResourceData, meta interface{}) error {
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
	// read in by the 'resourceKubernetesYAMLRead'
	d.Set("uid", response.GetUID())
	d.Set("resource_version", response.GetResourceVersion())
	builder := strings.Builder{}
	err = compareObjs(rawObj, response, &builder)
	if err != nil {
		return err
	}
	d.Set("yaml_incluster", getMD5Hash(builder.String()))

	return resourceKubernetesYAMLRead(d, meta)
}

func resourceKubernetesYAMLRead(d *schema.ResourceData, meta interface{}) error {
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

	builder := strings.Builder{}
	err = compareObjs(rawObj, metaObjLive, &builder)
	if err != nil {
		return err
	}

	d.Set("live_yaml_incluster", getMD5Hash(builder.String()))

	return nil
}

func resourceKubernetesYAMLDelete(d *schema.ResourceData, meta interface{}) error {
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

func resourceKubernetesYAMLUpdate(d *schema.ResourceData, meta interface{}) error {
	err := resourceKubernetesYAMLDelete(d, meta)
	if err != nil {
		return err
	}
	return resourceKubernetesYAMLCreate(d, meta)
}

func resourceKubernetesYAMLExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

func getResourceFromK8s(client rest.Interface, absPaths map[string]string) (*meta_v1beta1.PartialObjectMetadata, runtime.Object, bool, error) {
	result := client.Get().AbsPath(absPaths["GET"]).Do()

	var statusCode int
	result.StatusCode(&statusCode)
	// Resource doesn't exist
	if statusCode != 200 {
		return nil, nil, false, nil
	}

	// Another error occured
	response, err := result.Get()
	if err != nil {
		return nil, nil, false, err
	}

	// Get the metadata we need
	metaObj, err := runtimeObjToMetaObj(response)
	if err != nil {
		return nil, nil, true, err
	}

	return metaObj, response, true, err
}

func getRestClientFromYaml(yaml string, provider KubeProvider) (dynamic.ResourceInterface, *meta_v1_unstruct.Unstructured, error) {
	// To make things play nice we need the JSON representation of the object as
	// the `RawObj`
	// 1. UnMarshal YAML into map
	// 2. Marshal map into JSON
	// 3. UnMarshal JSON into the Unstructured type so we get some K8s checking
	// 4. Marshal back into JSON ... now we know it's likely to play nice with k8s
	rawYamlParsed := &map[string]interface{}{}
	err := yamlParser.Unmarshal([]byte(yaml), rawYamlParsed)
	if err != nil {
		return nil, nil, err
	}

	rawJSON, err := json.Marshal(dyno.ConvertMapI2MapS(*rawYamlParsed))
	if err != nil {
		return nil, nil, err
	}

	unstrut := meta_v1_unstruct.Unstructured{}
	err = unstrut.UnmarshalJSON(rawJSON)
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
	apiResource, exists := checkAPIResourceIsPresent(resources, unstrut)
	if !exists {
		return nil, nil, fmt.Errorf("resource provided in yaml isn't valid for cluster, check the APIVersion and Kind fields are valid")
	}

	resource := k8sschema.GroupVersionResource{Group: apiResource.Group, Version: apiResource.Version, Resource: apiResource.Name}
	client := dynamic.NewForConfigOrDie(&config).Resource(resource)

	if apiResource.Namespaced {
		namespace := unstrut.GetNamespace()
		if namespace == "" {
			namespace = "default"
		}
		return client.Namespace(namespace), &unstrut, nil
	}

	return client, &unstrut, nil
}

// getResourceMetaObjFromYaml Uses the UniversalDeserializer to deserialize
// the yaml provided into a k8s runtime.Object
func getResourceMetaObjFromYaml(yaml string) (*meta_v1beta1.PartialObjectMetadata, runtime.Object, error) {
	decoder := scheme.Codecs.UniversalDeserializer()

	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("[INFO] Error parsing type: %#v", err)
		return nil, nil, err
	}
	metaObj, err := runtimeObjToMetaObj(obj)
	if err != nil {
		return nil, nil, err
	}
	return metaObj, obj, nil

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

// runtimeObjToMetaObj Gets a subset of the full object information
// just enough to construct the API Calls needed and detect any changes
// made to the object in the cluster (UID & ResourceVersion)
func runtimeObjToMetaObj(obj runtime.Object) (*meta_v1beta1.PartialObjectMetadata, error) {
	metaObj := k8meta.AsPartialObjectMetadata(obj.(meta_v1.Object))
	typeMeta, err := k8meta.TypeAccessor(obj)
	if err != nil {
		return nil, err
	}
	metaObj.TypeMeta = meta_v1.TypeMeta{
		APIVersion: typeMeta.GetAPIVersion(),
		Kind:       typeMeta.GetKind(),
	}
	if metaObj.Namespace == "" {
		metaObj.Namespace = "default"
	}
	return metaObj, nil
}

func compareObjs(original, returned interface{}, builder *strings.Builder) error {
	// Check originalObj is valid
	originalObj, err := conversion.EnforcePtr(original)
	if err != nil {
		return err
	}

	// Check returnedObj is valid
	returnedObj, err := conversion.EnforcePtr(returned)
	if err != nil {
		return err
	}
	return compareObjsInternal(originalObj, returnedObj, builder)
}

var skipFields = map[string]bool{
	"Status":            true,
	"Finalizers":        true,
	"Initializers":      true,
	"OwnerReferences":   true,
	"CreationTimestamp": true,
	"Generation":        true,
	"ResourceVersion":   true,
	"UID":               true,
}

func compareObjsInternal(originalObj, returnedObj reflect.Value, builder *strings.Builder) error {
	originalObType := originalObj.Type()
	if originalObType.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, but got %v: %v", originalObj.Kind(), originalObj)
	}

	returnedObjType := returnedObj.Type()
	if returnedObjType.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, but got %v: %v", returnedObj.Kind(), returnedObj)
	}

	// Loop through all fields on the original Obj
	// for each field on the original get it's value on the returned obj
	// and use this to build a hash
	for iO := 0; iO < originalObType.NumField(); iO++ {
		originalField := originalObType.Field(iO)

		for iR := 0; iR < returnedObjType.NumField(); iR++ {
			returnedField := returnedObjType.Field(iR)

			// Check we're comparing the right field
			if returnedField.Name != originalField.Name {
				log.Printf("[COMPARE] Skipping: %#v %#v", returnedField, originalField)
				continue
			}

			// Skip any fields we want to ignore
			if _, exists := skipFields[returnedField.Name]; exists {
				log.Printf("[COMPARE] Skipping as in SkipFields: %#v %#v", returnedField, originalField)
				continue
			}

			// Get the value of the field and pull value
			// out if the field is a ptr
			originalValue := originalObj.Field(iO)
			if originalValue.Kind() == reflect.Ptr {
				if originalValue.IsNil() {
					log.Printf("[COMPARE] Skipping as is nil ptr: %#v %#v", returnedField, originalField)
					continue
				}
				originalValue = originalValue.Elem()
			}
			returnedValue := returnedObj.Field(iO)
			if returnedValue.Kind() == reflect.Ptr {
				if returnedValue.IsNil() {
					log.Printf("[COMPARE] Skipping as is nil ptr: %#v %#v", returnedField, originalField)
					continue
				}
				returnedValue = returnedValue.Elem()
			}

			log.Printf("[COMPARE] Found matching field: %#v, %#v", returnedField.Name, returnedValue.Type().Kind().String())

			// Recurse into the struct to compare it's fields
			if returnedValue.Type().Kind() == reflect.Struct {
				log.Printf("[COMPARE] Found struct recurrsing: %#v", returnedField)

				err := compareObjsInternal(originalValue, returnedValue, builder)
				if err != nil {
					return err
				}
				continue
			}

			// Skip unneeded fields
			k := returnedValue.Kind()
			switch k {
			case reflect.String:
				if returnedValue.String() == "" {
					log.Printf("[COMPARE] Skipping empty string value: %#v %#v", returnedField, originalField)
					continue
				}
			case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:

				// We can check if these are nil
				if returnedValue.IsNil() {
					log.Printf("[COMPARE] Skipping nil value: %#v %#v", returnedField, originalField)
					continue
				}
			}

			// We can do a more detailed comparison on map fields
			if k == reflect.Map {
				log.Printf("[COMPARE] Comparing map: %#v %#v", returnedField, originalField)

				returnedKeys := returnedValue.MapKeys()
				originalKeys := originalValue.MapKeys()

				for _, oKey := range originalKeys {
					for _, rKey := range returnedKeys {
						if oKey.String() == rKey.String() {
							rValue := returnedValue.MapIndex(rKey)

							log.Printf("[COMPARE] Found matching map value. Writing to string builder: %s->%#v", returnedField.Name, returnedValue.Interface())
							builder.WriteString(fmt.Sprintf("fieldName:%s,keyName:%s,fieldValue:%v", returnedField.Name, oKey.String(), rValue.Interface()))
						}
					}
				}

				return nil
			}

			// We can do a more detailed comparison for arrays too
			if k == reflect.Slice {
				log.Printf("[COMPARE] Comparing slice: %#v %#v", returnedField, originalField)

				oSliceLen := originalValue.Len()
				rSliceLen := returnedValue.Len()
				if rSliceLen < oSliceLen {
					//Todo: what do we do here?
					panic("wrong size")
				}

				for i := 0; i < oSliceLen; i++ {
					log.Printf("[COMPARE] Recurse for Array/slice item: %#v %#v", returnedField, originalField)

					// Handle case in which this is an array of ints or strings NOT structs
					oValueSlice := originalValue.Index(i)
					rValueSlice := returnedValue.Index(i)
					rValueSliceKind := rValueSlice.Kind()
					if rValueSliceKind == reflect.String || rValueSliceKind == reflect.Int {
						builder.WriteString(fmt.Sprintf("fieldName:%s,fieldValue:%v", returnedField.Name+string(i), returnedValue.Interface()))
					} else {
						err := compareObjsInternal(oValueSlice, rValueSlice, builder)
						if err != nil {
							return err
						}
					}
				}

				return nil
			}

			if returnedValue.CanInterface() {
				log.Printf("[COMPARE] Found value writing to string builder: %s->%#v  (%#v)", returnedField.Name, returnedValue.Interface(), returnedValue.Kind().String())
				builder.WriteString(fmt.Sprintf("fieldName:%s,fieldValue:%v", returnedField.Name, returnedValue.Interface()))
			} else {
				log.Printf("[COMPARE] Found unsettable field :(: %#v", returnedField.Name)
			}
		}
	}

	return nil
}

func getMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
