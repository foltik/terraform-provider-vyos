package vyos

import (
	providerStructure "github.com/foltik/terraform-provider-vyos/vyos/provider-structure"
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
			"save": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Save after making changes in Vyos",
			},
			"save_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "File to save configuration. Uses config.boot by default.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"vyos_config":                          resourceConfig(),
			"vyos_config_block":                    resourceConfigBlock(),
			"vyos_firewall_service":                resourceInfoFirewallService().ResourceSchema,
			"vyos_firewall_port_group":             resourceFirewallPortGroup().ResourceSchema,
			"vyos_firewall_address_group":          resourceFirewallAddressGroup().ResourceSchema,
			"vyos_firewall_network_group":          resourceFirewallNetworkGroup().ResourceSchema,
			"vyos_firewall_rule_set":               resourceFirewallRuleSet().ResourceSchema,
			"vyos_firewall_rule":                   resourceFirewallRule().ResourceSchema,
			"vyos_static_host_mapping":             resourceStaticHostMapping(),
			"vyos_dhcp_service":                    resourceInfoDhcpService().ResourceSchema,
			"vyos_dhcp_server":                     resourceInfoDhcpServer().ResourceSchema,
			"vyos_dhcp_server_subnet":              resourceInfoDhcpServerSubnet().ResourceSchema,
			"vyos_dhcp_server_subnet_address_pool": resourceInfoDhcpServerSubnetAddressPool().ResourceSchema,
			"vyos_vrrp_group":                      resourceInfoVrrpGroup().ResourceSchema,
		},
		DataSourcesMap: map[string]*schema.Resource{
			"vyos_config": dataSourceConfig(),
		},
		ConfigureContextFunc: providerStructure.ProviderConfigure,
	}
}
