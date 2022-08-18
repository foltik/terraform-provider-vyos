package resourceInfo

import (
	"context"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/config"
	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceReadGlobal(ctx context.Context, d *schema.ResourceData, client *client.Client, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Reading resource, global type")

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := config.ConfigKey{Key: key_string}
	resouce_id := resourceInfo.StaticId

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diags_ret...)

	for parameter, value := range vyos_config.MarshalTerraform() {
		logger.Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
		d.Set(parameter, value)
	}

	d.SetId(resouce_id)

	return diags
}

func ResourceRead(ctx context.Context, d *schema.ResourceData, client *client.Client, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Reading resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diags_ret...)

	for parameter, value := range vyos_config.MarshalTerraform() {
		logger.Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
		d.Set(parameter, value)
	}

	d.SetId(resouce_id)

	return diags
}

func ResourceCreate(ctx context.Context, d *schema.ResourceData, client *client.Client, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Creating resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Check for required resources before create
	for _, reqKeyTemplateStr := range resourceInfo.CreateRequiredTemplates {
		reqKeyTemplate := config.ConfigKeyTemplate{Template: reqKeyTemplateStr}
		resouce_id := config.FormatResourceId(reqKeyTemplate, d)
		reqKey := config.FormatKey(reqKeyTemplate, resouce_id, d)
		val, err := client.Config.Show(ctx, reqKey)
		if err != nil {
			return diag.FromErr(err)
		}
		if val != nil {
			return diag.Errorf("Configuration '%s' already exists with value '%s' set, try a resource import instead.", key, val)
		}
	}

	// Create terraform config struct
	terraform_key := key
	terraform_config, diags_ret := config.NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diags_ret...)

	for _, field := range config.GetKeyFields(key_template) {
		terraform_config.PopChild(field)
		logger.Log("INFO", "Removed key field from config object: %v", field)
	}

	path, value := terraform_config.MarshalVyos()
	err := client.Config.Set(ctx, path, value)

	if err != nil {
		logger.Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret = resourceInfo.ResourceSchema.ReadContext(ctx, d, client)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceUpdate(ctx context.Context, d *schema.ResourceData, client *client.Client, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Updating resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Create terraform config struct
	terraform_key := key
	terraform_config, diags_ret := config.NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diags_ret...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diags_ret...)

	// Remove fields/parameters only internal to terraform so they are not part of the comparison
	terraform_config.PopChild("id")
	for _, field := range config.GetKeyFields(key_template) {
		terraform_config.PopChild(field)
		logger.Log("INFO", "Removed key field from config object: %v", field)
	}

	// Find config changes
	changed, deleted := terraform_config.GetDifference(vyos_config)

	// Remove deleted parameters
	if deleted != nil {
		deleted_path, deleted_config := deleted.MarshalVyos()
		logger.Log("INFO", "Deleted detected: %#v", deleted_config)
		err := client.Config.Delete(ctx, deleted_path, deleted_config)
		if err != nil {
			logger.Log("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Apply changed parameters
	if changed != nil {
		changed_path, changed_config := changed.MarshalVyos()
		logger.Log("INFO", "Changes detected: %#v", changed_config)
		err := client.Config.Set(ctx, changed_path, changed_config)
		if err != nil {
			logger.Log("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Refresh tf state after update
	diags_ret = resourceInfo.ResourceSchema.ReadContext(ctx, d, client)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceDelete(ctx context.Context, d *schema.ResourceData, client *client.Client, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Deleting resource")

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	resouce_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKey(key_template, resouce_id, d)
	key := config.ConfigKey{Key: key_string}

	// Check for blocking resources before delete
	for _, blockKeyTemplateStr := range resourceInfo.DeleteBlockerTemplates {
		blockKeyTemplate := config.ConfigKeyTemplate{Template: blockKeyTemplateStr}
		resouce_id := config.FormatResourceId(blockKeyTemplate, d)
		blockKey := config.FormatKey(blockKeyTemplate, resouce_id, d)
		val, err := client.Config.Show(ctx, blockKey)
		if err != nil {
			return diag.FromErr(err)
		}
		if val != nil {
			return diag.Errorf("Configuration '%s' has blocker '%s' delete before continuing.", key, blockKey)
		}
	}

	// Generate config object from VyOS (only used for logs)
	vyos_key := key
	vyos_config, diags_ret := config.NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diags_ret...)

	// Remove resource
	_, delete_config := vyos_config.MarshalVyos()
	logger.Log("INFO", "Deleting key: '%s' using strategy: '%s' vyos resource: %#v", key_string, resourceInfo.DeleteStrategy, delete_config)

	var err error

	if resourceInfo.DeleteStrategy == DeleteTypeResource {
		err = client.Config.Delete(ctx, key_string)
	} else if resourceInfo.DeleteStrategy == DeleteTypeParameters {
		err = client.Config.Delete(ctx, key_string, delete_config)
	} else {
		return diag.Errorf("Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
	}

	if err != nil {
		logger.Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret = resourceInfo.ResourceSchema.ReadContext(ctx, d, client)
	diags = append(diags, diags_ret...)

	return diags
}
