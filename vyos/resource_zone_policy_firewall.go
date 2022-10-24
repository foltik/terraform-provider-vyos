package vyos

import (
	"context"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceZonePolicyFirewall() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "zone-policy zone {{to_zone}} from {{from_zone}} firewall",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates:  nil,
		ResourceSchema: &schema.Resource{
			Description: "Set firewall rule-set for zone based firewalling.",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreate(ctx, d, m, resourceZonePolicyFirewall())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceRead(ctx, d, m, resourceZonePolicyFirewall())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdate(ctx, d, m, resourceZonePolicyFirewall())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDelete(ctx, d, m, resourceZonePolicyFirewall())
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
				"to_zone": {
					Description:      "Zone to apply firewall rule-set on.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: schemabased.ValidateDiagStringKeyField(),
				},
				"from_zone": {
					Description:      "Zone from which to filter traffic.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: schemabased.ValidateDiagStringKeyField(),
				},
				"name": {
					Description:  "IPv4 firewall rule-set.",
					Type:         schema.TypeString,
					Optional:     true,
					AtLeastOneOf: []string{"name", "ipv6_name"},
				},
				"ipv6_name": {
					Description: "IPv6 firewall rule-set.",
					Type:        schema.TypeString,
					Optional:    true,
				},
			},
		},
	}
}
