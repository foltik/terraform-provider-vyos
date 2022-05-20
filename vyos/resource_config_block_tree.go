package vyos

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/foltik/vyos-client-go/client"
)

func resourceConfigBlockTree() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource is useful when a single command is not enough for a valid config commit and children paths are needed.",
		CreateContext: resourceConfigBlockTreeCreate,
		ReadContext:   resourceConfigBlockTreeRead,
		UpdateContext: resourceConfigBlockTreeUpdate,
		DeleteContext: resourceConfigBlockTreeDelete,
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
			},
		},
        Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(10 * time.Second),
			Read:    schema.DefaultTimeout(10 * time.Second),
			Update:  schema.DefaultTimeout(10 * time.Second),
			Delete:  schema.DefaultTimeout(10 * time.Second),
			Default: schema.DefaultTimeout(10 * time.Second),
		},
	}
}

func resourceConfigBlockTreeCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	client := *p.client
	path := d.Get("path").(string)

	// Check if config already exists
	configs, err := client.Config.ShowTree(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}
	
	for attr, _ := range configs {
		return diag.Errorf("Configuration block '%s' already exists and has '%s' set, try a resource import instead.", path, attr)
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

func resourceConfigBlockTreeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client
	path := d.Id()

	configsTree, err := c.Config.ShowTree(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	flat, err := client.Flatten(configsTree)
	if err != nil {
		return diag.FromErr(err)
	}

	configs := map[string]string{}
	for _, path_value := range flat {
		path := path_value[0]
		value := path_value[1]
		configs[path] = value
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

func resourceConfigBlockTreeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

func resourceConfigBlockTreeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
