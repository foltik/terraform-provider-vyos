package vyos

import (
	"context"
	"regexp"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	ResourceFirewallRuleSetKeyTemplate = "firewall name {{name}}"
)

func resourceFirewallRuleSet() *schema.Resource {
	return &schema.Resource{
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
				Description:      "Name for this rule-set, _must be unique_.",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile("^[-A-Za-z]+$"), "Rule-set name can only contain letters and hyphens.")),
				ForceNew:         true,
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
	}
}

func resourceFirewallRuleSetRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {

	// Client
	p := m.(*ProviderClass)
	client := *p.client

	// Key template
	resource_key_template := ResourceFirewallRuleSetKeyTemplate

	// Schema
	resource_schema := resourceFirewallRuleSet()

	return resource.ResourceRead(ctx, d, resource_key_template, resource_schema, &client)

}

func resourceFirewallRuleSetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	// Key template
	resource_key_template := ResourceFirewallRuleSetKeyTemplate

	// Schema
	resource_schema := resourceFirewallRuleSet()

	return resource.ResourceCreate(ctx, d, resource_key_template, resource_schema, &client)
}

func resourceFirewallRuleSetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	// Key template
	resource_key_template := ResourceFirewallRuleSetKeyTemplate

	// Schema
	resource_schema := resourceFirewallRuleSet()

	return resource.ResourceUpdate(ctx, d, resource_key_template, resource_schema, &client)
}

func resourceFirewallRuleSetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	// Client
	p := m.(*ProviderClass)
	client := *p.client

	// Key template
	resource_key_template := ResourceFirewallRuleSetKeyTemplate

	// Schema
	resource_schema := resourceFirewallRuleSet()

	return resource.ResourceDelete(ctx, d, resource_key_template, resource_schema, &client)
}
