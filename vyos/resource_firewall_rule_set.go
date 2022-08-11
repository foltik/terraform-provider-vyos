package vyos

import (
	"context"
	"regexp"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceFirewallRuleSet() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "firewall name {{name}}",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          resourceInfo.DeleteTypeParameters,
		DeleteBlockerTemplates:  []string{"firewall name {{name}} rule"},
		ResourceSchema: &schema.Resource{
			Description:   "A rule-set is a named collection of firewall rules that can be applied to an interface or a zone, for more information see [VyOS Firewall doc](https://docs.vyos.io/en/latest/configuration/firewall/index.html#overview).",
			CreateContext: resourceFirewallRuleSetCreate,
			ReadContext:   resourceFirewallRuleSetRead,
			UpdateContext: resourceFirewallRuleSetUpdate,
			DeleteContext: resourceFirewallRuleSetDelete,
			Importer: &schema.ResourceImporter{
				StateContext: schema.ImportStatePassthroughContext,
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
							resourceInfo.ValidateStringKeyField(),
							validation.StringMatch(regexp.MustCompile("^[-A-Za-z]+$"), "Rule-set name can only contain letters."),
						),
					),
					ForceNew: true,
				},
				"description": {
					Description: "Rule-set description text.",
					Type:        schema.TypeString,
					Optional:    true,
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

func resourceFirewallRuleSetRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := p.client

	return resourceInfo.ResourceRead(ctx, d, client, resourceFirewallRuleSet())
}

func resourceFirewallRuleSetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := p.client

	return resourceInfo.ResourceCreate(ctx, d, client, resourceFirewallRuleSet())
}

func resourceFirewallRuleSetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := p.client

	return resourceInfo.ResourceUpdate(ctx, d, client, resourceFirewallRuleSet())
}

func resourceFirewallRuleSetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := p.client

	return resourceInfo.ResourceDelete(ctx, d, client, resourceFirewallRuleSet())
}
