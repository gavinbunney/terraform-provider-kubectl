package kubernetes

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	k8sdelete "k8s.io/kubectl/pkg/cmd/delete"

	"github.com/alekc/terraform-provider-kubectl/flatten"
	"github.com/alekc/terraform-provider-kubectl/internal/types"

	"github.com/alekc/terraform-provider-kubectl/yaml"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	validate2 "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/mapstructure"
	"github.com/thedevsaddam/gojsonq/v2"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/validation"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apiMachineryTypes "k8s.io/apimachinery/pkg/types"
	k8sresource "k8s.io/cli-runtime/pkg/resource"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/kubectl/pkg/cmd/apply"

	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		CreateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			// if there is no retry required, perform a simple apply
			if kubectlApplyRetryCount == 0 {
				if applyErr := resourceKubectlManifestApply(ctx, d, meta); applyErr != nil {
					return diag.FromErr(applyErr)
				}
			}
			// retry count is not 0, so we need to leverage exponential backoff and multiple retries
			exponentialBackoffConfig := backoff.NewExponentialBackOff()
			exponentialBackoffConfig.InitialInterval = 3 * time.Second
			exponentialBackoffConfig.MaxInterval = 30 * time.Second

			retryConfig := backoff.WithMaxRetries(exponentialBackoffConfig, kubectlApplyRetryCount)
			retryErr := backoff.Retry(func() error {
				err := resourceKubectlManifestApply(ctx, d, meta)
				if err != nil {
					log.Printf("[ERROR] creating manifest failed: %+v", err)
				}
				return err
			}, retryConfig)

			if retryErr != nil {
				return diag.FromErr(retryErr)
			}
			return nil
		},
		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			if err := resourceKubectlManifestRead(ctx, d, meta); err != nil {
				return diag.FromErr(err)
			}
			return nil
		},
		DeleteContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			if err := resourceKubectlManifestDelete(ctx, d, meta); err != nil {
				return diag.FromErr(err)
			}

			return nil
		},
		UpdateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			exponentialBackoffConfig := backoff.NewExponentialBackOff()
			exponentialBackoffConfig.InitialInterval = 3 * time.Second
			exponentialBackoffConfig.MaxInterval = 30 * time.Second

			if kubectlApplyRetryCount > 0 {
				retryConfig := backoff.WithMaxRetries(exponentialBackoffConfig, kubectlApplyRetryCount)
				retryErr := backoff.Retry(func() error {
					err := resourceKubectlManifestApply(ctx, d, meta)
					if err != nil {
						log.Printf("[ERROR] updating manifest failed: %+v", err)
					}
					return err
				}, retryConfig)

				if retryErr != nil {
					return diag.FromErr(retryErr)
				}

				return nil
			} else {
				if applyErr := resourceKubectlManifestApply(ctx, d, meta); applyErr != nil {
					return diag.FromErr(applyErr)
				}

				return nil
			}
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "//")
				if len(idParts) != 3 && len(idParts) != 4 {
					return []*schema.ResourceData{}, fmt.Errorf("expected ID in format apiVersion//kind//name//namespace, received: %s", d.Id())
				}

				apiVersion := idParts[0]
				kind := idParts[1]
				name := idParts[2]

				var yamlString = ""
				if len(idParts) == 4 {
					yamlString = fmt.Sprintf(`
apiVersion: %s
kind: %s
metadata:
  namespace: %s
  name: %s
`, apiVersion, kind, idParts[3], name)
				} else {
					yamlString = fmt.Sprintf(`
apiVersion: %s
kind: %s
metadata:
  name: %s
`, apiVersion, kind, name)
				}

				manifest, err := yaml.ParseYAML(yamlString)
				if err != nil {
					return nil, err
				}
				restClient := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))

				if restClient.Error != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to create kubernetes rest client for import of resource: %s %s %s %+v %s %s", apiVersion, kind, name, restClient.Error, yamlString, manifest.Raw)
				}

				// Get the resource from Kubernetes
				metaObjLiveRaw, err := restClient.ResourceInterface.Get(ctx, manifest.GetName(), meta_v1.GetOptions{})
				if err != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to get resource %s %s %s from kubernetes: %+v", apiVersion, kind, name, err)
				}

				if metaObjLiveRaw.GetUID() == "" {
					return []*schema.ResourceData{}, fmt.Errorf("failed to parse item and get UUID: %+v", metaObjLiveRaw)
				}

				metaObjLive := yaml.NewFromUnstructured(metaObjLiveRaw)

				// Capture the UID from the cluster at the current time
				_ = d.Set("uid", metaObjLive.GetUID())
				_ = d.Set("live_uid", metaObjLive.GetUID())

				liveManifestFingerprint := getLiveManifestFingerprint(d, metaObjLive, metaObjLive)
				_ = d.Set("yaml_incluster", liveManifestFingerprint)
				_ = d.Set("live_manifest_incluster", liveManifestFingerprint)

				// set fields captured normally during creation/updates
				d.SetId(metaObjLive.GetSelfLink())
				_ = d.Set("api_version", metaObjLive.GetAPIVersion())
				_ = d.Set("kind", metaObjLive.GetKind())
				_ = d.Set("namespace", metaObjLive.GetNamespace())
				_ = d.Set("name", metaObjLive.GetName())
				_ = d.Set("force_new", false)
				_ = d.Set("server_side_apply", false)
				_ = d.Set("apply_only", false)

				// clear out fields user can't set to try and get parity with yaml_body
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Raw.Object, "metadata", "creationTimestamp")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Raw.Object, "metadata", "resourceVersion")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Raw.Object, "metadata", "selfLink")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Raw.Object, "metadata", "uid")
				meta_v1_unstruct.RemoveNestedField(metaObjLive.Raw.Object, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")

				if len(metaObjLive.Raw.GetAnnotations()) == 0 {
					meta_v1_unstruct.RemoveNestedField(metaObjLive.Raw.Object, "metadata", "annotations")
				}

				yamlParsed, err := metaObjLive.AsYAML()
				if err != nil {
					return []*schema.ResourceData{}, fmt.Errorf("failed to convert manifest to yaml: %+v", err)
				}

				_ = d.Set("yaml_body", yamlParsed)
				_ = d.Set("yaml_body_parsed", yamlParsed)

				return []*schema.ResourceData{d}, nil
			},
		},
		CustomizeDiff: func(context context.Context, d *schema.ResourceDiff, meta interface{}) error {

			// trigger a recreation if the yaml-body has any pending changes
			if d.Get("force_new").(bool) {
				_ = d.ForceNew("yaml_body")
			}

			if !d.NewValueKnown("yaml_body") {
				log.Printf("[TRACE] yaml_body value interpolated, skipping customized diff")
				d.SetNewComputed("yaml_body_parsed")
				d.SetNewComputed("yaml_incluster")
				return nil
			}

			parsedYaml, err := yaml.ParseYAML(d.Get("yaml_body").(string))
			if err != nil {
				return err
			}

			if overrideNamespace, ok := d.GetOk("override_namespace"); ok {
				parsedYaml.SetNamespace(overrideNamespace.(string))
			}

			// set calculated fields based on parsed yaml values
			_ = d.SetNew("api_version", parsedYaml.GetAPIVersion())
			_ = d.SetNew("kind", parsedYaml.GetKind())
			_ = d.SetNew("namespace", parsedYaml.GetNamespace())
			_ = d.SetNew("name", parsedYaml.GetName())

			// set the yaml_body_parsed field to provided value and obfuscate the yaml_body values manually
			// this allows us to show a nice diff to the users with specific fields obfuscated, whilst storing the
			// real value to apply in yaml_body
			obfuscatedYaml, _ := yaml.ParseYAML(d.Get("yaml_body").(string))
			if obfuscatedYaml.Raw.Object == nil {
				obfuscatedYaml.Raw.Object = make(map[string]interface{})
			}

			if overrideNamespace, ok := d.GetOk("override_namespace"); ok {
				obfuscatedYaml.SetNamespace(overrideNamespace.(string))
			}

			var sensitiveFields []string
			sensitiveFieldsRaw, hasSensitiveFields := d.GetOk("sensitive_fields")
			if hasSensitiveFields {
				sensitiveFields = expandStringList(sensitiveFieldsRaw.([]interface{}))
			} else if parsedYaml.GetKind() == "Secret" && parsedYaml.GetAPIVersion() == "v1" {
				sensitiveFields = []string{"data", "stringData"}
			}

			for _, s := range sensitiveFields {
				fields := strings.Split(s, ".")
				_, fieldExists, err := meta_v1_unstruct.NestedFieldNoCopy(obfuscatedYaml.Raw.Object, fields...)
				if fieldExists {
					err = meta_v1_unstruct.SetNestedField(obfuscatedYaml.Raw.Object, "(sensitive value)", fields...)
					if err != nil {
						return fmt.Errorf("failed to obfuscate sensitive field '%s': %+v\nNote: only map values are supported!", s, err)
					}
				} else {
					log.Printf("[TRACE] sensitive field %s skipped does not exist", s)
				}
			}

			obfuscatedYamlBytes, obfuscatedYamlBytesErr := yamlWriter.Marshal(obfuscatedYaml.Raw.Object)
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

			if UID != createdAtUID {
				log.Printf("[TRACE] DETECTED %s vs %s", UID, createdAtUID)
				_ = d.SetNewComputed("uid")
				return nil
			}

			// Check that the fields specified in our YAML for diff against cluster representation
			stateYaml := d.Get("yaml_incluster").(string)
			liveStateYaml := d.Get("live_manifest_incluster").(string)
			if stateYaml != liveStateYaml {
				log.Printf("[TRACE] DETECTED YAML STATE DIFFERENCE %s vs %s", stateYaml, liveStateYaml)
				// disabled due to a bug in go-diff library. See https://github.com/alekc/terraform-provider-kubectl/issues/181
				//dmp := diffmatchpatch.New()
				//patches := dmp.PatchMake(stateYaml, liveStateYaml)
				//patchText := dmp.PatchToText(patches)
				//log.Printf("[DEBUG] DETECTED YAML INCLUSTER STATE DIFFERENCE. Patch diff: %s", patchText)
				_ = d.SetNewComputed("yaml_incluster")
			}

			return nil
		},
		Schema:        kubectlManifestSchema,
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceKubectlManifestV0().CoreConfigSchema().ImpliedType(),
				Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
					rawState["yaml_incluster"] = getFingerprint(rawState["yaml_incluster"].(string))
					rawState["live_manifest_incluster"] = getFingerprint(rawState["live_manifest_incluster"].(string))
					return rawState, nil
				},
			},
		},
	}
}

