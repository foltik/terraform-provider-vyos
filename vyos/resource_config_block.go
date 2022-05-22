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
	configs, err := client.Config.ShowTree(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}
	// Dont care about sub config blocks
	for attr, val := range configs {
		switch val.(type) {
		default:
			continue
		case string:
			return diag.Errorf("Configuration block '%s' already exists and has '%s' set, try a resource import instead.", path, attr)
		case int:
			return diag.Errorf("Configuration block '%s' already exists and has '%s' set, try a resource import instead.", path, attr)
		}
	}

	configs = d.Get("configs").(map[string]interface{})

	commands := map[string]interface{}{}
	for attr, val := range configs {
		commands[path+" "+attr] = val
	}

	err = client.Config.SetTree(ctx, commands)
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

	configs, err := c.Config.ShowTree(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	// Remove child blocks of config
	for attr, val := range configs {
		switch val.(type) {
		default:
			delete(configs, attr)
		case string:
			continue
		case int:
			continue
		}
	}

	// Easiest way to allow ImportStatePassthroughContext to work is to set the path
	if d.Get("path") == "" {
		if err := d.Set("path", path); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("configs", configs); err != nil {
		return diag.FromErr(err)
	}

	return diags
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
			deleted_attrs = append(deleted_attrs, path+" "+old_attr)
		}
	}

	errDel := c.Config.Delete(ctx, deleted_attrs...)
	if errDel != nil {
		return diag.FromErr(errDel)
	}

	commands := map[string]interface{}{}
	for attr, val := range new_configs {
		commands[path+" "+attr] = val
	}

	errSet := c.Config.SetTree(ctx, commands)
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
