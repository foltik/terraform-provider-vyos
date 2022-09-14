package vyos

import (
	"context"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceInfoFirewallService() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "firewall",
		CreateRequiredTemplates: []string{},
		DeleteStrategy:          resourceInfo.DeleteTypeParameters,
		DeleteBlockerTemplates:  []string{},
		StaticId:                "firewallService",
		ResourceSchema: &schema.Resource{
			Description: "[Firewall Global Config](https://docs.vyos.io/en/latest/configuration/firewall/index.html). " +
				"**This is a global config, having more than one of this resource will casue continues diffs to occur.**",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceCreate(ctx, d, m, resourceInfoFirewallService())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceRead(ctx, d, m, resourceInfoFirewallService())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceUpdate(ctx, d, m, resourceInfoFirewallService())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceDelete(ctx, d, m, resourceInfoFirewallService())
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
				"all_ping": {
					Description:      "By default, when VyOS receives an ICMP echo request packet destined for itself, it will answer with an ICMP echo reply, unless you avoid it through its firewall.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"broadcast_ping": {
					Description:      "This setting enable or disable the response of icmp broadcast messages.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"ip_src_route": {
					Description:      "This setting handle if VyOS accept packets with a source route option.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"ipv6_src_route": {
					Description:      "This setting handle if VyOS accept packets with a source route option.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"receive_redirects": {
					Description:      "enable or disable of ICMPv4 or ICMPv6 redirect messages accepted by VyOS.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"ipv6_receive_redirects": {
					Description:      "enable or disable of ICMPv4 or ICMPv6 redirect messages accepted by VyOS.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"send_redirects": {
					Description:      "enable or disable ICMPv4 redirect messages send by VyOS",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"log_martians": {
					Description:      "enable or disable the logging of martian IPv4 packets.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"source_validation": {
					Description:      "Set the IPv4 source validation mode.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"strict", "loose", "disable"}, false)),
				},
				"syn_cookies": {
					Description:      "Enable or Disable if VyOS use IPv4 TCP SYN Cookies.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},
				"twa_hazards_protection": {
					Description:      "Enable or Disable VyOS to be [RFC 1337](https://datatracker.ietf.org/doc/html/rfc1337.html) conform.",
					Type:             schema.TypeString,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
				},

				"state_policy": {
					Description: "Global state policy settings",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"established": {
								Description: "established connections",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"action": {
											Description:      "Action top take",
											Type:             schema.TypeString,
											Required:         true,
											ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "drop", "reject"}, false)),
										},
										"log": {
											Description: "Logging configuration",
											Type:        schema.TypeList,
											Optional:    true,
											MaxItems:    1,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"enable": {
														Description: "Enable logging",
														Type:        schema.TypeBool,
														Required:    true,
													},
												},
											},
										},
									},
								},
							},
							"invalid": {
								Description: "invalid connections",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"action": {
											Description:      "Action top take",
											Type:             schema.TypeString,
											Required:         true,
											ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "drop", "reject"}, false)),
										},
										"log": {
											Description: "Logging configuration",
											Type:        schema.TypeList,
											Optional:    true,
											MaxItems:    1,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"enable": {
														Description: "Enable logging",
														Type:        schema.TypeBool,
														Required:    true,
													},
												},
											},
										},
									},
								},
							},
							"related": {
								Description: "related connections",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"action": {
											Description:      "Action top take",
											Type:             schema.TypeString,
											Required:         true,
											ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "drop", "reject"}, false)),
										},
										"log": {
											Description: "Logging configuration",
											Type:        schema.TypeList,
											Optional:    true,
											MaxItems:    1,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"enable": {
														Description: "Enable logging",
														Type:        schema.TypeBool,
														Required:    true,
													},
												},
											},
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