func resourceKubectlManifestV0() *schema.Resource {
	return &schema.Resource{
		Schema: kubectlManifestSchema,
	}
}

var (
	kubectlManifestSchema = map[string]*schema.Schema{
		"uid": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"live_uid": {
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
		"override_namespace": {
			Type:        schema.TypeString,
			Description: "Override the namespace to apply the kubernetes resource to",
			Optional:    true,
		},
		"yaml_body": {
			Type:      schema.TypeString,
			Required:  true,
			Sensitive: true,
		},
		"yaml_body_parsed": {
			Type:        schema.TypeString,
			Description: "Yaml body that is being applied, with sensitive values obfuscated",
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
		"server_side_apply": {
			Type:        schema.TypeBool,
			Description: "Default to client-side-apply. Setting to true will use server-side apply.",
			Optional:    true,
			Default:     false,
		},
		"field_manager": {
			Type:        schema.TypeString,
			Description: "Override the default field manager name. This is only relevant when using server-side apply.",
			Optional:    true,
			Default:     "kubectl",
		},
		"force_conflicts": {
			Type:        schema.TypeBool,
			Description: "Default false.",
			Optional:    true,
			Default:     false,
		},
		"apply_only": {
			Type:        schema.TypeBool,
			Description: "Apply only. In other words, it does not delete resource in any case.",
			Optional:    true,
			Default:     false,
		},
		"ignore_fields": {
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "List of yaml keys to ignore changes to. Set these for fields set by Operators or other processes in kubernetes and as such you don't want to update.",
			Optional:    true,
		},
		"wait": {
			Type:        schema.TypeBool,
			Description: "Default to false (not waiting). Set this flag to wait or not for any deleted resources to be gone. This waits for finalizers.",
			Optional:    true,
		},
		"wait_for_rollout": {
			Type:        schema.TypeBool,
			Description: "Default to true (waiting). Set this flag to wait or not for Deployments and APIService to complete rollout",
			Optional:    true,
			Default:     true,
		},
		"validate_schema": {
			Type:        schema.TypeBool,
			Description: "Default to true (validate). Set this flag to not validate the yaml schema before applying.",
			Optional:    true,
			Default:     true,
		},
		"wait_for": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "If set, will wait until either all of conditions are satisfied, or until timeout is reached",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"condition": {
						Type:        schema.TypeList,
						MinItems:    0,
						Description: "Condition criteria for a Status Condition",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"type": {
									Type:        schema.TypeString,
									Description: "Type as expected from the resulting Condition object",
									Required:    true,
								},
								"status": {
									Type:        schema.TypeString,
									Description: "Status to wait for in the resulting Condition object",
									Required:    true,
								},
							},
						},
					},
					"field": {
						Type:        schema.TypeList,
						MinItems:    0,
						Description: "Condition criteria for a field",
						Optional:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"key": {
									Type:        schema.TypeString,
									Description: "Key which should be matched from resulting object",
									Required:    true,
								},
								"value": {
									Type:        schema.TypeString,
									Description: "Value to wait for",
									Required:    true,
								},
								"value_type": {
									Type:             schema.TypeString,
									Description:      "Value type. Can be either a `eq` (equivalent) or `regex`",
									ValidateDiagFunc: validate2.ToDiagFunc(validate2.StringInSlice([]string{"eq", "regex"}, false)),
									Default:          "eq",
									Optional:         true,
								},
							},
						},
					},
				},
			},
		},
		"delete_cascade": {
			Type:             schema.TypeString,
			Description:      "Cascade mode for delete operations, explicitly setting this to Background to match kubectl is recommended. Default is Background unless wait has been set when it will be Foreground.",
			Optional:         true,
			ValidateDiagFunc: validate2.ToDiagFunc(validate2.StringInSlice([]string{string(meta_v1.DeletePropagationBackground), string(meta_v1.DeletePropagationForeground)}, false)),
		},
	}
)

