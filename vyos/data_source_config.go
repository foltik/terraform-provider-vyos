package vyos

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/foltik/vyos-client-go/client"
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
			Read:    schema.DefaultTimeout(10 * time.Second),
			Default: schema.DefaultTimeout(10 * time.Second),
		},
	}
}

func dataSourceConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	key := d.Get("key").(string)

	value, err := c.Config.Show(ctx, key)
	if err != nil {
		return diag.FromErr(err)
	}

	if value != nil {
		if err := d.Set("value", *value); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diag.Diagnostics{}
}
