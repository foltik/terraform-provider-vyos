package vyos

import (
	"context"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInfoDhcpServerSubnetAddressPool() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "service dhcp-server shared-network-name {{shared_network_name}} subnet {{subnet}} range {{pool}}",
		CreateRequiredTemplates: []string{"service dhcp-server shared-network-name {{shared_network_name}} subnet {{subnet}}"},
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates:  []string{},
		ResourceSchema: &schema.Resource{
			Description: "[Create DHCP address range](https://docs.vyos.io/en/latest/configuration/service/dhcp-server.html#cfgcmd-set-service-dhcp-server-shared-network-name-name-subnet-subnet-range-n-start-address). DHCP leases are taken from this pool.",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreate(ctx, d, m, resourceInfoDhcpServerSubnetAddressPool())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceRead(ctx, d, m, resourceInfoDhcpServerSubnetAddressPool())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdate(ctx, d, m, resourceInfoDhcpServerSubnetAddressPool())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDelete(ctx, d, m, resourceInfoDhcpServerSubnetAddressPool())
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
				"shared_network_name": {
					Description: "Name of the DHCP server network.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"subnet": {
					Description: "Name of the DHCP subnet.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"pool": {
					Description:      "Name of the address pool.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: schemabased.ValidateDiagStringKeyField(),
				},
				"start": {
					Description: "The pool starts at `address`.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"stop": {
					Description: "The pool stops with `address`.",
					Type:        schema.TypeString,
					Required:    true,
				},
			},
		},
	}
}
