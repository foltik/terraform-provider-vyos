package schemabased

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	providerStructure "github.com/foltik/terraform-provider-vyos/vyos/provider-structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Sleep time between retries
const API_BACKOFF_TIME_IN_SECONDS = 2

// Retries will have this many seconds less than configured timeout for reources
const API_TIMEOUT_BUFFER_IN_SECONDS = 5

// TODO refactor common logic into smaller functions and reuse them in global / non global functions.
// TODO refactor commom logic between operations into smaller functions for reuse

func ResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger("INFO", "Reading resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key, _ := keyAndTemplate(d, resourceInfo)

	// Generate config object from VyOS
	vyos_config, err_ret := newConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_ret != nil {
		diags = append(diags, diag.FromErr(err_ret)...)
		return diags
	}

	// If resource does not exist in VyOS
	if vyos_config == nil {
		logger("WARNING", "Resource not found on remote server, setting id to empty string for: %s", key.Key)
		d.SetId("")
		return diags
	} else {

		// Update state
		terraform_values := vyos_config.MarshalTerraform()
		for parameter := range resourceInfo.ResourceSchema.Schema {

			if value, ok := terraform_values[parameter]; ok {
				logger("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
				d.Set(parameter, value)
			} else {
				logger("DEBUG", "Parameter: %s, not in config, setting to nil", parameter)
				d.Set(parameter, nil)
			}
		}

		// Set fields that make up the key / resource ID
		for key_parameter, key_value := range getFieldValuePairsFromId(d.Id()) {
			logger("DEBUG", "Setting key parameter: %s, to key_value: %v", key_parameter, key_value)

			switch resourceInfo.ResourceSchema.Schema[key_parameter].Type {
			case schema.TypeBool:
				logger("DEBUG", "Converting to bool")

				if strings.ToLower(key_value) == "true" {
					d.Set(key_parameter, true)
				} else if strings.ToLower(key_value) == "false" {
					d.Set(key_parameter, false)
				} else {
					diags = append(diags, diag.Errorf("Key parameter should be bool, but was not true or false. Instead got: %s", key_value)...)
					return diags
				}

			case schema.TypeFloat:
				logger("DEBUG", "Converting to float")

				f, err := strconv.ParseFloat(key_value, 64)
				if err != nil {
					diags = append(diags, diag.FromErr(err)...)
					return diags
				}
				d.Set(key_parameter, f)
			case schema.TypeInt:
				logger("DEBUG", "Converting to int")

				i, err := strconv.ParseInt(key_value, 10, 32)
				if err != nil {
					diags = append(diags, diag.FromErr(err)...)
					return diags
				}
				d.Set(key_parameter, i)
			case schema.TypeString:
				logger("DEBUG", "Keeping as string")
				d.Set(key_parameter, key_value)
			default:
				diags = append(diags, diag.Errorf("Key parameter can only be of type bool, float, int, or string. Got schema.Type...: %s", resourceInfo.ResourceSchema.Schema[key_parameter].Type)...)
				return diags
			}

		}
	}

	return diags
}

func ResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Supports timeout
	*/

	logger("INFO", "Creating resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_template := configKeyTemplate{Template: resourceInfo.KeyTemplate}
	resource_id := formatResourceId(key_template, d)
	key_string := formatKeyFromResource(key_template, d)
	key := configKey{Key: key_string}

	// Check for required resources before create
	for _, reqKeyTemplateStr := range resourceInfo.CreateRequiredTemplates {
		reqKeyTemplate := configKeyTemplate{Template: reqKeyTemplateStr}
		reqKey := configKey{Key: formatKeyFromResource(reqKeyTemplate, d)}

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {

			// Get required config
			vyos_config, sub_err := newConfigFromVyos(ctx, &reqKey, resourceInfo.ResourceSchema, client)

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

	// Check if resource exists (rare bug: changing name of a resource will cause this to error out before the old resource is deleted at times)
	err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {
		vyos_config_self, err_self := newConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
		if err_self != nil {
			return resource.NonRetryableError(err_self)
		}

		if vyos_config_self != nil {
			logger("ERROR", "Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resource_id)
			return resource.RetryableError(fmt.Errorf("Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resource_id))
		}

		return nil
	})

	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := newConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	for _, field := range getKeyFieldsFromTemplate(key_template) {
		terraform_config.PopChild(field)
		logger("INFO", "Removed key field from config object: %v", field)
	}

	path, value := terraform_config.MarshalVyos()
	err = client.Config.Set(ctx, path, value)

	if err != nil {
		logger("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Set ID (used in read function and must be set before that)
	d.SetId(resource_id)

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	logger("INFO", "Updating resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key, key_template := keyAndTemplate(d, resourceInfo)

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := newConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, err_ret := newConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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
	for _, field := range getKeyFieldsFromTemplate(key_template) {
		terraform_config.PopChild(field)
		logger("INFO", "Removed key field from config object: %v", field)
	}

	// Find config changes
	changed, deleted := terraform_config.GetDifference(vyos_config)

	// Remove deleted parameters
	if deleted != nil {
		deleted_path, deleted_config := deleted.MarshalVyos()
		logger("INFO", "Deleted detected: %#v", deleted_config)
		err := client.Config.Delete(ctx, deleted_path, deleted_config)
		if err != nil {
			logger("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Apply changed parameters
	if changed != nil {
		changed_path, changed_config := changed.MarshalVyos()
		logger("INFO", "Changes detected: %#v", changed_config)
		err := client.Config.Set(ctx, changed_path, changed_config)
		if err != nil {
			logger("ERROR", "API Client error: %v", err)
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

	logger("INFO", "Deleting resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key, _ := keyAndTemplate(d, resourceInfo)

	// Check for blocking resources before delete
	for _, blockKeyTemplateStr := range resourceInfo.DeleteBlockerTemplates {
		blockKeyTemplate := configKeyTemplate{Template: blockKeyTemplateStr}
		blockKey := formatKeyFromResource(blockKeyTemplate, d)

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {
			val, err := client.Config.Show(ctx, blockKey)
			if err != nil {
				return resource.NonRetryableError(err)
			}
			if val != nil {
				return resource.RetryableError(fmt.Errorf("Configuration '%s' has blocker '%s' delete before continuing.", key.Key, blockKey))
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
	vyos_config, err_ret := newConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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
	logger("INFO", "Deleting key: '%s' using strategy: '%s' vyos resource: %#v", key.Key, resourceInfo.DeleteStrategy, delete_config)

	var err error

	if resourceInfo.DeleteStrategy == DeleteTypeResource {
		err = client.Config.Delete(ctx, key.Key)
	} else if resourceInfo.DeleteStrategy == DeleteTypeParameters {
		err = client.Config.Delete(ctx, key.Key, delete_config)
	} else {
		logger("ERROR", "Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
		return diag.Errorf("Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
	}

	if err != nil {
		logger("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceReadGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Global resources must have a static ID defined in the resourceInfo struct
	*/

	logger("INFO", "Reading global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := configKey{Key: key_string}

	// Generate config object from VyOS
	vyos_config, err_ret := newConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_ret != nil {
		diags = append(diags, diag.FromErr(err_ret)...)
		return diags
	}

	if vyos_config == nil {
		logger("DEBUG", "Resource not found on remote server, setting id to empty string for: %s", key.Key)
		d.SetId("")
		return diags
	} else {

		// Trim config
		if err := vyos_config.GlobalResourceRemoveSuperfluous(resourceInfo); err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}

		// Update state
		terraform_values := vyos_config.MarshalTerraform()
		for parameter := range resourceInfo.ResourceSchema.Schema {

			if value, ok := terraform_values[parameter]; ok {
				logger("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
				d.Set(parameter, value)
			} else {
				logger("DEBUG", "Parameter: %s, not in config, setting to nil", parameter)
				d.Set(parameter, nil)
			}
		}
	}

	return diags
}

func ResourceCreateGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Global resources must have a static ID defined in the resourceInfo struct
		Supports timeout
	*/

	logger("INFO", "Creating global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := configKey{Key: key_string}

	// Set ID (used in read function and must be set before that)
	d.SetId(resourceInfo.StaticId)

	// Check if resource exists
	vyos_config_self, err_self := newConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_self != nil {
		return diag.FromErr(err_self)
	}

	if vyos_config_self != nil {
		// Trim config
		if err := vyos_config_self.GlobalResourceRemoveSuperfluous(resourceInfo); err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}

		if _, has_children := vyos_config_self.GetChildren(); has_children {
			logger("ERROR", "Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resourceInfo.StaticId)
			return diag.Errorf("Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resourceInfo.StaticId)
		}
	}

	// Check for required resources before create
	for _, reqKeyTemplateStr := range resourceInfo.CreateRequiredTemplates {
		reqKeyTemplate := configKeyTemplate{Template: reqKeyTemplateStr}
		reqKey := configKey{Key: formatKeyFromResource(reqKeyTemplate, d)}

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {

			// Get required config
			vyos_config, sub_err := newConfigFromVyos(ctx, &reqKey, resourceInfo.ResourceSchema, client)

			if sub_err != nil {
				return resource.NonRetryableError(sub_err)
			} else if vyos_config == nil {
				time.Sleep(API_BACKOFF_TIME_IN_SECONDS * time.Second)
				return resource.RetryableError(fmt.Errorf("Required configuration '%s' missing.", reqKey.Key))
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
	terraform_config, err_ret := newConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	if err_ret != nil {
		diags = append(diags, diag.FromErr(err_ret)...)
		return diags
	}

	path, value := terraform_config.MarshalVyos()
	err := client.Config.Set(ctx, path, value)

	if err != nil {
		logger("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceUpdateGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Global resources must have a static ID defined in the resourceInfo struct
	*/
	logger("INFO", "Updating global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := configKey{Key: key_string}

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := newConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, err_ret := newConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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

	// Trim config
	if err := vyos_config.GlobalResourceRemoveSuperfluous(resourceInfo); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// Remove fields/parameters only internal to terraform so they are not part of the comparison
	terraform_config.PopChild("id")

	// Find config changes
	changed, deleted := terraform_config.GetDifference(vyos_config)

	// Remove deleted parameters
	if deleted != nil {
		deleted_path, deleted_config := deleted.MarshalVyos()
		logger("INFO", "Deleted detected: %#v", deleted_config)
		err := client.Config.Delete(ctx, deleted_path, deleted_config)
		if err != nil {
			logger("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Apply changed parameters
	if changed != nil {
		changed_path, changed_config := changed.MarshalVyos()
		logger("INFO", "Changes detected: %#v", changed_config)
		err := client.Config.Set(ctx, changed_path, changed_config)
		if err != nil {
			logger("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceDeleteGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Global resources must have a static ID defined in the resourceInfo struct
		Supports timeout
	*/

	logger("INFO", "Deleting global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := configKey{Key: key_string}

	// Check for blocking resources before delete
	for _, blockKeyTemplateStr := range resourceInfo.DeleteBlockerTemplates {
		blockKeyTemplate := configKeyTemplate{Template: blockKeyTemplateStr}
		blockKey := formatKeyFromResource(blockKeyTemplate, d)

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {
			val, err := client.Config.Show(ctx, blockKey)
			if err != nil {
				return resource.NonRetryableError(err)
			}
			if val != nil {
				return resource.RetryableError(fmt.Errorf("Configuration '%s' has blocker '%s' delete before continuing.", key.Key, blockKey))
			}
			return nil
		})

		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, err_ret := newConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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

	// Trim config
	if err := vyos_config.GlobalResourceRemoveSuperfluous(resourceInfo); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// Remove resource
	_, delete_config := vyos_config.MarshalVyos()
	logger("INFO", "Deleting key: '%s' using strategy: '%s' vyos resource: %#v", key_string, resourceInfo.DeleteStrategy, delete_config)

	var err error

	if resourceInfo.DeleteStrategy == DeleteTypeResource {
		err = client.Config.Delete(ctx, key_string)
	} else if resourceInfo.DeleteStrategy == DeleteTypeParameters {
		err = client.Config.Delete(ctx, key_string, delete_config)
	} else {
		logger("ERROR", "Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
		return diag.Errorf("Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
	}

	if err != nil {
		logger("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}
