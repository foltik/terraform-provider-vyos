package vyos

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceConfigRead,
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Read:    schema.DefaultTimeout(10 * time.Minute),
			Default: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func dataSourceConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	p := m.(*ProviderClass)
	c := *p.client
	key := d.Get("key").(string)

	value, err := c.Config.Show(ctx, key)
	if err != nil {
		return diag.FromErr(err)
	}

	switch value := value.(type) {
	case string:
		if err := d.Set("value", value); err != nil {
			return diag.FromErr(err)
		}
	default:
		return diag.Errorf("Configuration at '%s' is not a string: %s.", key, value)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diag.Diagnostics{}
}
