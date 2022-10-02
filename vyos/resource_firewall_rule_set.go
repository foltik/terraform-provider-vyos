package vyos

import (
	"context"
	"regexp"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceFirewallRuleSet() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "firewall name {{name}}",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates:  []string{"firewall name {{name}} rule"},
		ResourceSchema: &schema.Resource{
			Description: "A rule-set is a named collection of firewall rules that can be applied to an interface or a zone, for more information see [VyOS Firewall doc](https://docs.vyos.io/en/latest/configuration/firewall/index.html#overview).",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreate(ctx, d, m, resourceFirewallRuleSet())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceRead(ctx, d, m, resourceFirewallRuleSet())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdate(ctx, d, m, resourceFirewallRuleSet())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDelete(ctx, d, m, resourceFirewallRuleSet())
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
					Description: "The resource ID, same as the `name`",
					Type:        schema.TypeString,
					Computed:    true,
				},
				"name": {
					Description: "Name for this rule-set, _must be unique_.",
					Type:        schema.TypeString,
					Required:    true,
					ValidateDiagFunc: validation.ToDiagFunc(
						validation.All(
							schemabased.ValidateStringKeyField(),
							validation.StringMatch(regexp.MustCompile("^[-A-Za-z]+$"), "Rule-set name can only contain letters and hyphens."),
						),
					),
					ForceNew: true,
				},
				"description": {
					Description: "Group description text.",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "Managed by terraform",
				},
				"default_action": {
					Description:      "Default action of the rule-set if no rule matched a packet criteria.",
					Type:             schema.TypeString,
					Default:          "drop",
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "reject", "drop"}, false)),
				},
				"enable_default_log": {
					Description: "Enable the logging of the default action.",
					Type:        schema.TypeBool,
					Default:     false,
					Optional:    true,
				},
			},
		},
	}
}
