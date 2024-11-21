package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/alekc/terraform-provider-kubectl/yaml"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform/lang/funcs"
	ctyyaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func dataSourceKubectlPathDocuments() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubectlPathDocumentsRead,
		Schema: map[string]*schema.Schema{
			"pattern": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Glob pattern to search for",
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
			"vars": {
				Type:         schema.TypeMap,
				Optional:     true,
				Default:      make(map[string]interface{}),
				Description:  "Variables to substitute",
				ValidateFunc: validateVarsAttribute,
			},
			"sensitive_vars": {
				Type:         schema.TypeMap,
				Optional:     true,
				Default:      make(map[string]interface{}),
				Sensitive:    true,
				Description:  "Sensitive variables to substitute, allowing for hiding sensitive variables in terraform output",
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

func dataSourceKubectlPathDocumentsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	p := d.Get("pattern").(string)
	vars := d.Get("vars").(map[string]interface{})
	sensitiveVars := d.Get("sensitive_vars").(map[string]interface{})
	for k, v := range sensitiveVars {
		vars[k] = v
	}
	disableTemplate := d.Get("disable_template").(bool)

	items, err := filepath.Glob(p)
	if err != nil {
		return diag.FromErr(err)
	}
	sort.Strings(items)
	var allDocuments []string
	for _, item := range items {
		content, err := ioutil.ReadFile(item)
		if err != nil {
			return diag.Errorf("error loading document from file: %v\n%v", item, err)
		}

		// before splitting the document, parse out any template details
		rendered := string(content)
		if !disableTemplate {
			rendered, err = parseTemplate(rendered, vars)
			if err != nil {
				return diag.Errorf("failed to render %v: %v", item, err)
			}
		}

		documents, err := yaml.SplitMultiDocumentYAML(rendered)
		if err != nil {
			return diag.FromErr(err)
		}

		allDocuments = append(allDocuments, documents...)
	}

	manifests := make(map[string]string, 0)
	for _, doc := range allDocuments {
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

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(allDocuments, "")))))
	_ = d.Set("documents", allDocuments)
	_ = d.Set("manifests", manifests)
	return nil
}

