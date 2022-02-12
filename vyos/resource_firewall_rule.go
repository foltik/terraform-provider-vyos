package vyos

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/foltik/vyos-client-go/client"
)

const (
	ResourceFirewallRuleKeyTemplate = "firewall name {{rule_set}} rule {{priority}}"
)

func resourceFirewallRule() *schema.Resource {
	return &schema.Resource{
		Description:   "Firewall rules with criteria matching that can be applied to an interface or a zone, for more information see [VyOS Firewall doc](https://docs.vyos.io/en/latest/configuration/firewall/index.html#matching-criteria).",
		CreateContext: resourceFirewallRuleCreate,
		ReadContext:   resourceFirewallRuleRead,
		UpdateContext: resourceFirewallRuleUpdate,
		DeleteContext: resourceFirewallRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The resource ID, same as the `priority`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"rule_set": {
				Description: "Rule set name this rule belongs to.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"priority": {
				Description: "_Must be unique within a rule set_. Data packets go through the rules based on the priority, from lowest to highest beginning at 0, at the first match the action of the rule will be applied and execution stops.",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"description": {
				Description: "Rule description text. Without a good description it can be hard to know why the rule exists.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"action": {
				Description:      "Action of this rule.",
				Type:             schema.TypeString,
				Default:          "drop",
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "reject", "drop"}, false)),
			},
			"log": {
				Description:      "Enable the logging of the this rule.",
				Type:             schema.TypeString,
				Default:          "disable",
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
			},
			"disable": {
				Description: "Disable this rule, but keep it in the config.",
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
			},
			"protocol": {
				Description: "Match a protocol criteria. A protocol number or a name which is defined in VyOS instances: `/etc/protocols` file. Special names are `all` for all protocols and `tcp_udp` for tcp and udp based packets. The `!` negate the selected protocol. `[<text> | <0-255> | all | tcp_udp]`",
				Type:        schema.TypeString,
				Default:     "tcp",
				Optional:    true,
			},

			"tcp": {
				Description: "TCP specific match criteria.",
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"flags": {
							Description: "Allowed values for TCP flags: `SYN`, `ACK`, `FIN`, `RST`, `URG`, `PSH`, `ALL` When specifying more than one flag, flags should be comma separated. The `!` negate the selected protocol.",
							Type:        schema.TypeString,
							Optional:    true,
						},
					},
				},
			},

			"state": {
				Description: "Match against the state of a packet.",
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"established": {
							Description:      "If this rule should match against the connection state `established`, valied values: `[enable | disable]`",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
						},
						"invalid": {
							Description:      "If this rule should match against the connection state `invalid`, valied values: `[enable | disable]`",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
						},
						"new": {
							Description:      "If this rule should match against the connection state `new`, valied values: `[enable | disable]`",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
						},
						"related": {
							Description:      "If this rule should match against the connection state `related`, valied values: `[enable | disable]`",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"enable", "disable"}, false)),
						},
					},
				},
			},

			"source": {
				Description: "Traffic source match criteria.",
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description: "Source address to match against, can be in format of: `[<x.x.x.x> | <x.x.x.x>-<x.x.x.x> | <x.x.x.x/x>]`. By starting the field with a `!` it will be a negative match.",
							Type:        schema.TypeString,
							Optional:    true,
							// Let VyOS do validation, these helers would not be compatible with the ! (not) marker
							//ValidateDiagFunc: validation.ToDiagFunc(validation.Any(validation.IsIPv4Address, validation.IsIPv4Range, validation.IsCIDR)),
						},
						"mac_address": {
							Description: "Source mac-address to match against. By starting the field with a `!` it will be a negative match.",
							Type:        schema.TypeString,
							Optional:    true,
							// Let VyOS do validation, these helers would not be compatible with the ! (not) marker
							//ValidateDiagFunc: validation.ToDiagFunc(validation.IsMACAddress),
						},
						"port": {
							Description: "A port can be set with port number in format: `[<xx> | <xx>-<xx>]` or a name which is here defined: `/etc/services`. Multiple source ports can be specified as a comma-separated list. The whole list can also be “negated” using `!`.",
							Type:        schema.TypeString,
							Optional:    true,
							// Let VyOS do validation, these helers would not be compatible with the ! (not) marker
							//ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 65535)),
						},
						"group": {
							Description: "Use a pre-defined group.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"address_group": {
										Description: "Address group name.",
										Type:        schema.TypeString,
										Optional:    true,
									},
									"network_group": {
										Description: "Network group name.",
										Type:        schema.TypeString,
										Optional:    true,
									},
									"port_group": {
										Description: "Port group name.",
										Type:        schema.TypeString,
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},

			"destination": {
				Description: "Traffic destination match criteria.",
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description: "Destination address to match against, can be in format of: `[<x.x.x.x> | <x.x.x.x>-<x.x.x.x> | <x.x.x.x/x>]`. By starting the field with a `!` it will be a negative match.",
							Type:        schema.TypeString,
							Optional:    true,
							// Let VyOS do validation, these helers would not be compatible with the ! (not) marker
							//ValidateDiagFunc: validation.ToDiagFunc(validation.Any(validation.IsIPv4Address, validation.IsIPv4Range, validation.IsCIDR)),
						},
						"mac_address": {
							Description: "Destination mac-address to match against. By starting the field with a `!` it will be a negative match.",
							Type:        schema.TypeString,
							Optional:    true,
							// Let VyOS do validation, these helers would not be compatible with the ! (not) marker
							//ValidateDiagFunc: validation.ToDiagFunc(validation.IsMACAddress),
						},
						"port": {
							Description: "A port can be set with port number in format: `[<xx> | <xx>-<xx>]` or a name which is here defined: `/etc/services`. Multiple source ports can be specified as a comma-separated list. The whole list can also be “negated” using `!`.",
							Type:        schema.TypeString,
							Optional:    true,
							// Let VyOS do validation, these helers would not be compatible with the ! (not) marker
							//ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 65535)),
						},
						"group": {
							Description: "Use a pre-defined group.",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"address_group": {
										Description: "Address group name.",
										Type:        schema.TypeString,
										Optional:    true,
									},
									"network_group": {
										Description: "Network group name.",
										Type:        schema.TypeString,
										Optional:    true,
									},
									"port_group": {
										Description: "Port group name.",
										Type:        schema.TypeString,
										Optional:    true,
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

func resourceFirewallRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.Client)
	return helperSchemaBasedConfigRead(ctx, client, ResourceFirewallRuleKeyTemplate, d, resourceFirewallRule().Schema)
}

func resourceFirewallRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.Client)
	return helperSchemaBasedConfigCreate(ctx, client, ResourceFirewallRuleKeyTemplate, d, resourceFirewallRule().Schema)
}

func resourceFirewallRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.Client)
	return helperSchemaBasedConfigUpdate(ctx, client, ResourceFirewallRuleKeyTemplate, d, resourceFirewallRule().Schema)
}

func resourceFirewallRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.Client)
	return helperSchemaBasedConfigDelete(ctx, client, ResourceFirewallRuleKeyTemplate, d, resourceFirewallRule().Schema)
}
