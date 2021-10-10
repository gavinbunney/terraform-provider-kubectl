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
		},
	}
}

func dataSourceKubectlFileDocumentsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	content := d.Get("content").(string)
	documents, err := yaml.SplitMultiDocumentYAML(content)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(content))))
	d.Set("documents", documents)
	return nil
}
