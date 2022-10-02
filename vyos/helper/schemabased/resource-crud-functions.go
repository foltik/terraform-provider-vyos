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

func keyAndTemplate(d *schema.ResourceData, resourceInfo *ResourceInfo) (ConfigKey, ConfigKeyTemplate) {
	/*
		Useful for read, update and delete functions.
		Create function does not have an ID to rely on and can currently not use this to get the key and template
	*/
	key_template := ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	key_string := FormatKeyFromId(key_template, d.Id())
	key := ConfigKey{Key: key_string}

	return key, key_template
}

func ResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	Log("INFO", "Reading resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key, _ := keyAndTemplate(d, resourceInfo)

	// Generate config object from VyOS
	vyos_config, err_ret := NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_ret != nil {
		diags = append(diags, diag.FromErr(err_ret)...)
		return diags
	}

	// If resource does not exist in VyOS
	if vyos_config == nil {
		Log("WARNING", "Resource not found on remote server, setting id to empty string for: %s", key.Key)
		d.SetId("")
		return diags
	} else {

		// Update state
		terraform_values := vyos_config.MarshalTerraform()
		for parameter := range resourceInfo.ResourceSchema.Schema {

			if value, ok := terraform_values[parameter]; ok {
				Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
				d.Set(parameter, value)
			} else {
				Log("DEBUG", "Parameter: %s, not in config, setting to nil", parameter)
				d.Set(parameter, nil)
			}
		}

		// Set fields that make up the key / resource ID
		for key_parameter, key_value := range GetFieldValuePairsFromId(d.Id()) {
			Log("DEBUG", "Setting key parameter: %s, to key_value: %v", key_parameter, key_value)

			switch resourceInfo.ResourceSchema.Schema[key_parameter].Type {
			case schema.TypeBool:
				Log("DEBUG", "Converting to bool")

				if strings.ToLower(key_value) == "true" {
					d.Set(key_parameter, true)
				} else if strings.ToLower(key_value) == "false" {
					d.Set(key_parameter, false)
				} else {
					diags = append(diags, diag.Errorf("Key parameter should be bool, but was not true or false. Instead got: %s", key_value)...)
					return diags
				}

			case schema.TypeFloat:
				Log("DEBUG", "Converting to float")

				f, err := strconv.ParseFloat(key_value, 64)
				if err != nil {
					diags = append(diags, diag.FromErr(err)...)
					return diags
				}
				d.Set(key_parameter, f)
			case schema.TypeInt:
				Log("DEBUG", "Converting to int")

				i, err := strconv.ParseInt(key_value, 10, 32)
				if err != nil {
					diags = append(diags, diag.FromErr(err)...)
					return diags
				}
				d.Set(key_parameter, i)
			case schema.TypeString:
				Log("DEBUG", "Keeping as string")
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

	Log("INFO", "Creating resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_template := ConfigKeyTemplate{Template: resourceInfo.KeyTemplate}
	resource_id := FormatResourceId(key_template, d)
	key_string := FormatKeyFromResource(key_template, d)
	key := ConfigKey{Key: key_string}

	// Check for required resources before create
	for _, reqKeyTemplateStr := range resourceInfo.CreateRequiredTemplates {
		reqKeyTemplate := ConfigKeyTemplate{Template: reqKeyTemplateStr}
		reqKey := ConfigKey{Key: FormatKeyFromResource(reqKeyTemplate, d)}

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {

			// Check if resource exists (changing name of a resource will cause this to error out before the old resource is deleted at times)
			vyos_config_self, err_self := NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
			if err_self != nil {
				return resource.NonRetryableError(err_self)
			}

			if vyos_config_self != nil {
				Log("ERROR", "Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resource_id)
				return resource.RetryableError(fmt.Errorf("Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resource_id))
			}

			// Get required config
			vyos_config, sub_err := NewConfigFromVyos(ctx, &reqKey, resourceInfo.ResourceSchema, client)

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
	terraform_config, err_ret := NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	for _, field := range GetKeyFieldsFromTemplate(key_template) {
		terraform_config.PopChild(field)
		Log("INFO", "Removed key field from config object: %v", field)
	}

	path, value := terraform_config.MarshalVyos()
	err := client.Config.Set(ctx, path, value)

	if err != nil {
		Log("ERROR", "API Client error: %v", err)
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
	Log("INFO", "Updating resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key, key_template := keyAndTemplate(d, resourceInfo)

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, err_ret := NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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
	for _, field := range GetKeyFieldsFromTemplate(key_template) {
		terraform_config.PopChild(field)
		Log("INFO", "Removed key field from config object: %v", field)
	}

	// Find config changes
	changed, deleted := terraform_config.GetDifference(vyos_config)

	// Remove deleted parameters
	if deleted != nil {
		deleted_path, deleted_config := deleted.MarshalVyos()
		Log("INFO", "Deleted detected: %#v", deleted_config)
		err := client.Config.Delete(ctx, deleted_path, deleted_config)
		if err != nil {
			Log("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Apply changed parameters
	if changed != nil {
		changed_path, changed_config := changed.MarshalVyos()
		Log("INFO", "Changes detected: %#v", changed_config)
		err := client.Config.Set(ctx, changed_path, changed_config)
		if err != nil {
			Log("ERROR", "API Client error: %v", err)
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

	Log("INFO", "Deleting resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key, _ := keyAndTemplate(d, resourceInfo)

	// Check for blocking resources before delete
	for _, blockKeyTemplateStr := range resourceInfo.DeleteBlockerTemplates {
		blockKeyTemplate := ConfigKeyTemplate{Template: blockKeyTemplateStr}
		blockKey := FormatKeyFromResource(blockKeyTemplate, d)

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
	vyos_config, err_ret := NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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
	Log("INFO", "Deleting key: '%s' using strategy: '%s' vyos resource: %#v", key.Key, resourceInfo.DeleteStrategy, delete_config)

	var err error

	if resourceInfo.DeleteStrategy == DeleteTypeResource {
		err = client.Config.Delete(ctx, key.Key)
	} else if resourceInfo.DeleteStrategy == DeleteTypeParameters {
		err = client.Config.Delete(ctx, key.Key, delete_config)
	} else {
		Log("ERROR", "Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
		return diag.Errorf("Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
	}

	if err != nil {
		Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func globalResourceRemoveSuperfluous(cfg *ConfigBlock, resourceInfo *ResourceInfo) error {
	/*
		Readies global config by removing extra children.
		Returns error if there are any values at the top level
	*/

	// There should not be any top level values on global resources
	if _, has_values := cfg.GetValues(); has_values {
		return fmt.Errorf("Global resources with top level values are currently not supported")
	}

	// Remove parameters and children not defined in schema
	children, _ := cfg.GetChildren()
	for child_key := range children {
		if _, ok := resourceInfo.ResourceSchema.Schema[child_key.Key]; ok == false {
			cfg.PopChild(child_key.Key)
		}
	}

	return nil
}

func ResourceReadGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Global resources must have a static ID defined in the resourceInfo struct
	*/

	Log("INFO", "Reading global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := ConfigKey{Key: key_string}

	// Generate config object from VyOS
	vyos_config, err_ret := NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_ret != nil {
		diags = append(diags, diag.FromErr(err_ret)...)
		return diags
	}

	if vyos_config == nil {
		Log("DEBUG", "Resource not found on remote server, setting id to empty string for: %s", key.Key)
		d.SetId("")
		return diags
	} else {

		// Trim config
		if err := globalResourceRemoveSuperfluous(vyos_config, resourceInfo); err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}

		// Update state
		terraform_values := vyos_config.MarshalTerraform()
		for parameter := range resourceInfo.ResourceSchema.Schema {

			if value, ok := terraform_values[parameter]; ok {
				Log("DEBUG", "Setting parameter: %s, to value: %v", parameter, value)
				d.Set(parameter, value)
			} else {
				Log("DEBUG", "Parameter: %s, not in config, setting to nil", parameter)
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

	Log("INFO", "Creating global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := ConfigKey{Key: key_string}

	// Check if resource exists
	vyos_config_self, err_self := NewConfigFromVyos(ctx, &key, resourceInfo.ResourceSchema, client)
	if err_self != nil {
		return diag.FromErr(err_self)
	}

	// Trim config
	if err := globalResourceRemoveSuperfluous(vyos_config_self, resourceInfo); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	if _, has_children := vyos_config_self.GetChildren(); has_children {
		Log("ERROR", "Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resourceInfo.StaticId)
		return diag.Errorf("Configuration under key '%s' already exists, consider an import of id: '%s'", key.Key, resourceInfo.StaticId)
	}

	// Check for required resources before create
	for _, reqKeyTemplateStr := range resourceInfo.CreateRequiredTemplates {
		reqKeyTemplate := ConfigKeyTemplate{Template: reqKeyTemplateStr}
		reqKey := ConfigKey{Key: FormatKeyFromResource(reqKeyTemplate, d)}

		// Retry until timeout
		err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-(API_TIMEOUT_BUFFER_IN_SECONDS*time.Second), func() *resource.RetryError {

			// Get required config
			vyos_config, sub_err := NewConfigFromVyos(ctx, &reqKey, resourceInfo.ResourceSchema, client)

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
	terraform_config, err_ret := NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	path, value := terraform_config.MarshalVyos()
	err := client.Config.Set(ctx, path, value)

	if err != nil {
		Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Set ID (used in read function and must be set before that)
	d.SetId(resourceInfo.StaticId)

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}

func ResourceUpdateGlobal(ctx context.Context, d *schema.ResourceData, m interface{}, resourceInfo *ResourceInfo) (diags diag.Diagnostics) {
	/*
		Global resources must have a static ID defined in the resourceInfo struct
	*/
	Log("INFO", "Updating global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := ConfigKey{Key: key_string}

	// Create terraform config struct
	terraform_key := key
	terraform_config, err_ret := NewConfigFromTerraform(ctx, &terraform_key, resourceInfo.ResourceSchema, d)
	diags = append(diags, diag.FromErr(err_ret)...)

	// Generate config object from VyOS
	vyos_key := key
	vyos_config, err_ret := NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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
	if err := globalResourceRemoveSuperfluous(vyos_config, resourceInfo); err != nil {
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
		Log("INFO", "Deleted detected: %#v", deleted_config)
		err := client.Config.Delete(ctx, deleted_path, deleted_config)
		if err != nil {
			Log("ERROR", "API Client error: %v", err)
			return diag.FromErr(err)
		}
	}

	// Apply changed parameters
	if changed != nil {
		changed_path, changed_config := changed.MarshalVyos()
		Log("INFO", "Changes detected: %#v", changed_config)
		err := client.Config.Set(ctx, changed_path, changed_config)
		if err != nil {
			Log("ERROR", "API Client error: %v", err)
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

	Log("INFO", "Deleting global resource")

	// Client
	client := m.(*providerStructure.ProviderClass).Client

	// Key and ID
	key_string := resourceInfo.KeyTemplate
	key := ConfigKey{Key: key_string}

	// Check for blocking resources before delete
	for _, blockKeyTemplateStr := range resourceInfo.DeleteBlockerTemplates {
		blockKeyTemplate := ConfigKeyTemplate{Template: blockKeyTemplateStr}
		blockKey := FormatKeyFromResource(blockKeyTemplate, d)

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
	vyos_config, err_ret := NewConfigFromVyos(ctx, &vyos_key, resourceInfo.ResourceSchema, client)
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
	if err := globalResourceRemoveSuperfluous(vyos_config, resourceInfo); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// Remove resource
	_, delete_config := vyos_config.MarshalVyos()
	Log("INFO", "Deleting key: '%s' using strategy: '%s' vyos resource: %#v", key_string, resourceInfo.DeleteStrategy, delete_config)

	var err error

	if resourceInfo.DeleteStrategy == DeleteTypeResource {
		err = client.Config.Delete(ctx, key_string)
	} else if resourceInfo.DeleteStrategy == DeleteTypeParameters {
		err = client.Config.Delete(ctx, key_string, delete_config)
	} else {
		Log("ERROR", "Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
		return diag.Errorf("Configuration '%s' has unknown delete strategy '%s', this is a provider error.", key, resourceInfo.DeleteStrategy)
	}

	if err != nil {
		Log("ERROR", "API Client error: %v", err)
		return diag.FromErr(err)
	}

	// Refresh tf state after update
	diags_ret := resourceInfo.ResourceSchema.ReadContext(ctx, d, m)
	diags = append(diags, diags_ret...)

	return diags
}
