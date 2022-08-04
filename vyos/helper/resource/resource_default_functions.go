package resource

import (
	"context"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/config"
	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceRead(ctx context.Context, d *schema.ResourceData, resource_key_template string, resource_schema *schema.Resource, client *client.Client) (diags diag.Diagnostics) {
	logger.Log("INFO", "Reading resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resource_key_template}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resource_schema, client)
	diags = append(diags, diags_ret...)

	for parameter, value := range vyos_config.MarshalTerraform() {
		logger.Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
		d.Set(parameter, value)
	}

	d.SetId(resouce_id)

	return diags
}

func ResourceCreate(ctx context.Context, d *schema.ResourceData, resource_key_template string, resource_schema *schema.Resource, client *client.Client) (diags diag.Diagnostics) {
	logger.Log("INFO", "Creating resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resource_key_template}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Create terraform config struct
	terraform_key := key
	terraform_config, diags_ret := config.NewConfigFromTerraform(ctx, &terraform_key, resource_schema, d)
	diags = append(diags, diags_ret...)

	for _, field := range config.GetKeyFields(key_template) {
		terraform_config.PopChild(field)
		logger.Log("INFO", "Removed key field from config object: %v", field)
	}

	err := client.Config.SetTree(ctx, terraform_config.MarshalVyos())

	if err != nil {
		logger.Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret = ResourceRead(ctx, d, resource_key_template, resource_schema, client)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceUpdate(ctx context.Context, d *schema.ResourceData, resource_key_template string, resource_schema *schema.Resource, client *client.Client) (diags diag.Diagnostics) {
	logger.Log("INFO", "Updating resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resource_key_template}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Create terraform config struct
	terraform_key := key
	terraform_config, diags_ret := config.NewConfigFromTerraform(ctx, &terraform_key, resource_schema, d)
	diags = append(diags, diags_ret...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resource_schema, client)
	diags = append(diags, diags_ret...)

	// Remove fields/parameters only internal to terraform so they are not part of the comparison
	terraform_config.PopChild("id")
	for _, field := range config.GetKeyFields(key_template) {
		terraform_config.PopChild(field)
		logger.Log("INFO", "Removed key field from config object: %v", field)
	}

	// Find changes
	changed, deleted := terraform_config.GetDifference(vyos_config)

	// Apply changed parameters
	if changed != nil {
		changed_vyos := changed.MarshalVyos()
		logger.Log("INFO", "Changes detected: %#v", changed_vyos)
		err := client.Config.SetTree(ctx, changed_vyos)
		if err != nil {
			logger.Log("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Remove deleted parameters
	if deleted != nil {
		deleted_vyos := deleted.MarshalVyos()
		logger.Log("INFO", "Deleted detected: %#v", deleted_vyos)
		err := client.Config.DeleteTree(ctx, deleted_vyos)
		if err != nil {
			logger.Log("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Refresh tf state after update
	diags_ret = ResourceRead(ctx, d, resource_key_template, resource_schema, client)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceDelete(ctx context.Context, d *schema.ResourceData, resource_key_template string, resource_schema *schema.Resource, client *client.Client) (diags diag.Diagnostics) {
	logger.Log("INFO", "Deleting resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resource_key_template}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Generate config object from VyOS (only used for logs)
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resource_schema, client)
	diags = append(diags, diags_ret...)

	// Remove resource
	delete_config := vyos_config.MarshalVyos()
	logger.Log("INFO", "Deleting key: '%s' vyos resource: %#v", key_string, delete_config)
	err := client.Config.Delete(ctx, key_string)
	if err != nil {
		logger.Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret = ResourceRead(ctx, d, resource_key_template, resource_schema, client)
	diags = append(diags, diags_ret...)

	return diags
}
