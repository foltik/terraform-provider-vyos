package vyos

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
)

func resourceFirewallPortGroup() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "firewall group port-group {{name}}",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          resourceInfo.DeleteTypeResource,
		DeleteBlockerTemplates:  nil, //TODO can we support that firewall rules are using the group?
		ResourceSchema: &schema.Resource{
			Description: "A port group represents only port numbers, not the protocol. Port groups can be referenced for either TCP or UDP. It is recommended that TCP and UDP groups are created separately to avoid accidentally filtering unnecessary ports. Ranges of ports can be specified by using -, for more information see [VyOS Firewall doc](https://docs.vyos.io/en/latest/configuration/firewall/general.html#port-groups).",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceCreate(ctx, d, m, resourceFirewallPortGroup())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceRead(ctx, d, m, resourceFirewallPortGroup())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceUpdate(ctx, d, m, resourceFirewallPortGroup())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceDelete(ctx, d, m, resourceFirewallPortGroup())
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
				"name": {
					Description:      "Name for this port-group, _must be unique_.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: resourceInfo.ValidateDiagStringKeyField(),
				},
				"port": {
					Description: "A port name, port number, or port range. A port name can be any name defined in /etc/services. `portname | portnumber | startport-endport`",
					Type:        schema.TypeList,
					Required:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"description": {
					Description: "Group description text.",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "Managed by terraform",
				},
			},
		},
	}
}
