package vyos

import (
	"context"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/schemabased"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceInfoTftpService() *schemabased.ResourceInfo {
	return &schemabased.ResourceInfo{
		KeyTemplate:             "service tftp-server",
		CreateRequiredTemplates: []string{},
		DeleteStrategy:          schemabased.DeleteTypeResource,
		DeleteBlockerTemplates:  []string{},
		StaticId:                "global",
		ResourceSchema: &schema.Resource{
			Description: "[Trivial File Transfer Protocol (TFTP) server](https://docs.vyos.io/en/latest/configuration/firewall/general.html). " +
				"**This is a global config, having more than one of this resource will cause continues diffs to occur.**",
			CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceCreateGlobal(ctx, d, m, resourceInfoTftpService())
			},
			ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceReadGlobal(ctx, d, m, resourceInfoTftpService())
			},
			UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceUpdateGlobal(ctx, d, m, resourceInfoTftpService())
			},
			DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
				return schemabased.ResourceDeleteGlobal(ctx, d, m, resourceInfoTftpService())
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
				"allow_upload": {
					Description: "Allow TFTP file uploads",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
				"directory": {
					Description: "Folder containing files served by TFTP.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"listen_address": {
					Description: "`<x.x.x.x | h:h:h:h:h:h:h:h>` IPv4/IPv6 address to listen for incoming connections. This can not be empty at any point, due to quirks in the provider if you have to change all adresses at the same time, append the new address, run apply before removing the old one.",
					Type:        schema.TypeList,
					Required:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"port": {
					Description:      "Port number used by connection",
					Type:             schema.TypeInt,
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 65535)),
				},
			},
		},
	}
}
