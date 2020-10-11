package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubectlFileDocuments() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubectlFileDocumentsRead,
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

func dataSourceKubectlFileDocumentsRead(d *schema.ResourceData, m interface{}) error {
	content := d.Get("content").(string)
	documents, err := splitMultiDocumentYAML(content)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(content))))
	d.Set("documents", documents)
	return nil
}