// execute parses and executes a template using vars.
func parseTemplate(s string, vars map[string]interface{}) (string, error) {
	expr, diags := hclsyntax.ParseTemplate([]byte(s), "<template_file>", hcl.Pos{Line: 1, Column: 1})
	if expr == nil || (diags != nil && diags.HasErrors()) {
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
	// The provider sdk doesn't provide access to these, so we need to fetch into
	// Terraform itself - not great, but only way to make our own "templating" feature
	ctx.Functions = kubectlPathDocumentsFunctions()

	result, diags := expr.Value(ctx)
	if diags != nil && diags.HasErrors() {
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

var (
	pathDocsFuncsLock                              = &sync.Mutex{}
	pathDocsFuncs     map[string]function.Function = nil
)

// Functions returns the set of functions that should be used to when evaluating
// expressions in the receiving scope.
func kubectlPathDocumentsFunctions() map[string]function.Function {
	pathDocsFuncsLock.Lock()
	if pathDocsFuncs == nil {
		// Some of our functions are just directly the cty stdlib functions.
		// Others are implemented in the subdirectory "funcs" here in this
		// repository. New functions should generally start out their lives
		// in the "funcs" directory and potentially graduate to cty stdlib
		// later if the functionality seems to be something domain-agnostic
		// that would be useful to all applications using cty functions.
		baseDir := "."
		pathDocsFuncs = map[string]function.Function{
			"abs":              stdlib.AbsoluteFunc,
			"abspath":          funcs.AbsPathFunc,
			"basename":         funcs.BasenameFunc,
			"base64decode":     funcs.Base64DecodeFunc,
			"base64encode":     funcs.Base64EncodeFunc,
			"base64gzip":       funcs.Base64GzipFunc,
			"base64sha256":     funcs.Base64Sha256Func,
			"base64sha512":     funcs.Base64Sha512Func,
			"bcrypt":           funcs.BcryptFunc,
			"can":              tryfunc.CanFunc,
			"ceil":             funcs.CeilFunc,
			"chomp":            funcs.ChompFunc,
			"cidrhost":         funcs.CidrHostFunc,
			"cidrnetmask":      funcs.CidrNetmaskFunc,
			"cidrsubnet":       funcs.CidrSubnetFunc,
			"cidrsubnets":      funcs.CidrSubnetsFunc,
			"coalesce":         funcs.CoalesceFunc,
			"coalescelist":     funcs.CoalesceListFunc,
			"compact":          funcs.CompactFunc,
			"concat":           stdlib.ConcatFunc,
			"contains":         funcs.ContainsFunc,
			"csvdecode":        stdlib.CSVDecodeFunc,
			"dirname":          funcs.DirnameFunc,
			"distinct":         funcs.DistinctFunc,
			"element":          funcs.ElementFunc,
			"chunklist":        funcs.ChunklistFunc,
			"file":             funcs.MakeFileFunc(baseDir, false),
			"fileexists":       funcs.MakeFileExistsFunc(baseDir),
			"fileset":          funcs.MakeFileSetFunc(baseDir),
			"filebase64":       funcs.MakeFileFunc(baseDir, true),
			"filebase64sha256": funcs.MakeFileBase64Sha256Func(baseDir),
			"filebase64sha512": funcs.MakeFileBase64Sha512Func(baseDir),
			"filemd5":          funcs.MakeFileMd5Func(baseDir),
			"filesha1":         funcs.MakeFileSha1Func(baseDir),
			"filesha256":       funcs.MakeFileSha256Func(baseDir),
			"filesha512":       funcs.MakeFileSha512Func(baseDir),
			"flatten":          funcs.FlattenFunc,
			"floor":            funcs.FloorFunc,
			"format":           stdlib.FormatFunc,
			"formatdate":       stdlib.FormatDateFunc,
			"formatlist":       stdlib.FormatListFunc,
			"indent":           funcs.IndentFunc,
			"index":            funcs.IndexFunc,
			"join":             funcs.JoinFunc,
			"jsondecode":       stdlib.JSONDecodeFunc,
			"jsonencode":       stdlib.JSONEncodeFunc,
			"keys":             funcs.KeysFunc,
			"length":           funcs.LengthFunc,
			"list":             funcs.ListFunc,
			"log":              funcs.LogFunc,
			"lookup":           funcs.LookupFunc,
			"lower":            stdlib.LowerFunc,
			"map":              funcs.MapFunc,
			"matchkeys":        funcs.MatchkeysFunc,
			"max":              stdlib.MaxFunc,
			"md5":              funcs.Md5Func,
			"merge":            funcs.MergeFunc,
			"min":              stdlib.MinFunc,
			"parseint":         funcs.ParseIntFunc,
			"pathexpand":       funcs.PathExpandFunc,
			"pow":              funcs.PowFunc,
			"range":            stdlib.RangeFunc,
			"regex":            stdlib.RegexFunc,
			"regexall":         stdlib.RegexAllFunc,
			"replace":          funcs.ReplaceFunc,
			"reverse":          funcs.ReverseFunc,
			"rsadecrypt":       funcs.RsaDecryptFunc,
			"setintersection":  stdlib.SetIntersectionFunc,
			"setproduct":       funcs.SetProductFunc,
			"setsubtract":      stdlib.SetSubtractFunc,
			"setunion":         stdlib.SetUnionFunc,
			"sha1":             funcs.Sha1Func,
			"sha256":           funcs.Sha256Func,
			"sha512":           funcs.Sha512Func,
			"signum":           funcs.SignumFunc,
			"slice":            funcs.SliceFunc,
			"sort":             funcs.SortFunc,
			"split":            funcs.SplitFunc,
			"strrev":           stdlib.ReverseFunc,
			"substr":           stdlib.SubstrFunc,
			"timestamp":        funcs.TimestampFunc,
			"timeadd":          funcs.TimeAddFunc,
			"title":            funcs.TitleFunc,
			"tostring":         funcs.MakeToFunc(cty.String),
			"tonumber":         funcs.MakeToFunc(cty.Number),
			"tobool":           funcs.MakeToFunc(cty.Bool),
			"toset":            funcs.MakeToFunc(cty.Set(cty.DynamicPseudoType)),
			"tolist":           funcs.MakeToFunc(cty.List(cty.DynamicPseudoType)),
			"tomap":            funcs.MakeToFunc(cty.Map(cty.DynamicPseudoType)),
			"transpose":        funcs.TransposeFunc,
			"trim":             funcs.TrimFunc,
			"trimprefix":       funcs.TrimPrefixFunc,
			"trimspace":        funcs.TrimSpaceFunc,
			"trimsuffix":       funcs.TrimSuffixFunc,
			"try":              tryfunc.TryFunc,
			"upper":            stdlib.UpperFunc,
			"urlencode":        funcs.URLEncodeFunc,
			"uuid":             funcs.UUIDFunc,
			"uuidv5":           funcs.UUIDV5Func,
			"values":           funcs.ValuesFunc,
			"yamldecode":       ctyyaml.YAMLDecodeFunc,
			"yamlencode":       ctyyaml.YAMLEncodeFunc,
			"zipmap":           funcs.ZipmapFunc,
		}

		pathDocsFuncs["templatefile"] = funcs.MakeTemplateFileFunc(baseDir, func() map[string]function.Function {
			// The templatefile function prevents recursive calls to itself
			// by copying this map and overwriting the "templatefile" entry.
			return pathDocsFuncs
		})
	}
	pathDocsFuncsLock.Unlock()
	return pathDocsFuncs
}
