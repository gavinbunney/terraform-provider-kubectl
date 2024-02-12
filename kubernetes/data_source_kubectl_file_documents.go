package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/gavinbunney/terraform-provider-kubectl/yaml"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubectlFileDocuments() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubectlFileDocumentsRead,
		Schema: map[string]*schema.Schema{
			"content": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"documents": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"manifests": &schema.Schema{
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func dataSourceKubectlFileDocumentsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	content := d.Get("content").(string)
	documents, err := yaml.SplitMultiDocumentYAML(content)
	if err != nil {
		return diag.FromErr(err)
	}

	manifests := make(map[string]string, 0)
	for _, doc := range documents {
		manifest, err := yaml.ParseYAML(doc)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to parse yaml as a kubernetes yaml manifest: %v", err))
		}

		parsed, err := manifest.AsYAML()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to parse convert manifest to yaml: %v", err))
		}

		if _, exists := manifests[manifest.GetSelfLink()]; exists {
			return diag.FromErr(fmt.Errorf("duplicate manifest found with id: %v", manifest.GetSelfLink()))
		}

		manifests[manifest.GetSelfLink()] = parsed
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(content))))
	_ = d.Set("documents", documents)
	_ = d.Set("manifests", manifests)
	return nil
}
