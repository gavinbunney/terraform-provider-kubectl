package kubernetes

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubectlServerVersion() *schema.Resource {
	return &schema.Resource{
		CreateContext: dataSourceKubectlServerVersionRead,
		ReadContext:   dataSourceKubectlServerVersionRead,
		DeleteContext: resourceKubectlServerVersionDelete,
		Schema: map[string]*schema.Schema{
			"triggers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"major": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"minor": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"patch": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_commit": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"build_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"platform": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceKubectlServerVersionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
