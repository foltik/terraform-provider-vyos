package resourceInfo

import (
	"context"
	"fmt"
	"time"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/config"
	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
	providerStructure "github.com/foltik/terraform-provider-vyos/vyos/provider-structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Sleep time between retries
const API_BACKOFF_TIME_IN_SECONDS = 2

// Retries will have this many seconds less than configured timeout for reources
const API_TIMEOUT_BUFFER_IN_SECONDS = 5

func ResourceReadGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Reading resource, global type")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := config.ConfigKey{Key: key_string}
	//resource_id := resourceInfo.StaticId

	// Generate config object from VyOS
	vyos_config, err_ret := config.NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diag.FromErr(err_ret)...)

	if vyos_config == nil {
		logger.Log("DEBUG", "Resource not found on remote server, setting id to empty string for: %s", key.Key)
		d.SetId("")
		return diags
	} else {
		for parameter, value := range vyos_config.MarshalTerraform() {
			logger.Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
			d.Set(parameter, value)
		}
	}

	//d.SetId(resouce_id)
	return diags
}

func ResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Reading resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	key_string := config.FormatKeyFromId(key_template, d.Id())
	key := config.ConfigKey{Key: key_string}

	// Generate config object from VyOS
	vyos_config, err_ret := config.NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diag.FromErr(err_ret)...)

	// If resource does not exist in VyOS
	if vyos_config == nil {
		logger.Log("WARNING", "Resource not found on remote server, setting id to empty string for: %s", key.Key)
		d.SetId("")
		return diags
	} else {
		for parameter, value := range vyos_config.MarshalTerraform() {
			logger.Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
			d.Set(parameter, value)
		}

		for key_parameter, key_value := range config.GetFieldValuePairsFromId(d.Id()) {
			logger.Log("DEBUG", "Setting key parameter: %s, to key_value: %v", key_parameter, key_value)
			d.Set(key_parameter, key_value)
		}
	}

	return diags
}

func ResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Supports timeout
	*/

	logger.Log("INFO", "Creating resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	resource_id := config.FormatResourceId(key_template, d)
	key_string := config.FormatKeyFromResource(key_template, d)
	key := config.ConfigKey{Key: key_string}

	// Check if resource exists
	vyos_config_self, err_self := config.NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_self != nil {
		return diag.FromErr(err_self)
	}

	if vyos_config_self != nil {
		return diag.Errorf("Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resource_id)
	}

	// Check for required resources before create
	for _, reqKeyTemplateStr := range resourceInfo.CreateRequiredTemplates {
		reqKeyTemplate := config.ConfigKeyTemplate{Template: reqKeyTemplateStr}
		reqKey := config.ConfigKey{Key: config.FormatKeyFromResource(reqKeyTemplate, d)}

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {

			// Get required config
			vyos_config, sub_err := config.NewConfigFromVyos(ctx, &reqKey, resourceInfo.ResourceSchema, client)

			if sub_err != nil {
				return resource.NonRetryableError(sub_err)
			} else if vyos_config == nil {
				time.Sleep(API_BACKOFF_TIME_IN_SECONDS * time.Second)
				return resource.RetryableError(fmt.Errorf("Required parent configuration '%s' missing.", reqKey.Key))
			} else {
				return nil
			}
		})

		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := config.NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	for _, field := range config.GetKeyFieldsFromTemplate(key_template) {
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
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	d.SetId(resource_id)

	return diags
}

func ResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger.Log("INFO", "Updating resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	key_string := config.FormatKeyFromId(key_template, d.Id())
	key := config.ConfigKey{Key: key_string}

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := config.NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, err_ret := config.NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diag.FromErr(err_ret)...)

	if vyos_config == nil {
		diags = append(
			diags,
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Resource not found on remote server: " + vyos_key.Key,
			},
		)
		return diags
	}

	// Remove fields/parameters only internal to terraform so they are not part of the comparison
	terraform_config.PopChild("id")
	for _, field := range config.GetKeyFieldsFromTemplate(key_template) {
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
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Supports timeout
	*/

	logger.Log("INFO", "Deleting resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_template := config.ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	key_string := config.FormatKeyFromId(key_template, d.Id())
	key := config.ConfigKey{Key: key_string}

	// Check for blocking resources before delete
	for _, blockKeyTemplateStr := range resourceInfo.DeleteBlockerTemplates {
		blockKeyTemplate := config.ConfigKeyTemplate{Template: blockKeyTemplateStr}
		blockKey := config.FormatKeyFromResource(blockKeyTemplate, d)

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {
			val, err := client.Config.Show(ctx, blockKey)
			if err != nil {
				return resource.NonRetryableError(err)
			}
			if val != nil {
				return resource.RetryableError(fmt.Errorf("Configuration '%s' has blocker '%s' delete before continuing.", key, blockKey))
			}
			return nil
		})

		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}

	// Generate config object from VyOS (only used for logs)
	vyos_key := key
	vyos_config, err_ret := config.NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
	diags = append(diags, diag.FromErr(err_ret)...)

	if vyos_config == nil {
		diags = append(
			diags,
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Resource not found on remote server: " + vyos_key.Key,
			},
		)
		return diags
	}

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
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}
