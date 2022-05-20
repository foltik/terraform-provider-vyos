package vyos

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/foltik/vyos-client-go/client"
)

func resourceConfig() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource manages a single configuration value. This as well as vyos_config_block can act as a fallback when a dedicated resource does not exist.",
		CreateContext: resourceConfigCreate,
		ReadContext:   resourceConfigRead,
		UpdateContext: resourceConfigUpdate,
		DeleteContext: resourceConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The resource ID, same as the `key`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"key": {
				Description: "Config path separated by spaces.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"value": {
				Description: "Config value.",
				Type:        schema.TypeString,
				Required:    true,
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

func resourceConfigCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key, value := d.Get("key").(string), d.Get("value").(string)

	var diags diag.Diagnostics

	// Check if config already exists
	val, err := c.Config.Show(ctx, key)
	if err != nil {
		return diag.FromErr(err)
	}
	// Dont care about sub config blocks
	if val != nil {
		return diag.Errorf("Configuration '%s' already exists with value '%s' set, try a resource import instead.", key, *val)
	}

	err = c.Config.Set(ctx, key, value)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(key)
	return diags
}

func resourceConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key := d.Id()

	// Convert old unix timestamp style ID to key path for existing resources to support importing
	if _, err := strconv.Atoi(key); err == nil {
		key = d.Get("key").(string)
		d.SetId(key)
	}

	// Easiest way to allow ImportStatePassthroughContext to work is to set the path
	if d.Get("key") == "" {
		if err := d.Set("key", key); err != nil {
			return diag.FromErr(err)
		}
	}

	value, err := c.Config.Show(ctx, key)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("value", value); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceConfigUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key, value := d.Get("key").(string), d.Get("value").(string)

	err := c.Config.Set(ctx, key, value)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceConfigDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key := d.Get("key").(string)

	err := c.Config.Delete(ctx, key)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}