// newApplyOptions defines flags and other configuration parameters for the `apply` command
func newApplyOptions(yamlBody string) *apply.ApplyOptions {
	applyOptions := &apply.ApplyOptions{
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme),

		IOStreams: genericiooptions.IOStreams{
			In:     strings.NewReader(yamlBody),
			Out:    log.Writer(),
			ErrOut: log.Writer(),
		},

		Overwrite:    true,
		OpenAPIPatch: true,
		Recorder:     genericclioptions.NoopRecorder{},

		VisitedUids:       sets.New[apiMachineryTypes.UID](),
		VisitedNamespaces: sets.New[string](),
	}
	return applyOptions
}
func resourceKubectlManifestApply(ctx context.Context, d *schema.ResourceData, meta interface{}) error {
	yamlBody := d.Get("yaml_body").(string)

	// convert hcl into an unstructured object
	manifest, err := yaml.ParseYAML(yamlBody)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes resource: %+v", err)
	}

	if overrideNamespace, ok := d.GetOk("override_namespace"); ok {
		manifest.SetNamespace(overrideNamespace.(string))
	}

	log.Printf("[DEBUG] %v apply kubernetes resource:\n%s", manifest, yamlBody)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	restClient := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))
	if restClient.Error != nil {
		return fmt.Errorf("%v failed to create kubernetes rest client for update of resource: %+v", manifest, restClient.Error)
	}

	// Update the resource in Kubernetes, using a temp file
	yamlBody, err = manifest.AsYAML()
	if err != nil {
		return fmt.Errorf("%v failed to convert to yaml: %+v", manifest, err)
	}

	tmpfile, _ := os.CreateTemp("", "*kubectl_manifest.yaml")
	_, _ = tmpfile.Write([]byte(yamlBody))
	_ = tmpfile.Close()

	applyOptions := newApplyOptions(yamlBody)
	applyOptions.Builder = k8sresource.NewBuilder(k8sresource.RESTClientGetter(meta.(*KubeProvider)))
	applyOptions.DeleteOptions = &k8sdelete.DeleteOptions{
		FilenameOptions: k8sresource.FilenameOptions{
			Filenames: []string{tmpfile.Name()},
		},
	}
	applyOptions.ToPrinter = func(string) (printers.ResourcePrinter, error) {
		return printers.NewDiscardingPrinter(), nil
	}
	if !d.Get("validate_schema").(bool) {
		applyOptions.Validator = validation.NullSchema{}
	}
	if d.Get("server_side_apply").(bool) {
		applyOptions.ServerSideApply = true
		applyOptions.FieldManager = d.Get("field_manager").(string)
	}
	if d.Get("force_conflicts").(bool) {
		applyOptions.ForceConflicts = true
	}
	if manifest.HasNamespace() {
		applyOptions.Namespace = manifest.GetNamespace()
	}

	log.Printf("[INFO] %s perform apply of manifest", manifest)

	err = applyOptions.Run()
	_ = os.Remove(tmpfile.Name())
	if err != nil {
		return fmt.Errorf("%v failed to run apply: %+v", manifest, err)
	}

	log.Printf("[INFO] %v manifest applied, fetch resource from kubernetes", manifest)

	// get the resource from Kubernetes
	rawResponse, err := restClient.ResourceInterface.Get(ctx, manifest.GetName(), meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("%v failed to fetch resource from kubernetes: %+v", manifest, err)
	}
	// set a wrapper from unstructured raw manifest
	response := yaml.NewFromUnstructured(rawResponse)

	d.SetId(response.GetSelfLink())
	log.Printf("[DEBUG] %v fetched successfully, set id to: %v", manifest, d.Id())

	// Capture the UID at time of update
	// this allows us to diff these against the actual values
	// read in by the 'resourceKubectlManifestRead'
	_ = d.Set("uid", response.GetUID())
	_ = d.Set("live_uid", response.GetUID())

	liveManifestFingerprint := getLiveManifestFingerprint(d, manifest, response)
	_ = d.Set("yaml_incluster", liveManifestFingerprint)
	_ = d.Set("live_manifest_incluster", liveManifestFingerprint)

	if d.Get("wait_for_rollout").(bool) {
		timeout := d.Timeout(schema.TimeoutCreate)

		switch {
		case manifest.GetKind() == "Deployment":
			log.Printf("[INFO] %v waiting for Deployment rollout for %vmin", manifest, timeout.Minutes())
			err = waitForDeploymentRollout(ctx, meta.(*KubeProvider), manifest.GetNamespace(), manifest.GetName(), timeout)
			if err != nil {
				return err
			}
		case manifest.GetKind() == "DaemonSet":
			log.Printf("[INFO] %v waiting for DaemonSet rollout for %vmin", manifest, timeout.Minutes())
			err = waitForDaemonSetRollout(ctx, meta.(*KubeProvider), manifest.GetNamespace(), manifest.GetName(), timeout)
			if err != nil {
				return err
			}
		case manifest.GetKind() == "StatefulSet":
			log.Printf("[INFO] %v waiting for v rollout for %vmin", manifest, timeout.Minutes())
			err = waitForStatefulSetRollout(ctx, meta.(*KubeProvider), manifest.GetNamespace(), manifest.GetName(), timeout)
			if err != nil {
				return err
			}
		case manifest.GetKind() == "APIService" && manifest.GetAPIVersion() == "apiregistration.k8s.io/v1":
			log.Printf("[INFO] %v waiting for APIService for %vmin", manifest, timeout.Minutes())
			err = waitForApiService(ctx, meta.(*KubeProvider), manifest.GetName(), timeout)
			if err != nil {
				return err
			}
		}
	}

	if v, ok := d.GetOk("wait_for"); ok {
		timeout := d.Timeout(schema.TimeoutCreate)

		waitFor := types.WaitFor{}
		if err := mapstructure.Decode((v.([]interface{}))[0], &waitFor); err != nil {
			return fmt.Errorf("cannot decode wait for conditions %v", err)
		}
		if len(waitFor.Field) == 0 && len(waitFor.Condition) == 0 {
			return fmt.Errorf("at least one of `field` or `condition` must be provided in `wait_for` block")
		}

		log.Printf("[INFO] %v waiting for wait conditions for %vmin", manifest, timeout.Minutes())
		err = waitForConditions(ctx, restClient, waitFor.Field, waitFor.Condition, manifest.GetName(), timeout)
		if err != nil {
			return err
		}
	}

	// So far we have set (live_)uid and (live_)yaml_incluster.
	// Perform the full read of the object
	return resourceKubectlManifestReadUsingClient(ctx, d, meta, restClient.ResourceInterface, manifest)
}

