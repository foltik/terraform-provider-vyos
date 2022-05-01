package vyos

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func configFromVyos(ctx context.Context, parent_config *ConfigBlock, resource_schema interface{}, parent_vyos_native_config interface{}) (diags diag.Diagnostics) {
	// Recursive function to walk VyOS config and return a ConfigBlock
	// Attempt to make any type of failure loud and immidiate to help discover edge cases.

	logF("TRACE", "parent_config.key:%#v", parent_config.key)

	// Loop over maps/lists/sets, handle values after the switch
	switch resource_schema := resource_schema.(type) {

	case map[string]*schema.Schema:
		// Config block
		logF("TRACE", "resource_schema is map of schema")

		for key_string, schema := range resource_schema {
			// Convert to map as expected based on schema type
			parent_vyos_config := parent_vyos_native_config.(map[string]interface{})

			// If VyOS has this parameter set create config object and populate it
			if vyos_config, ok := parent_vyos_config[key_string]; ok {
				key := ConfigKey{key_string}
				child_config := *parent_config.AddChild(&key)
				child_diags := configFromVyos(ctx, &child_config, schema, vyos_config)
				diags = append(diags, child_diags...)
			} else {
				logF("TRACE", "parent_vyos_config does not contain key: %s", key_string)
			}
		}

	case *schema.Schema:
		// Config parameters
		logF("TRACE", "resource_schema.Type: %s", resource_schema.Type)

		switch resource_schema.Type {
		case schema.TypeString, schema.TypeInt, schema.TypeFloat:
			// Handle simple native types here

			if vyos_native_config, ok := parent_vyos_native_config.(string); ok {
				parent_config.AddValue(vyos_native_config)
			} else {
				// Make unhandled cases visible
				logF("ERROR", "resource_schema is unhandled: %#v", resource_schema)
				diags = append(diags, diag.Errorf("resource_schema is unhandled: %#v", resource_schema)...)
			}
		case schema.TypeBool:
			// Handle bool here as it shows up differently in vyos
			logF("TRACE", "Should be bool: parent_vyos_native_config: %#v", parent_vyos_native_config)
			if _, ok := parent_vyos_native_config.(map[string]interface{}); ok {
				parent_config.AddValue("true")
			} else {
				// Make unhandled cases visible
				logF("ERROR", "resource_schema is unhandled: %#v", resource_schema)
				diags = append(diags, diag.Errorf("resource_schema is unhandled: %#v", resource_schema)...)
			}

		case schema.TypeList, schema.TypeMap, schema.TypeSet:

			// List/Set can be a collection of values or collection of nested config blocks
			logF("TRACE", "resource_schema set/list")

			if resource_schema_elem, ok := resource_schema.Elem.(*schema.Resource); ok {
				// If this is a config block recurse the block and return result
				resource_schema_elem_schema := resource_schema_elem.Schema

				logF("TRACE", "resource_schema_elem_schema: %#v", resource_schema_elem_schema)
				logF("TRACE", "parent_vyos_native_config: %#v", parent_vyos_native_config)

				// Currently have not come across a list/set of sub configs in VyOS, if they appear we might need to just create children with index numbers as the map key
				if resource_schema.MaxItems != 1 {
					logF("ERROR", "resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)
					diags = append(diags, diag.Errorf("resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)...)
					return diags
				} else {
					// Treat list/set as config block
					for key_string, schema := range resource_schema_elem_schema {
						// Convert to map as expected based on schema type
						parent_vyos_config := parent_vyos_native_config.(map[string]interface{})

						// If VyOS has this parameter set create config object and populate it
						if vyos_config, ok := parent_vyos_config[key_string]; ok {
							child_diags := configFromVyos(ctx, parent_config, schema, vyos_config)
							diags = append(diags, child_diags...)
						} else {
							logF("TRACE", "parent_vyos_config does not contain key: %s", key_string)
						}
					}
				}
			} else {
				// Make unhandled cases visible
				logF("ERROR", "resource_schema.Elem is unhandled: %#v", resource_schema)
				diags = append(diags, diag.Errorf("resource_schema.Elem is unhandled: %#v", resource_schema)...)
			}
		default:
			// Make unhandled cases visible
			logF("ERROR", "resource_schema is unhandled: %#v", resource_schema)
			diags = append(diags, diag.Errorf("resource_schema is unhandled: %#v", resource_schema)...)
		}

	default:
		// Make unhandled cases visible
		logF("ERROR", "resource_schema is unhandled: %#v", resource_schema)
		diags = append(diags, diag.Errorf("resource_schema is unhandled: %#v", resource_schema)...)
	}

	return diags
}

func configFromTerraform(ctx context.Context, parent_config *ConfigBlock, resource_schema interface{}, terraform_data *schema.ResourceData) (diags diag.Diagnostics) {
	return diags
}
