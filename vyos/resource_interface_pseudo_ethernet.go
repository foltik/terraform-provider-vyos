package vyos

import (
	"context"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceInterfacePseudoEthernet() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "interfaces pseudo-ethernet {{name}}",
		CreateRequiredTemplates: nil,
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates: []string{
			"interfaces pseudo-ethernet {{name}} vif",
			"interfaces pseudo-ethernet {{name}} vif-s",
		},
		ResourceSchema: &schema.Resource{
			Description: "MACVLAN - Pseudo Ethernet. [VyOS doc](https://docs.vyos.io/en/latest/configuration/interfaces/pseudo-ethernet.html).",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreate(ctx, d, m, resourceInterfacePseudoEthernet())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceRead(ctx, d, m, resourceInterfacePseudoEthernet())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdate(ctx, d, m, resourceInterfacePseudoEthernet())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDelete(ctx, d, m, resourceInterfacePseudoEthernet())
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
					Description:      "Ethernet interface name. Eg: peth0",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: schemabased.ValidateDiagStringKeyField(),
				},
				"source_interface": {
					Description: "Physical interface the traffic will go through. Eg: `eth0` or for vif 20 on eht0: `eth0.20`",
					Type:        schema.TypeString,
					Required:    true,
				},
				"address": {
					Description: "`<x.x.x.x/x | h:h:h:h:h:h:h:h/x | dhcp | dhcpv6>` Configure interface <interface> with one or more interface addresses.",
					Optional:    true,
					Type:        schema.TypeList,
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
				"dhcp_options": {
					Description: "DHCP client settings/options",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"client_id": {
								Description: "Identifier used by client to identify itself to the DHCP server. [RFC 2131](https://datatracker.ietf.org/doc/html/rfc2131.html) states: The client MAY choose to explicitly provide the identifier through the ‘client identifier’ option. If the client supplies a ‘client identifier’, the client MUST use the same ‘client identifier’ in all subsequent messages, and the server MUST use that identifier to identify the client.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"default_route_distance": {
								Description:      "Distance for the default route from DHCP server.",
								Type:             schema.TypeInt,
								Optional:         true,
								Default:          210,
								ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 255)),
							},
							"host_name": {
								Description: "Override system host-name sent to DHCP server.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"no_default_route": {
								Description: "Do not request routers from DHCP server.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"reject": {
								Description: "`<x.x.x.x | x.x.x.x/x>` IP addresses or subnets from which to reject DHCP leases.",
								Type:        schema.TypeList,
								Elem: &schema.Schema{
									Type: schema.TypeString,
								},
								Optional: true,
							},
							"vendor_class_id": {
								Description: "Identify the vendor client type to the DHCP server.",
								Type:        schema.TypeString,
								Optional:    true,
							},
						},
					},
				},
				"dhcpv6_options": {
					Description: "DHCPv6 client settings/options",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"duid": {
								Description: "DHCP unique identifier (DUID) to be sent by dhcpv6 client.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"parameters_only": {
								Description: "Acquire only config parameters, no address.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							// TODO can pd be supported? need extension of feature for list of complex schemas, and option to have extra key parameter, eg 0, 1, 2 in this case.
							"rapid_commit": {
								Description: "Wait for immediate reply instead of advertisements.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"temporary": {
								Description: "IPv6 temporary address. Request only a temporary address and not form an IA_NA (Identity Association for Non-temporary Addresses) partnership.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
						},
					},
				},
				"disable": {
					Description: "Administratively disable interface.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"disable_link_detect": {
					Description: "Ignore link state changes.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"firewall": {
					Description: "Firewall options. [Interface base firewall](https://docs.vyos.io/en/latest/configuration/firewall/general.html#applying-a-rule-set-to-an-interface)",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"in": {
								Description: "Ruleset for forwarded packets on inbound interface.",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"ipv6_name": {
											Description: "Inbound IPv6 firewall ruleset name for interface.",
											Type:        schema.TypeString,
											Optional:    true,
										},
										"name": {
											Description: "Inbound IPv4 firewall ruleset name for interface.",
											Type:        schema.TypeString,
											Optional:    true,
										},
									},
								},
							},
							"local": {
								Description: "Ruleset for packets destined for this router.",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"ipv6_name": {
											Description: "Inbound IPv6 firewall ruleset name for interface.",
											Type:        schema.TypeString,
											Optional:    true,
										},
										"name": {
											Description: "Inbound IPv4 firewall ruleset name for interface.",
											Type:        schema.TypeString,
											Optional:    true,
										},
									},
								},
							},
							"out": {
								Description: "Ruleset for forwarded packets on outbound interface.",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"ipv6_name": {
											Description: "Inbound IPv6 firewall ruleset name for interface.",
											Type:        schema.TypeString,
											Optional:    true,
										},
										"name": {
											Description: "Inbound IPv4 firewall ruleset name for interface.",
											Type:        schema.TypeString,
											Optional:    true,
										},
									},
								},
							},
						},
					},
				},
				"ip": {
					Description: "IPv4 routing parameters.",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"adjust_mss": {
								Description:      "`<clamp-mss-to-pmtu | 500-65535>`. _Automatically sets the MSS to the proper value_, or _TCP Maximum segment size in bytes_",
								Type:             schema.TypeString,
								Optional:         true,
								ValidateDiagFunc: validation.ToDiagFunc(validation.Any(validation.StringInSlice([]string{"clamp-mss-to-pmtu"}, false), validation.IntBetween(500, 65535))),
							},
							"arp_cache_timeout": {
								Description:      "ARP cache entry timeout in seconds.",
								Type:             schema.TypeInt,
								Optional:         true,
								Default:          30,
								ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 86400)),
							},
							"disable_arp_filter": {
								Description: "Disable ARP filter on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"disable_forwarding": {
								Description: "Disable IP forwarding on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"enable_arp_accept": {
								Description: "Enable ARP accept on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"enable_arp_announce": {
								Description: "Enable ARP announce on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"enable_arp_ignore": {
								Description: "Enable ARP ignore on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"enable_proxy_arp": {
								Description: "Enable proxy-arp on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"proxy_arp_pvlan": {
								Description: "Enable private VLAN proxy ARP on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
							"source_validation": {
								Description: "`<strict | loose | disable>`. Enable policy for source validation by reversed path, as specified in RFC [3704](https://datatracker.ietf.org/doc/html/rfc3704.html). " +
									"Current recommended practice in RFC 3704 is to enable strict mode to prevent IP spoofing from DDos attacks. " +
									"If using asymmetric routing or other complicated routing, then loose mode is recommended. " +
									"[VyOS doc](https://docs.vyos.io/en/latest/configuration/interfaces/ethernet.html#cfgcmd-set-interfaces-ethernet-interface-ip-source-validation-strict-loose-disable)",
								Type:             schema.TypeString,
								Optional:         true,
								ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"strict", "loose", "disable"}, false)),
							},
						},
					},
				},
				"ipv6": {
					Description: "ipv6 routing parameters.",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Description: "IPv6 address configuration modes",
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"autoconf": {
											Description: "Enable acquisition of IPv6 address using stateless autoconfig (SLAAC [RFC 4862](https://datatracker.ietf.org/doc/html/rfc4862.html)). __This method automatically disables IPv6 traffic forwarding on the interface in question.__",
											Type:        schema.TypeBool,
											Optional:    true,
											Default:     false,
										},
										"eui64": {
											Description: "`<h:h:h:h:h:h:h:h/64>` Prefix for IPv6 address with MAC-based EUI-64",
											Type:        schema.TypeList,
											Optional:    true,
											Elem: &schema.Schema{
												Type: schema.TypeString,
											},
										},
										"no_default_link_local": {
											Description: "Remove the default link-local address from the interface",
											Type:        schema.TypeBool,
											Optional:    true,
											Default:     false,
										},
									},
								},
							},
							"adjust_mss": {
								Description:      "`<clamp-mss-to-pmtu | 500-65535>`. _Automatically sets the MSS to the proper value_, or _TCP Maximum segment size in bytes_",
								Type:             schema.TypeString,
								Optional:         true,
								ValidateDiagFunc: validation.ToDiagFunc(validation.Any(validation.StringInSlice([]string{"clamp-mss-to-pmtu"}, false), validation.IntBetween(500, 65535))),
							},
							"disable_forwarding": {
								Description: "Disable IP forwarding on this interface.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
						},
					},
				},
				"mac": {
					Description: "Configure user defined MAC address on given <interface>. Media Access Control (MAC) address.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"mode": {
					Description:      "Receive mode. `private`: No communication with other pseudo-devices. `vepa`: Virtual Ethernet Port Aggregator reflective relay. `bridge`: Simple bridge between pseudo-devices. `passthru`: Promicious mode passthrough of underlying device",
					Type:             schema.TypeString,
					Optional:         true,
					Default:          "private",
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"private", "vepa", "bridge", "passthru"}, false)),
				},
				"mtu": {
					Description:      "Maximum Transmission Unit (MTU).",
					Type:             schema.TypeInt,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(68, 16_000)),
				},
				"policy": {
					Description: "Policy route options",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"ipv6_route": {
								Description: "IPv6 policy route ruleset for interface",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"route": {
								Description: "IPv4 policy route ruleset for interface",
								Type:        schema.TypeString,
								Optional:    true,
							},
						},
					},
				},
				"redirect": {
					Description: "Incoming packet redirection destination, interface for packet redirection",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"traffic_policy": {
					Description: "Traffic-policy for interface",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"in": {
								Description: "Ingress traffic policy for interface",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"out": {
								Description: "Egress traffic policy for interface",
								Type:        schema.TypeString,
								Optional:    true,
							},
						},
					},
				},
				"vrf": {
					Description: "VRF instance name",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"xdp": {
					Description: "Enable eXpress Data Path",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
			},
		},
	}
}