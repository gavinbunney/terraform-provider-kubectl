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
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"major": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"minor": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"patch": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_version": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_commit": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"build_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"platform": &schema.Schema{
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