func resourceKubectlManifestRead(ctx context.Context, d *schema.ResourceData, meta interface{}) error {
	yamlBody := d.Get("yaml_body").(string)
	manifest, err := yaml.ParseYAML(yamlBody)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes resource: %+v", err)
	}

	if overrideNamespace, ok := d.GetOk("override_namespace"); ok {
		manifest.SetNamespace(overrideNamespace.(string))
	}

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	restClient := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))
	if restClient.Status == RestClientInvalidTypeError {
		log.Printf("[WARN] kubernetes resource (%s) has an invalid type, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if restClient.Error != nil {
		return fmt.Errorf("failed to create kubernetes rest client for read of resource: %+v", restClient.Error)
	}

	return resourceKubectlManifestReadUsingClient(ctx, d, meta, restClient.ResourceInterface, manifest)
}

// resourceKubectlManifestReadUsingClient reads the object data from the cluster based on it's UID
// and sets live_uid and live_manifest_incluster to the latest values
func resourceKubectlManifestReadUsingClient(ctx context.Context, d *schema.ResourceData, meta interface{}, client dynamic.ResourceInterface, manifest *yaml.Manifest) error {

	log.Printf("[DEBUG] %v fetch from kubernetes", manifest)

	// Get the resource from Kubernetes
	metaObjLiveRaw, err := client.Get(ctx, manifest.GetName(), meta_v1.GetOptions{})
	if err != nil {
		if errors.IsGone(err) || errors.IsNotFound(err) {
			log.Printf("[WARN] kubernetes resource (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("%v failed to get resource from kubernetes: %+v", manifest, err)
	}

	if metaObjLiveRaw.GetUID() == "" {
		return fmt.Errorf("%v failed to parse item and get UUID: %+v", manifest, metaObjLiveRaw)
	}

	metaObjLive := yaml.NewFromUnstructured(metaObjLiveRaw)

	// Capture the UID from the cluster at the current time
	_ = d.Set("live_uid", metaObjLive.GetUID())

	liveManifestFingerprint := getLiveManifestFingerprint(d, manifest, metaObjLive)
	_ = d.Set("live_manifest_incluster", liveManifestFingerprint)

	return nil
}

func resourceKubectlManifestDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) error {
	if d.Get("apply_only").(bool) {
		return nil
	}
	yamlBody := d.Get("yaml_body").(string)
	manifest, err := yaml.ParseYAML(yamlBody)
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes resource: %+v", err)
	}

	if overrideNamespace, ok := d.GetOk("override_namespace"); ok {
		manifest.SetNamespace(overrideNamespace.(string))
	}

	log.Printf("[DEBUG] %v delete kubernetes resource:\n%s", manifest, yamlBody)

	restClient := getRestClientFromUnstructured(manifest, meta.(*KubeProvider))
	if restClient.Error != nil {
		return fmt.Errorf("%v failed to create kubernetes rest client for delete of resource: %+v", manifest, restClient.Error)
	}

	log.Printf("[INFO] %s perform delete of manifest", manifest)

	wait := d.Get("wait").(bool)

	var propagationPolicy meta_v1.DeletionPropagation
	cascadeInput := d.Get("delete_cascade").(string)
	if len(cascadeInput) > 0 {
		propagationPolicy = meta_v1.DeletionPropagation(cascadeInput)
	} else if wait {
		propagationPolicy = meta_v1.DeletePropagationForeground
	} else {
		propagationPolicy = meta_v1.DeletePropagationBackground
	}

	err = restClient.ResourceInterface.Delete(ctx, manifest.GetName(), meta_v1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	resourceGone := errors.IsGone(err) || errors.IsNotFound(err)
	if err != nil && !resourceGone {
		return fmt.Errorf("%v failed to delete kubernetes resource: %+v", manifest, err)
	}

	// The rest client doesn't wait for the delete so we need custom logic
	if wait && !resourceGone {
		log.Printf("[INFO] %s waiting for delete of manifest to complete", manifest)

		timeout := d.Timeout(schema.TimeoutDelete)

		err = waitForDelete(ctx, restClient, manifest.GetName(), timeout)
		if err != nil {
			return err
		}
	}

	// Success remove it from state
	d.SetId("")
	return nil
}

type RestClientStatus int

const (
	RestClientOk = iota
	RestClientGenericError
	RestClientInvalidTypeError
)

type RestClientResult struct {
	ResourceInterface dynamic.ResourceInterface
	Error             error
	Status            RestClientStatus
}

func RestClientResultSuccess(resourceInterface dynamic.ResourceInterface) *RestClientResult {
	return &RestClientResult{
		ResourceInterface: resourceInterface,
		Error:             nil,
		Status:            RestClientOk,
	}
}

func RestClientResultFromErr(err error) *RestClientResult {
	return &RestClientResult{
		ResourceInterface: nil,
		Error:             err,
		Status:            RestClientGenericError,
	}
}

func RestClientResultFromInvalidTypeErr(err error) *RestClientResult {
	return &RestClientResult{
		ResourceInterface: nil,
		Error:             err,
		Status:            RestClientInvalidTypeError,
	}
}

// getRestClientFromUnstructured creates a dynamic k8s client based on the provided manifest
func getRestClientFromUnstructured(manifest *yaml.Manifest, provider *KubeProvider) *RestClientResult {

	doGetRestClientFromUnstructured := func(manifest *yaml.Manifest, provider *KubeProvider) *RestClientResult {
		// Use the k8s Discovery service to find all valid APIs for this cluster
		discoveryClient, _ := provider.ToDiscoveryClient()
		var resources []*meta_v1.APIResourceList
		var err error
		_, resources, err = discoveryClient.ServerGroupsAndResources()

		// There is a partial failure mode here where not all groups are returned `GroupDiscoveryFailedError`
		// we'll try and continue in this condition as it's likely something we don't need
		// and if it is the `checkAPIResourceIsPresent` check will fail and stop the process
		if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
			return RestClientResultFromErr(err)
		}

		// Validate that the APIVersion provided in the YAML is valid for this cluster
		apiResource, exists := checkAPIResourceIsPresent(resources, *manifest.Raw)
		if !exists {
			// api not found, invalidate the cache and try again
			// this handles the case when a CRD is being created by another kubectl_manifest resource run
			discoveryClient.Invalidate()
			_, resources, err = discoveryClient.ServerGroupsAndResources()

			if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
				return RestClientResultFromErr(err)
			}

			// check for resource again
			apiResource, exists = checkAPIResourceIsPresent(resources, *manifest.Raw)
			if !exists {
				return RestClientResultFromInvalidTypeErr(fmt.Errorf("resource [%s/%s] isn't valid for cluster, check the APIVersion and Kind fields are valid", manifest.Raw.GroupVersionKind().GroupVersion().String(), manifest.GetKind()))
			}
		}

		resourceStruct := k8sschema.GroupVersionResource{
			Group:    apiResource.Group,
			Version:  apiResource.Version,
			Resource: apiResource.Name,
		}
		// For core services (ServiceAccount, Service etc) the group is incorrectly parsed.
		// "v1" should be empty group and "v1" for version
		if resourceStruct.Group == "v1" && resourceStruct.Version == "" {
			resourceStruct.Group = ""
			resourceStruct.Version = "v1"
		}
		// get dynamic client based on the found resource struct
		client := dynamic.NewForConfigOrDie(&provider.RestConfig).Resource(resourceStruct)

		// if the resource is namespaced and doesn't have a namespace defined, set it to default
		if apiResource.Namespaced {
			if !manifest.HasNamespace() {
				manifest.SetNamespace("default")
			}
			return RestClientResultSuccess(client.Namespace(manifest.GetNamespace()))
		}

		return RestClientResultSuccess(client)
	}

	discoveryWithTimeout := func(manifest *yaml.Manifest, provider *KubeProvider) <-chan *RestClientResult {
		ch := make(chan *RestClientResult)
		go func() {
			ch <- doGetRestClientFromUnstructured(manifest, provider)
		}()
		return ch
	}

	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()
	select {
	case res := <-discoveryWithTimeout(manifest, provider):
		return res
	case <-timeout.C:
		log.Printf("[ERROR] %v timed out fetching resources from discovery client", manifest)
		return RestClientResultFromErr(fmt.Errorf("%v timed out fetching resources from discovery client", manifest))
	}
}

