package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubectlServerVersion() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubectlServerVersionRead,
		Schema: map[string]*schema.Schema{
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

func dataSourceKubectlServerVersionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*KubeProvider)
	discoveryClient, err := provider.ToDiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}

	discoveryClient.Invalidate()
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return diag.FromErr(err)
	}

	serverSemver := strings.Split(serverVersion.String(), ".")
	if len(serverSemver) >= 3 {
		_ = d.Set("major", strings.ReplaceAll(serverSemver[0], "v", ""))
		_ = d.Set("minor", serverSemver[1])
		_ = d.Set("patch", strings.Split(serverSemver[2], "-")[0])
	} else {
		_ = d.Set("major", serverVersion.Major)
		_ = d.Set("minor", serverVersion.Minor)
		_ = d.Set("patch", "")
	}

	_ = d.Set("version", strings.Split(serverVersion.String(), "-")[0])
	_ = d.Set("git_version", serverVersion.GitVersion)
	_ = d.Set("git_commit", serverVersion.GitCommit)
	_ = d.Set("build_date", serverVersion.BuildDate)
	_ = d.Set("platform", serverVersion.Platform)

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(serverVersion.String()))))
	return nil
}
