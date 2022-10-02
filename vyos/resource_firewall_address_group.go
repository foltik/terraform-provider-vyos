package vyos

import (
	"context"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirewallAddressGroup() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "firewall group address-group {{name}}",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates:  nil, //? TODO can we support firewall rules using the group is blocking? do we want to?
		ResourceSchema: &schema.Resource{
			Description: "In an address group a single IP address or IP address ranges are defined., for more information see [VyOS Firewall doc](https://docs.vyos.io/en/latest/configuration/firewall/general.html#address-groups).",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreate(ctx, d, m, resourceFirewallAddressGroup())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceRead(ctx, d, m, resourceFirewallAddressGroup())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdate(ctx, d, m, resourceFirewallAddressGroup())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDelete(ctx, d, m, resourceFirewallAddressGroup())
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
					Description:      "Name for this address-group, _must be unique_.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: schemabased.ValidateDiagStringKeyField(),
				},
				"address": {
					Description: "IP address, or address range. `[address | address-address]`",
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