// checkAPIResourceIsPresent Loops through a list of available APIResources and
// checks there is a resource for the APIVersion and Kind defined in the 'resource'
// if found it returns true and the APIResource which matched
func checkAPIResourceIsPresent(available []*meta_v1.APIResourceList, resource meta_v1_unstruct.Unstructured) (*meta_v1.APIResource, bool) {
	resourceGroupVersionKind := resource.GroupVersionKind()
	for _, rList := range available {
		if rList == nil {
			continue
		}
		group := rList.GroupVersion
		for _, r := range rList.APIResources {
			if group == resourceGroupVersionKind.GroupVersion().String() && r.Kind == resource.GetKind() {
				r.Group = resourceGroupVersionKind.Group
				r.Version = resourceGroupVersionKind.Version
				r.Kind = resourceGroupVersionKind.Kind
				return &r, true
			}
		}
	}
	log.Printf("[ERROR] Could not find a valid ApiResource for this manifest %s/%s/%s", resourceGroupVersionKind.Group, resourceGroupVersionKind.Version, resourceGroupVersionKind.Kind)
	return nil, false
}

func waitForDelete(ctx context.Context, restClient *RestClientResult, name string, timeout time.Duration) error {
	timeoutSeconds := int64(timeout.Seconds())

	rawResponse, err := restClient.ResourceInterface.Get(ctx, name, meta_v1.GetOptions{})
	resourceGone := errors.IsGone(err) || errors.IsNotFound(err)
	if err != nil && !resourceGone {
		return err
	}

	if !resourceGone {
		resourceVersion, _, err := unstructured.NestedString(rawResponse.Object, "metadata", "resourceVersion")
		if err != nil {
			return err
		}

		watcher, err := restClient.ResourceInterface.Watch(
			ctx,
			meta_v1.ListOptions{
				Watch:           true,
				TimeoutSeconds:  &timeoutSeconds,
				FieldSelector:   fields.OneTermEqualSelector("metadata.name", name).String(),
				ResourceVersion: resourceVersion,
			})
		if err != nil {
			return err
		}

		defer watcher.Stop()

		deleted := false
		for !deleted {
			select {
			case event := <-watcher.ResultChan():
				if event.Type == watch.Deleted {
					deleted = true
				}

			case <-ctx.Done():
				return fmt.Errorf("%s failed to delete resource", name)
			}
		}
	}

	return nil
}

