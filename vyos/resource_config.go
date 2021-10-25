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
		Schema: map[string]*schema.Schema{
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
	}
}

func resourceConfigCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key, value := d.Get("key").(string), d.Get("value").(string)

	var diags diag.Diagnostics

	err := c.Config.Set(key, value)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diags
}

func resourceConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key := d.Get("key").(string)

	value, err := c.Config.Show(key)
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

	err := c.Config.Set(key, value)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceConfigDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key := d.Get("key").(string)

	err := c.Config.Delete(key)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}
