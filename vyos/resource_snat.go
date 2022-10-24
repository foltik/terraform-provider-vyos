package vyos

import (
	"context"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSourceNat() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "nat source rule {{rule_priority}}",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates:  nil,
		ResourceSchema: &schema.Resource{
			Description: "Source NAT settings. [VyOS doc](https://docs.vyos.io/en/latest/configuration/nat/nat44.html).",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreate(ctx, d, m, resourceSourceNat())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceRead(ctx, d, m, resourceSourceNat())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdate(ctx, d, m, resourceSourceNat())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDelete(ctx, d, m, resourceSourceNat())
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
				"rule_priority": {
					Description:      "Rules are numbered and evaluated by the underlying OS in numerical order",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: schemabased.ValidateDiagStringKeyField(),
				},
				"outbound_interface": {
					Description: "Outbound interface of NAT traffic.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"description": {
					Description: "Group description text.",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "Managed by terraform",
				},
				"source": {
					Description: "NAT source parameters",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Description: "`< x.x.x.x | x.x.x.x/x | x.x.x.x-x.x.x.x >` IP address, subnet, or range. If the adress/cidr/range is prefixed with a `!` it becomes a negative match.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"port": {
								Description: "Port. Multiple destination ports can be specified as a comma-separated list. The whole list can also be negated using `!`. For example: `!22,telnet,http,123,1001-1005`",
								Type:        schema.TypeString,
								Optional:    true,
							},
						},
					},
				},
				"destination": {
					Description: "NAT destination parameters",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Description: "`< x.x.x.x | x.x.x.x/x | x.x.x.x-x.x.x.x >` IP address, subnet, or range. If the adress/cidr/range is prefixed with a `!` it becomes a negative match.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"port": {
								Description: "Port. Multiple destination ports can be specified as a comma-separated list. The whole list can also be negated using `!`. For example: `!22,telnet,http,123,1001-1005`",
								Type:        schema.TypeString,
								Optional:    true,
							},
						},
					},
				},
				"disable": {
					Description: "Temporary disable.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"exclude": {
					Description: "Exclude packets matching this rule from NAT.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"log": {
					Description: "NAT rule logging.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"protocol": {
					Description: "Protocol to NAT. IP protocol name or number. [See traffic filters](https://docs.vyos.io/en/latest/configuration/nat/nat44.html#traffic-filters)",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "all",
				},
				"translation": {
					Description: "Inside NAT IP",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Description: "`< masquerade | x.x.x.x | x.x.x.x/x | x.x.x.x-x.x.x.x >` IP address, subnet, or range.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"port": {
								Description: "`< x | x-x >` Port or portrange.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"options": {
								Description: "Translation options",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"address_mapping": {
											Description:      "Address mapping options. `persistent`: Gives a client the same source or destination-address for each connection. `random`: Random source or destination address allocation for each connection",
											Type:             schema.TypeString,
											Optional:         true,
											ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"persistent", "random"}, false)),
											Default:          "random",
										},
										"port_mapping": {
											Description:      "Port mapping options. `random`: Randomize source port mapping. `fully-random`: Full port randomization. `none`: Do not apply port randomization.",
											Type:             schema.TypeString,
											Optional:         true,
											ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"random", "fully-random", "none"}, false)),
											Default:          "none",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