func waitForDeploymentRollout(ctx context.Context, provider *KubeProvider, ns string, name string, timeout time.Duration) error {
	// Borrowed from: https://github.com/kubernetes/kubectl/blob/c4be63c54b7188502c1a63bb884a0b05fac51ebd/pkg/polymorphichelpers/rollout_status.go#L59

	timeoutSeconds := int64(timeout.Seconds())

	watcher, err := provider.MainClientset.AppsV1().Deployments(ns).Watch(ctx, meta_v1.ListOptions{Watch: true, TimeoutSeconds: &timeoutSeconds, FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String()})
	if err != nil {
		return err
	}

	defer watcher.Stop()

	done := false
	for !done {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Modified {
				deployment, ok := event.Object.(*apps_v1.Deployment)
				if !ok {
					return fmt.Errorf("%s could not cast to Deployment", name)
				}

				if deployment.Generation <= deployment.Status.ObservedGeneration {
					condition := getDeploymentCondition(deployment.Status, apps_v1.DeploymentProgressing)
					if condition != nil && condition.Reason == TimedOutReason {
						continue
					}

					if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
						continue
					}

					if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
						continue
					}

					if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
						continue
					}

					done = true
				}
			}

		case <-ctx.Done():
			return fmt.Errorf("%s failed to rollout Deployment", name)
		}
	}

	return nil
}

