package vyos

import (
	"context"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceInfoDhcpService() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "service dhcp-server",
		CreateRequiredTemplates: []string{},
		DeleteStrategy:          resourceInfo.DeleteTypeParameters,
		DeleteBlockerTemplates:  []string{},
		StaticId:                "dhcpService",
		ResourceSchema: &schema.Resource{
			Description: "[IPv4 DHCP Server Global Config](https://docs.vyos.io/en/latest/configuration/service/dhcp-server.html). " +
				"**This is a global config, having more than one of this resource will casue continues diffs to occur.**",
			ReadContext:   resourceDhcpServiceRead,
			CreateContext: resourceDhcpServiceCreate,
			UpdateContext: resourceDhcpServiceUpdate,
			DeleteContext: resourceDhcpServiceDelete,
			Importer: &schema.ResourceImporter{
				StateContext: schema.ImportStatePassthroughContext,
			},
			Schema: map[string]*schema.Schema{
				"id": {
					Description: "The resource ID",
					Type:        schema.TypeString,
					Computed:    true,
				},
				"hostfile_update": {
					Description: "Create DNS record per client lease, by adding clients to /etc/hosts file. Entry will have format: `<shared-network-name>_<hostname>.<domain-name>`.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"host_decl_name": {
					Description: "Will drop `<shared-network-name>_` from client DNS record, using only the host declaration name and domain: `<hostname>.<domain-name>`.",
					Type:        schema.TypeBool,
					Optional:    true,
				},
				"listen_address": {
					Description: "This configuration parameter lets the DHCP server to listen for DHCP requests sent to the specified address, it is only realistically useful for a server whose only clients are reached via unicasts, such as via DHCP relay agents.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"failover": {
					Description: "VyOS provides support for DHCP failover. DHCP failover must be configured explicitly by the following statements. " +
						"**In order for the primary and the secondary DHCP server to keep their lease tables in sync, they must be able to reach each other on TCP port 647. If you have firewall rules in effect, adjust them accordingly.**",
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"source_address": {
								Description: "Local IP `address` used when communicating to the failover peer.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"remote": {
								Description: "Remote peer IP `address` of the second DHCP server in this failover cluster.",
								Type:        schema.TypeString,
								Required:    true,
							},
							"name": {
								Description: "A generic `name` referencing this sync service. **`name` must be identical on both sides!**",
								Type:        schema.TypeString,
								Required:    true,
							},
							"status": {
								Description:      "The primary and secondary statements determines whether the server is primary or secondary.",
								Type:             schema.TypeString,
								Required:         true,
								ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"primary", "secondary"}, false)),
							},
						},
					},
				},
			},
		},
	}
}

func resourceDhcpServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {

	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceReadGlobal(ctx, d, &client, resourceInfoDhcpServer())

}

func resourceDhcpServiceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceCreate(ctx, d, &client, resourceInfoDhcpServer())
}

func resourceDhcpServiceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceUpdate(ctx, d, &client, resourceInfoDhcpServer())
}

func resourceDhcpServiceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	return resourceInfo.ResourceDelete(ctx, d, &client, resourceInfoDhcpServer())
}
