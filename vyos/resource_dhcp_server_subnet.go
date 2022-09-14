package vyos

import (
	"context"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInfoDhcpServerSubnet() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "service dhcp-server shared-network-name {{shared_network_name}} subnet {{subnet}}",
		CreateRequiredTemplates: []string{"service dhcp-server shared-network-name {{shared_network_name}}"},
		DeleteStrategy:          resourceInfo.DeleteTypeResource,
		DeleteBlockerTemplates:  []string{},
		ResourceSchema: &schema.Resource{
			Description: "[IPv4 DHCP Server Subnet](https://docs.vyos.io/en/latest/configuration/service/dhcp-server.html).",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceCreate(ctx, d, m, resourceInfoDhcpServerSubnet())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceRead(ctx, d, m, resourceInfoDhcpServerSubnet())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceUpdate(ctx, d, m, resourceInfoDhcpServerSubnet())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceDelete(ctx, d, m, resourceInfoDhcpServerSubnet())
			},
			Importer: &schema.ResourceImporter{
				StateContext: schema.ImportStatePassthroughContext,
			},
			Schema: map[string]*schema.Schema{
				"id": {
					Description: "The resource ID",
					Type:        schema.TypeString,
					Computed:    true,
				},
				"shared_network_name": {
					Description: "Name of the DHCP server network.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"subnet": {
					Description:      "Name of the DHCP subnet.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: resourceInfo.ValidateDiagStringKeyField(),
				},
				"default_router": {
					Description: "This is a configuration parameter for the `subnet`, saying that as part of the response, tell the client that the default gateway can be reached at `address`.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"name_server": {
					Description: "This is a configuration parameter for the subnet, saying that as part of the response, tell the client that the DNS server can be found at `address`.",
					Type:        schema.TypeList,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"lease": {
					Description: "Assign the IP address to this machine for `time` seconds. " +
						"The default value is 86400 seconds which corresponds to one day.",
					Type:     schema.TypeInt,
					Optional: true,
					Default:  86400,
				},
				"exclude": {
					Description: "Always exclude this address from any defined range. This address will never be assigned by the DHCP server. " +
						"This option can be specified multiple times.",
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"domain_name": {
					Description: "The domain-name parameter should be the domain name that will be appended to the client’s hostname to form a fully-qualified domain-name (FQDN) (DHCP Option 015).",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"domain_search": {
					Description: "The domain-name parameter should be the domain name used when completing DNS request where no full FQDN is passed. This option can be given multiple times if you need multiple search domains (DHCP Option 119).",
					Type:        schema.TypeList,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"ping_check": {
					Description: "When the DHCP server is considering dynamically allocating an IP address to a client, it first sends an ICMP Echo request (a ping) to the address being assigned. It waits for a second, and if no ICMP Echo response has been heard, it assigns the address. " +
						"If a response is heard, the lease is abandoned, and the server does not respond to the client. The lease will remain abandoned for a minimum of abandon-lease-time seconds (defaults to 24 hours). " +
						"If a there are no free addresses but there are abandoned IP addresses, the DHCP server will attempt to reclaim an abandoned IP address regardless of the value of abandon-lease-time.",
					Type:     schema.TypeBool,
					Optional: true,
				},
				"enable_failover": {
					Description: "Enable DHCP failover configuration for this address pool.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
			},
		},
	}
}