func getDeploymentCondition(status apps_v1.DeploymentStatus, condType apps_v1.DeploymentConditionType) *apps_v1.DeploymentCondition {
	// Borrowed from: https://github.com/kubernetes/kubectl/blob/c4be63c54b7188502c1a63bb884a0b05fac51ebd/pkg/util/deployment/deployment.go#L60
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func waitForDaemonSetRollout(ctx context.Context, provider *KubeProvider, ns string, name string, timeout time.Duration) error {
	// Borrowed from: https://github.com/kubernetes/kubectl/blob/c4be63c54b7188502c1a63bb884a0b05fac51ebd/pkg/polymorphichelpers/rollout_status.go#L95

	timeoutSeconds := int64(timeout.Seconds())

	watcher, err := provider.MainClientset.AppsV1().DaemonSets(ns).Watch(ctx, meta_v1.ListOptions{Watch: true, TimeoutSeconds: &timeoutSeconds, FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String()})
	if err != nil {
		return err
	}

	defer watcher.Stop()

	done := false
	for !done {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Modified {
				daemon, ok := event.Object.(*apps_v1.DaemonSet)
				if !ok {
					return fmt.Errorf("%s could not cast to DaemonSet", name)
				}

				if daemon.Spec.UpdateStrategy.Type != apps_v1.RollingUpdateDaemonSetStrategyType {
					done = true
					continue
				}

				if daemon.Generation <= daemon.Status.ObservedGeneration {
					if daemon.Status.UpdatedNumberScheduled < daemon.Status.DesiredNumberScheduled {
						continue
					}

					if daemon.Status.NumberAvailable < daemon.Status.DesiredNumberScheduled {
						continue
					}

					done = true
				}
			}

		case <-ctx.Done():
			return fmt.Errorf("%s failed to rollout DaemonSet", name)
		}
	}

	return nil
}

func waitForStatefulSetRollout(ctx context.Context, provider *KubeProvider, ns string, name string, timeout time.Duration) error {
	// Borrowed from: https://github.com/kubernetes/kubectl/blob/c4be63c54b7188502c1a63bb884a0b05fac51ebd/pkg/polymorphichelpers/rollout_status.go#L120

	timeoutSeconds := int64(timeout.Seconds())

	watcher, err := provider.MainClientset.AppsV1().StatefulSets(ns).Watch(ctx, meta_v1.ListOptions{Watch: true, TimeoutSeconds: &timeoutSeconds, FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String()})
	if err != nil {
		return err
	}

	defer watcher.Stop()

	done := false
	for !done {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Modified {
				sts, ok := event.Object.(*apps_v1.StatefulSet)
				if !ok {
					return fmt.Errorf("%s could not cast to StatefulSet", name)
				}

				if sts.Spec.UpdateStrategy.Type != apps_v1.RollingUpdateStatefulSetStrategyType {
					done = true
					continue
				}

				if sts.Status.ObservedGeneration == 0 || sts.Generation > sts.Status.ObservedGeneration {
					continue
				}

				if sts.Spec.Replicas != nil && sts.Status.ReadyReplicas < *sts.Spec.Replicas {
					continue
				}

				if sts.Spec.UpdateStrategy.Type == apps_v1.RollingUpdateStatefulSetStrategyType && sts.Spec.UpdateStrategy.RollingUpdate != nil {
					if sts.Spec.Replicas != nil && sts.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
						if sts.Status.UpdatedReplicas < (*sts.Spec.Replicas - *sts.Spec.UpdateStrategy.RollingUpdate.Partition) {
							continue
						}
					}

					done = true
					continue
				}

				if sts.Status.UpdateRevision != sts.Status.CurrentRevision {
					continue
				}

				done = true
			}

		case <-ctx.Done():
			return fmt.Errorf("%s failed to rollout StatefulSet", name)
		}
	}

	return nil
}

func waitForApiService(ctx context.Context, provider *KubeProvider, name string, timeout time.Duration) error {
	timeoutSeconds := int64(timeout.Seconds())

	watcher, err := provider.AggregatorClientset.ApiregistrationV1().APIServices().Watch(ctx, meta_v1.ListOptions{Watch: true, TimeoutSeconds: &timeoutSeconds, FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String()})
	if err != nil {
		return err
	}

	defer watcher.Stop()

	done := false
	for !done {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Modified {
				apiService, ok := event.Object.(*apiregistration.APIService)
				if !ok {
					return fmt.Errorf("%s could not cast to APIService", name)
				}

				for i := range apiService.Status.Conditions {
					if apiService.Status.Conditions[i].Type == apiregistration.Available {
						done = true
						continue
					}
				}
			}

		case <-ctx.Done():
			return fmt.Errorf("%s failed to wait for APIService", name)
		}
	}

	return nil
}

