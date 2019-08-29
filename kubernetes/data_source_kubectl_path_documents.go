package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

func dataSourceKubectlPathDocuments() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubectlPathDocumentsRead,
		Schema: map[string]*schema.Schema{
			"pattern": &schema.Schema{
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

func dataSourceKubectlPathDocumentsRead(d *schema.ResourceData, m interface{}) error {
	p := d.Get("pattern").(string)
	items, err := filepath.Glob(p)
	if err != nil {
		return err
	}
	sort.Strings(items)
	var allDocuments []string
	for _, item := range items {
		content, err := ioutil.ReadFile(item)
		if err != nil {
			return fmt.Errorf("error loading document from file: %v\n%v", item, err)
		}

		documents, err := splitMultiDocumentYAML(string(content))
		if err != nil {
			return err
		}

		allDocuments = append(allDocuments, documents...)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(allDocuments, "")))))
	d.Set("documents", allDocuments)
	return nil
}
