package vyos

import (
	"context"
	"time"

	resourceInfo "github.com/foltik/terraform-provider-vyos/vyos/helper/resource-info"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceInfoVrrpGroup() *resourceInfo.ResourceInfo {
	return &resourceInfo.ResourceInfo{
		KeyTemplate:             "high-availability vrrp group {{name}}",
		CreateRequiredTemplates: []string{},
		DeleteStrategy:          resourceInfo.DeleteTypeResource,
		DeleteBlockerTemplates:  []string{},
		ResourceSchema: &schema.Resource{
			Description: "[VRRP](https://docs.vyos.io/en/latest/configuration/highavailability/index.html) (Virtual Router Redundancy Protocol) provides active/backup redundancy for routers. " +
				"Every VRRP router has a physical IP/IPv6 address, and a virtual address. " +
				"On startup, routers elect the master, and the router with the highest priority becomes the master and assigns the virtual address to its interface. " +
				"All routers with lower priorities become backup routers. The master then starts sending keepalive packets to notify other routers that it’s available. " +
				"If the master fails and stops sending keepalive packets, the router with the next highest priority becomes the new master and takes over the virtual address." +
				"VRRP keepalive packets use multicast, and VRRP setups are limited to a single datalink layer segment. You can setup multiple VRRP groups (also called virtual routers). " +
				"Virtual routers are identified by a VRID (Virtual Router IDentifier). " +
				"If you setup multiple groups on the same interface, their VRIDs must be unique, but it’s possible (even if not recommended for readability reasons) to use duplicate VRIDs on different interfaces.",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceCreate(ctx, d, m, resourceInfoVrrpGroup())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceRead(ctx, d, m, resourceInfoVrrpGroup())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceUpdate(ctx, d, m, resourceInfoVrrpGroup())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return resourceInfo.ResourceDelete(ctx, d, m, resourceInfoVrrpGroup())
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
					Description:      "Name of the VRRP group.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: resourceInfo.ValidateDiagStringKeyField(),
				},
				"interface": {
					Description: "Network interface.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"vrid": {
					Description:      "Virtual router identifier.",
					Type:             schema.TypeInt,
					Required:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 255)),
				},
				"virtual_address": {
					Description: "Virtual address in CIDR form (IPv4 or IPv6, but they must not be mixed in one group).",
					Type:        schema.TypeList,
					Required:    true,
					MinItems:    1,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"priority": {
					Description:      "Router priority.",
					Type:             schema.TypeInt,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 255)),
					Default:          100,
				},
				"disable": {
					Description: "A disabled group will be removed from the VRRP process and your router will not participate in VRRP for that VRID. It will disappear from operational mode commands output, rather than enter the backup state.",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"excluded_address": {
					Description: "Exclude IP addresses from VRRP packets. This option is used when you want to set IPv4 + IPv6 addresses on the same virtual interface or when used more than 20 IP addresses.",
					Type:        schema.TypeList,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"no_preempt": {
					Description: "VRRP can use two modes: preemptive and non-preemptive. " +
						"In the preemptive mode, if a router with a higher priority fails and then comes back, routers with lower priority will give up their master status. " +
						"In non-preemptive mode, the newly elected master will keep the master status and the virtual address indefinitely." +
						"By default VRRP uses preemption.",
					Type:         schema.TypeBool,
					Optional:     true,
					RequiredWith: []string{"preempt_delay"},
				},
				"preempt_delay": {
					Description:      "The time interval for preemption with the “preempt-delay” option.",
					Type:             schema.TypeInt,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 1000)),
				},
				"track": {
					Description: "Track option to track non VRRP interface states. VRRP changes status to FAULT if one of the track interfaces in state down.",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"interface": {
								Description: "Interfaces to track.",
								Type:        schema.TypeList,
								Required:    true,
								Elem: &schema.Schema{
									Type: schema.TypeString,
								},
							},
							"exclude_vrrp_interface": {
								Description: "Ignore VRRP main interface faults.",
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
							},
						},
					},
				},
				"peer_address": {
					Description: "Unicast VRRP peer address. By default VRRP uses multicast packets. If your network does not support multicast for whatever reason, you can make VRRP use unicast communication instead.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"hello_source_address": {
					Description:  "VRRP hello source address.",
					Type:         schema.TypeString,
					Optional:     true,
					RequiredWith: []string{"peer_address"},
				},
				"rfc3768_compatibility": {
					Description: "RFC 3768 defines a virtual MAC address to each VRRP virtual router. " +
						"This virtual router MAC address will be used as the source in all periodic VRRP messages sent by the active node. " +
						"When the rfc3768-compatibility option is set, a new VRRP interface is created, to which the MAC address and the virtual IP address is automatically assigned.",
					Type:     schema.TypeBool,
					Optional: true,
				},
				"health_check": {
					Description: "Health check scripts execute custom checks in addition to the master router reachability.",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"script": {
								Description: "Health check script file path on VyOS box.",
								Type:        schema.TypeString,
								Required:    true,
							},
							"interval": {
								Description: "Health check execution interval in seconds",
								Type:        schema.TypeInt,
								Optional:    true,
								Default:     60,
							},
							"failure_count": {
								Description: "Health check failure count required for transition to fault.",
								Type:        schema.TypeInt,
								Optional:    true,
								Default:     3,
							},
						},
					},
				},
				"transition_script": {
					Description: "Transition scripts can help you implement various fixups, such as starting and stopping services, or even modifying the VyOS config on VRRP transition.",
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"backup": {
								Description: "Script to run on VRRP state transition to backup.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"fault": {
								Description: "Script to run on VRRP state transition to fault.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"master": {
								Description: "Script to run on VRRP state transition to master.",
								Type:        schema.TypeString,
								Optional:    true,
							},
							"stop": {
								Description: "Script to run on VRRP state transition to stop.",
								Type:        schema.TypeString,
								Optional:    true,
							},
						},
					},
				},
			},
		},
	}
}