func waitForConditions(ctx context.Context, restClient *RestClientResult, waitFields []types.WaitForField, waitConditions []types.WaitForStatusCondition, name string, timeout time.Duration) error {
	timeoutSeconds := int64(timeout.Seconds())

	watcher, err := restClient.ResourceInterface.Watch(
		ctx,
		meta_v1.ListOptions{
			Watch:          true,
			TimeoutSeconds: &timeoutSeconds,
			FieldSelector:  fields.OneTermEqualSelector("metadata.name", name).String(),
		},
	)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	done := false
	for !done {
		select {
		case event := <-watcher.ResultChan():
			log.Printf("[TRACE] Received event type %s for %s", event.Type, name)
			if event.Type == watch.Modified || event.Type == watch.Added {
				rawResponse, ok := event.Object.(*meta_v1_unstruct.Unstructured)
				if !ok {
					return fmt.Errorf("%s could not cast resource to unstructured", name)
				}

				totalConditions := len(waitConditions) + len(waitFields)
				totalMatches := 0

				yamlJson, err := rawResponse.MarshalJSON()
				if err != nil {
					return err
				}

				gq := gojsonq.New().FromString(string(yamlJson))

				for _, c := range waitConditions {
					// Find the conditions by status and type
					count := gq.Reset().From("status.conditions").
						Where("type", "=", c.Type).
						Where("status", "=", c.Status).Count()
					if count == 0 {
						log.Printf("[TRACE] Condition %s with status %s not found in %s", c.Type, c.Status, name)
						continue
					}
					log.Printf("[TRACE] Condition %s with status %s found in %s", c.Type, c.Status, name)
					totalMatches++
				}

				for _, c := range waitFields {
					// Find the key
					v := gq.Reset().Find(c.Key)
					if v == nil {
						log.Printf("[TRACE] Key %s not found in %s", c.Key, name)
						continue
					}

					// For the sake of comparison we will convert everything to a string
					stringVal := fmt.Sprintf("%v", v)
					switch c.ValueType {
					case "regex":
						matched, err := regexp.Match(c.Value, []byte(stringVal))
						if err != nil {
							return err
						}

						if !matched {
							log.Printf("[TRACE] Value %s does not match regex %s in %s (key %s)", stringVal, c.Value, name, c.Key)
							continue
						}

						log.Printf("[TRACE] Value %s matches regex %s in %s (key %s)", stringVal, c.Value, name, c.Key)
						totalMatches++

					case "eq", "":
						if stringVal != c.Value {
							log.Printf("[TRACE] Value %s does not match %s in %s (key %s)", stringVal, c.Value, name, c.Key)
							continue
						}
						log.Printf("[TRACE] Value %s matches %s in %s (key %s)", stringVal, c.Value, name, c.Key)
						totalMatches++
					}
				}
				if totalMatches == totalConditions {
					log.Printf("[TRACE] All conditions met for %s", name)
					done = true
					continue
				}
				log.Printf("[TRACE] %d/%d conditions met for %s. Waiting for next ", totalMatches, totalConditions, name)
			}

		case <-ctx.Done():
			return fmt.Errorf("%s failed to wait for resource", name)
		}
	}

	return nil
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

func getLiveManifestFingerprint(d *schema.ResourceData, userProvided *yaml.Manifest, liveManifest *yaml.Manifest) string {
	var ignoreFields []string = nil
	ignoreFieldsRaw, hasIgnoreFields := d.GetOk("ignore_fields")
	if hasIgnoreFields {
		ignoreFields = expandStringList(ignoreFieldsRaw.([]interface{}))
	}
	return getLiveManifestFields_WithIgnoredFields(ignoreFields, userProvided, liveManifest)
}

func getFingerprint(s string) string {
	fingerprint := sha256.New()
	fingerprint.Write([]byte(s))
	return fmt.Sprintf("%x", fingerprint.Sum(nil))
}

func getLiveManifestFields_WithIgnoredFields(ignoredFields []string, userProvided *yaml.Manifest, liveManifest *yaml.Manifest) string {

	// there is a special user case for secrets.
	// If they are defined as manifests with StringData, it will always provide a non-empty plan
	// so we will do a small lifehack here
	if userProvided.GetKind() == "Secret" && userProvided.GetAPIVersion() == "v1" {
		if stringData, found := userProvided.Raw.Object["stringData"]; found {
			// there is an edge case where stringData might be nil and not a map[string]interface{}
			// in this case we will just ignore it
			if stringData, ok := stringData.(map[string]interface{}); ok {
				// move all stringdata values to the data
				for k, v := range stringData {
					encodedString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v", v)))
					meta_v1_unstruct.SetNestedField(userProvided.Raw.Object, encodedString, "data", k)
				}
				// and unset the stringData entirely
				meta_v1_unstruct.RemoveNestedField(userProvided.Raw.Object, "stringData")
			}
		}
	}

	flattenedUser := flatten.Flatten(userProvided.Raw.Object)
	flattenedLive := flatten.Flatten(liveManifest.Raw.Object)

	// remove any fields from the user provided set or control fields that we want to ignore
	fieldsToTrim := append(kubernetesControlFields, ignoredFields...)
	for _, field := range fieldsToTrim {
		delete(flattenedUser, field)

		// check for any nested fields to ignore
		for k, _ := range flattenedUser {
			if strings.HasPrefix(k, field+".") {
				delete(flattenedUser, k)
			}
		}
	}

	// update the user provided flattened string with the live versions of the keys
	// this implicitly excludes anything that the user didn't provide as it was added by kubernetes runtime (annotations/mutations etc)
	var userKeys []string
	for userKey, userValue := range flattenedUser {
		normalizedUserValue := strings.TrimSpace(userValue)

		// only include the value if it exists in the live version
		// that is, don't add to the userKeys array unless the key still exists in the live manifest
		if _, exists := flattenedLive[userKey]; exists {
			userKeys = append(userKeys, userKey)
			normalizedLiveValue := strings.TrimSpace(flattenedLive[userKey])
			if normalizedUserValue != normalizedLiveValue {
				log.Printf("[TRACE] yaml drift detected in %s for %s, was: %s now: %s", userProvided.GetSelfLink(), userKey, normalizedUserValue, normalizedLiveValue)
			}
			flattenedUser[userKey] = getFingerprint(normalizedLiveValue)
		} else {
			if normalizedUserValue != "" {
				log.Printf("[TRACE] yaml drift detected in %s for %s, was %s now blank", userProvided.GetSelfLink(), userKey, normalizedUserValue)
			}
		}
	}

	sort.Strings(userKeys)
	var returnedValues []string
	for _, k := range userKeys {
		returnedValues = append(returnedValues, fmt.Sprintf("%s=%s", k, flattenedUser[k]))
	}

	return strings.Join(returnedValues, "\n")
}

var kubernetesControlFields = []string{
	"status",
	"metadata.finalizers",
	"metadata.initializers",
	"metadata.ownerReferences",
	"metadata.creationTimestamp",
	"metadata.generation",
	"metadata.resourceVersion",
	"metadata.uid",
	"metadata.annotations.kubectl.kubernetes.io/last-applied-configuration",
	"metadata.managedFields",
}
