package kubernetes

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
)

func dataSourceKubectlKustomizeDocuments() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubectlKustomizeDocumentsRead,
		Schema: map[string]*schema.Schema{
			"target": {
				Type:     schema.TypeString,
				Required: true,
			},
			"load_restrictor": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "rootOnly",
			},
			"add_managed_by_label": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"documents": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceKubectlKustomizeDocumentsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	target := d.Get("target").(string)

	opts, err := makeKustOpts(d)
	if err != nil {
		return diag.FromErr(err)
	}

	k := krusty.MakeKustomizer(opts)
	memFS := filesys.MakeFsOnDisk()

	rm, err := k.Run(memFS, target)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error rendering kustomization: %w", err))
	}

	documents, err := readFromResMap(rm)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading documents: %w", err))
	}

	d.SetId(target)
	d.Set("documents", documents)
	return nil
}

func makeKustOpts(d *schema.ResourceData) (*krusty.Options, error) {
	opts := krusty.MakeDefaultOptions()

	rName := d.Get("load_restrictor").(string)
	switch rName {
	case "none":
		opts.LoadRestrictions = types.LoadRestrictionsNone
	case "rootOnly":
		opts.LoadRestrictions = types.LoadRestrictionsRootOnly
	default:
		return nil, fmt.Errorf("invalid restrictor '%s'", rName)
	}

	opts.AddManagedbyLabel = d.Get("add_managed_by_label").(bool)

	return opts, nil
}

func readFromResMap(rm resmap.ResMap) ([]string, error) {
	docs := make([]string, 0)

	for _, res := range rm.Resources() {
		b, err := res.AsYAML()
		if err != nil {
			return nil, err
		}

		docs = append(docs, string(b))
	}

	return docs, nil
}
