package vyos

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("VYOS_KEY", nil),
			},
			"cert": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"vyos_config":       resourceConfig(),
			"vyos_config_block": resourceConfigBlock(),
			//"vyos_firewall_group_address": resourceFirewallGroupAddress(),
			"vyos_firewall_rule_set":   resourceFirewallRuleSet(),
			"vyos_firewall_rule":       resourceFirewallRule(),
			"vyos_static_host_mapping": resourceStaticHostMapping(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"vyos_config": dataSourceConfig(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	url := d.Get("url").(string)
	key := d.Get("key").(string)

	cert := d.Get("cert").(string)

	//lint:ignore SA4006 placeholder
	c := &client.Client{}

	if cert != "" {
		return nil, diag.Errorf("TODO: Use trusted self signed certificate")
	} else {
		// Just allow self signed certificates if a trusted cert isn't specified
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		cc := &http.Client{Transport: tr, Timeout: 10 * time.Second}
		c = client.NewWithClient(cc, url, key)
	}

	return c, diag.Diagnostics{}
}
