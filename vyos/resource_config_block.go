package vyos

import (
	"context"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceConfigBlock() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource is useful when a single command is not enough for a valid config commit. This as well as vyos_config can act as a fallback when a dedicated resource does not exist.",
		CreateContext: resourceConfigBlockCreate,
		ReadContext:   resourceConfigBlockRead,
		UpdateContext: resourceConfigBlockUpdate,
		DeleteContext: resourceConfigBlockDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The resource ID, same as the `path`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"path": {
				Description:      "Config path seperated by spaces.",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				ForceNew:         true,
			},
			"configs": {
				Description: "Key/Value map of config parameters.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:         true,
				ValidateDiagFunc: validation.MapKeyMatch(regexp.MustCompile("^[^ ]+$"), "Config keys can not contain whitespace"),
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(10 * time.Minute),
			Read:    schema.DefaultTimeout(10 * time.Minute),
			Update:  schema.DefaultTimeout(10 * time.Minute),
			Delete:  schema.DefaultTimeout(10 * time.Minute),
			Default: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceConfigBlockCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	client := *p.client
	path := d.Get("path").(string)

	// Check if config already exists
	configs, err := client.Config.Show(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	// Dont care about sub config blocks
	if configs != nil {
		return diag.Errorf("Configuration '%s' already exists with value '%s' set, try a resource import instead.", path, configs)
	}

	configs = d.Get("configs").(map[string]interface{})

	err = client.Config.Set(ctx, path, configs)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(path)
	p.conditionalSave(ctx)
	return diags
}

func resourceConfigBlockRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client
	path := d.Id()

	configs, err := c.Config.Show(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	switch value := configs.(type) {
	case map[string]any:
		if err := d.Set("configs", value); err != nil {
			return diag.FromErr(err)
		}
		return diags
	default:
		return diag.Errorf("Configuration at '%s' is not a string: %s.", path, value)
	}

	// // Remove child blocks of config
	// for attr, val := range configs {
	// 	switch val.(type) {
	// 	default:
	// 		delete(configs, attr)
	// 	case string:
	// 		continue
	// 	case int:
	// 		continue
	// 	}
	// }

}

func resourceConfigBlockUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client

	path := d.Get("path").(string)
	o, n := d.GetChange("configs")
	old_configs := o.(map[string]interface{})
	new_configs := n.(map[string]interface{})

	deleted_attrs := []string{}

	for old_attr := range old_configs {
		value, ok := new_configs[old_attr]
		_ = value
		if !ok {
			deleted_attrs = append(deleted_attrs, old_attr)
		}
	}

	errDel := c.Config.Delete(ctx, path, deleted_attrs)
	if errDel != nil {
		return diag.FromErr(errDel)
	}

	errSet := c.Config.Set(ctx, path, new_configs)
	if errSet != nil {
		return diag.FromErr(errSet)
	}

	p.conditionalSave(ctx)
	return diags
}

func resourceConfigBlockDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client
	path := d.Get("path").(string)

	err := c.Config.Delete(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	p.conditionalSave(ctx)
	return diags
}
