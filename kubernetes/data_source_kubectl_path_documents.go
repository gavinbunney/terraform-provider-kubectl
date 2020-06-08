package kubernetes

import (
	"crypto/sha256"
	"fmt"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/helper/schema"
	tflang "github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
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
			"vars": {
				Type:         schema.TypeMap,
				Optional:     true,
				Default:      make(map[string]interface{}),
				Description:  "variables to substitute",
				ValidateFunc: validateVarsAttribute,
			},
			"disable_template": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag to disable template parsing of the loaded documents",
			},
		},
	}
}

func dataSourceKubectlPathDocumentsRead(d *schema.ResourceData, m interface{}) error {
	p := d.Get("pattern").(string)
	vars := d.Get("vars").(map[string]interface{})
	disableTemplate := d.Get("disable_template").(bool)

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

		// before splitting the document, parse out any template details
		rendered := string(content)
		if !disableTemplate {
			rendered, err = parseTemplate(rendered, vars)
			if err != nil {
				return fmt.Errorf("failed to render %v: %v", item, err)
			}
		}

		documents, err := splitMultiDocumentYAML(rendered)
		if err != nil {
			return err
		}

		allDocuments = append(allDocuments, documents...)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(allDocuments, "")))))
	d.Set("documents", allDocuments)
	return nil
}

// execute parses and executes a template using vars.
func parseTemplate(s string, vars map[string]interface{}) (string, error) {
	expr, diags := hclsyntax.ParseTemplate([]byte(s), "<template_file>", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return "", diags
	}

	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{},
	}

	for k, v := range vars {
		// In practice today this is always a string due to limitations of
		// the schema system. In future we'd like to support other types here.
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("unexpected type for variable %q: %T", k, v)
		}
		ctx.Variables[k] = cty.StringVal(s)
	}

	// We borrow the functions from Terraform itself here. This is convenient
	// but note that this is coming from whatever version of Terraform we
	// have vendored in to this codebase, not from the version of Terraform
	// the user is running, and so the set of functions won't always match
	// between Terraform itself and this provider.
	// (Over time users will hopefully transition over to Terraform's built-in
	// templatefile function instead and we can phase this provider out.)
	scope := &tflang.Scope{
		BaseDir: ".",
	}
	ctx.Functions = scope.Functions()

	result, diags := expr.Value(ctx)
	if diags.HasErrors() {
		return "", diags
	}

	// Our result must always be a string, so we'll try to convert it.
	var err error
	result, err = ctyconvert.Convert(result, cty.String)
	if err != nil {
		return "", fmt.Errorf("invalid template result: %s", err)
	}

	return result.AsString(), nil
}

func validateVarsAttribute(v interface{}, key string) (ws []string, es []error) {
	var badVars []string
	for k, v := range v.(map[string]interface{}) {
		switch v.(type) {
		case []interface{}:
			badVars = append(badVars, fmt.Sprintf("%s (list)", k))
		case map[string]interface{}:
			badVars = append(badVars, fmt.Sprintf("%s (map)", k))
		}
	}

	if len(badVars) > 0 {
		es = append(es, fmt.Errorf(
			"%s: cannot contain non-primitives; bad keys: %s",
			key, strings.Join(badVars, ", ")))
	}
	return
}
