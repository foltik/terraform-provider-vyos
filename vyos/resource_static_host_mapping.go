package vyos

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/foltik/vyos-client-go/client"
)

func resourceStaticHostMapping() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource manages a static host mapping with the given hostname and ipv4 address.",
		CreateContext: resourceStaticHostMappingCreate,
		ReadContext:   resourceStaticHostMappingRead,
		UpdateContext: resourceStaticHostMappingUpdate,
		DeleteContext: resourceStaticHostMappingDelete,
		Schema: map[string]*schema.Schema{
			"host": {
				Description: "Hostname.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"ip": {
				Description: "IPv4 address.",
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

func resourceStaticHostMappingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	host, ip := d.Get("host").(string), d.Get("ip").(string)

	path := fmt.Sprintf("system static-host-mapping host-name %s inet", host)
	err := c.Config.Set(path, ip)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diag.Diagnostics{}
}

func resourceStaticHostMappingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	host := d.Get("host").(string)

	path := fmt.Sprintf("system static-host-mapping host-name %s inet", host)
	ip, err := c.Config.Show(path)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("ip", ip); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceStaticHostMappingUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	host, ip := d.Get("host").(string), d.Get("ip").(string)

	// If the hostname changes, so does the configuration path,
	// so we need to delete the old mapping.
	if d.HasChange("host") {
		old, _ := d.GetChange("host")
		path := fmt.Sprintf("system static-host-mapping host-name %s", old)
		err := c.Config.Delete(path)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	path := fmt.Sprintf("system static-host-mapping host-name %s inet", host)
	err := c.Config.Set(path, ip)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceStaticHostMappingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	host := d.Get("host").(string)

	path := fmt.Sprintf("system static-host-mapping host-name %s", host)
	err := c.Config.Delete(path)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}
