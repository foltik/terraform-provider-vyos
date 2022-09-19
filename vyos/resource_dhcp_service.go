package vyos

import (
	"context"
	"time"

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
		StaticId:                "global",
		ResourceSchema: &schema.Resource{
			Description: "[IPv4 DHCP Server Global Config](https://docs.vyos.io/en/latest/configuration/service/dhcp-server.html). " +
				"**This is a global config, having more than one of this resource will cause continues diffs to occur.**",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceCreateGlobal(ctx, d, m, resourceInfoDhcpService())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceReadGlobal(ctx, d, m, resourceInfoDhcpService())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceUpdateGlobal(ctx, d, m, resourceInfoDhcpService())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceDeleteGlobal(ctx, d, m, resourceInfoDhcpService())
			},
			Importer: &schema.ResourceImporter{
				StateContext: schema.ImportStatePassthroughContext,
			},
			Timeouts: &schema.ResourceTimeout{
				Create: schema.DefaultTimeout(10 * time.Minute),
				Delete: schema.DefaultTimeout(10 * time.Minute),
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
