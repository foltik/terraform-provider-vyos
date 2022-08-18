package vyos

import (
	"context"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInfoDhcpServer() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "service dhcp-server shared-network-name {{shared_network_name}}",
		CreateRequiredTemplates: []string{},
		DeleteStrategy:          resourceInfo.DeleteTypeParameters,
		DeleteBlockerTemplates:  []string{"service dhcp-server shared-network-name {{shared_network_name}} subnet"},
		ResourceSchema: &schema.Resource{
			Description:   "IPv4 DHCP Server. VyOS uses ISC DHCP server for both IPv4. The network topology is declared by shared-network-name and the subnet declarations. The DHCP service can serve multiple shared networks, with each shared network having 1 or more subnets. Each subnet must be present on an interface. A range can be declared inside a subnet to define a pool of dynamic addresses. Multiple ranges can be defined and can contain holes. Static mappings can be set to assign “static” addresses to clients based on their MAC address.",
			ReadContext:   resourceDhcpServerRead,
			CreateContext: resourceDhcpServerCreate,
			UpdateContext: resourceDhcpServerUpdate,
			DeleteContext: resourceDhcpServerDelete,
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
					Description: "Name of the network the DCHP server config is responsible for.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"domain_name": {
					Description: "The domain-name parameter should be the domain name that will be appended to the client’s hostname to form a fully-qualified domain-name (FQDN) (DHCP Option 015).",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"domain_search": {
					Description: "The domain-name parameter should be the domain name used when completing DNS request where no full FQDN is passed. (DHCP Option 119)",
					Type:        schema.TypeList,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"name_server": {
					Description: "Inform client that the DNS server can be found at <address>.",
					Type:        schema.TypeList,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"ping_check": {
					Description: "When the DHCP server is considering dynamically allocating an IP address to a client, it first sends an ICMP Echo request (a ping) to the address being assigned. It waits for a second, and if no ICMP Echo response has been heard, it assigns the address. If a response is heard, the lease is abandoned, and the server does not respond to the client. The lease will remain abandoned for a minimum of abandon-lease-time seconds (defaults to 24 hours). If there are no free addresses but there are abandoned IP addresses, the DHCP server will attempt to reclaim an abandoned IP address regardless of the value of abandon-lease-time.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"authoritative": {
					Description: "This says that this device is the only DHCP server for this network. If other devices are trying to offer DHCP leases, this machine will send ‘DHCPNAK’ to any device trying to request an IP address that is not valid for this network.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
			},
		},
	}
}

func resourceDhcpServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {

	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceRead(ctx, d, &client, resourceInfoDhcpServer())

}

func resourceDhcpServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceCreate(ctx, d, &client, resourceInfoDhcpServer())
}

func resourceDhcpServerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceUpdate(ctx, d, &client, resourceInfoDhcpServer())
}

func resourceDhcpServerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceDelete(ctx, d, &client, resourceInfoDhcpServer())
}
