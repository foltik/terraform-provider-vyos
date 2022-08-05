package config

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
)

func NewConfigFromVyos(ctx context.Context, vyos_key *ConfigKey, resource_schema *schema.Resource, vyos_client *client.Client) (*ConfigBlock, diag.Diagnostics) {
	logger.Log("DEBUG", "vyos_key: %#v", vyos_key)
	var diags diag.Diagnostics

	config_block := ConfigBlock{
		key: vyos_key,
	}

	logger.Log("DEBUG", "Asking client to fetch vyos config: %#v", config_block.key)
	vyos_native_config, err := vyos_client.Config.Show(ctx, vyos_key.Key)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// Create VyOS config struct
	diags_ret := vyosWalker(ctx, &config_block, resource_schema.Schema, vyos_native_config)
	diags = append(diags, diags_ret...)

	return &config_block, diags
}

func vyosWalker(ctx context.Context, config_block *ConfigBlock, resource_schema interface{}, parent_vyos_native_config interface{}) (diags diag.Diagnostics) {
	// Recursive function to walk VyOS config and return a ConfigBlock
	// Attempt to make any type of failure loud and immidiate to help discover edge cases.

	logger.Log("TRACE", "{%s} Walking", config_block.key.Key)

	if parent_vyos_native_config == nil {
		logger.Log("TRACE", "{%s} parent_vyos_native_config is nil", config_block.key.Key)
		return diags
	}

	// Loop over maps/lists/sets, handle values after the switch
	switch resource_schema := resource_schema.(type) {

	case map[string]*schema.Schema:
		// Config block
		logger.Log("TRACE", "resource_schema is map of schema")

		for key_string, parameter_schema := range resource_schema {
			// Convert to map as expected based on schema type
			logger.Log("TRACE", "key_string: '%s' parent_vyos_native_config: '%#v'", key_string, parent_vyos_native_config)
			parent_vyos_config := parent_vyos_native_config.(map[string]interface{})
			vyos_key_string := strings.Replace(key_string, "_", "-", -1)

			// If VyOS has this parameter set create config object and populate it
			if vyos_config, ok := parent_vyos_config[vyos_key_string]; ok {
				child_config := config_block.CreateChild(key_string, parameter_schema.Type)
				child_diags := vyosWalker(ctx, child_config, parameter_schema, vyos_config)
				diags = append(diags, child_diags...)
			} else {
				logger.Log("DEBUG", "parent_vyos_config does not contain key: %s", vyos_key_string)

			}
		}

	case *schema.Schema:
		// Config parameters
		logger.Log("TRACE", "resource_schema.Type: %s", resource_schema.Type)
		logger.Log("TRACE", "parent_vyos_native_config: %#v", parent_vyos_native_config)

		switch resource_schema.Type {
		case schema.TypeString, schema.TypeInt, schema.TypeFloat:
			// Handle simple native types here

			if vyos_native_config, ok := parent_vyos_native_config.(string); ok {
				config_block.AddValue(resource_schema.Type, vyos_native_config)
			} else {
				// Make unhandled cases visible
				logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

				diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)...)
			}
		case schema.TypeBool:
			// Handle bool here as it shows up differently in vyos
			logger.Log("TRACE", "Should be bool: parent_vyos_native_config: %#v", parent_vyos_native_config)

			if _, ok := parent_vyos_native_config.(map[string]interface{}); ok {
				config_block.AddValue(resource_schema.Type, "true")
			} else {
				// Make unhandled cases visible
				logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

				diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)...)
			}

		case schema.TypeList, schema.TypeMap, schema.TypeSet:

			// List/Set can be a collection of values or collection of nested config blocks
			logger.Log("TRACE", "resource_schema set/list")

			if resource_schema_elem, ok := resource_schema.Elem.(*schema.Resource); ok {
				// If this is a config block recurse the block and return result
				resource_schema_elem_schema := resource_schema_elem.Schema

				logger.Log("TRACE", "resource_schema_elem_schema: %#v", resource_schema_elem_schema)

				// Currently have not come across a list/set of sub configs in VyOS, if they appear we might need to just create children with index numbers as the map key
				if resource_schema.MaxItems != 1 {
					logger.Log("ERROR", "resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)

					diags = append(diags, diag.Errorf("resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)...)
				} else {
					// Treat list/set as config block
					for key_string, schema := range resource_schema_elem_schema {
						// Convert to map as expected based on schema type
						parent_vyos_config := parent_vyos_native_config.(map[string]interface{})
						vyos_key_string := strings.Replace(key_string, "_", "-", -1)

						// If VyOS has this parameter set create config object and populate it
						if vyos_config, ok := parent_vyos_config[vyos_key_string]; ok {
							sub_config := config_block.CreateChild(key_string, schema.Type)
							sub_diags := vyosWalker(ctx, sub_config, schema, vyos_config)
							diags = append(diags, sub_diags...)
						} else {
							logger.Log("DEBUG", "parent_vyos_config does not contain key: %s", vyos_key_string)

						}
					}
				}
			} else {
				// Make unhandled cases visible
				logger.Log("ERROR", "resource_schema.Elem is unhandled: %#v", resource_schema)

				diags = append(diags, diag.Errorf("resource_schema.Elem is unhandled: %#v", resource_schema)...)
			}
		default:
			// Make unhandled cases visible
			logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

			diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)...)
		}

	default:
		// Make unhandled cases visible
		logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
		diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)...)
	}

	return diags
}

func NewConfigFromTerraform(ctx context.Context, vyos_key *ConfigKey, resource_schema *schema.Resource, data *schema.ResourceData) (*ConfigBlock, diag.Diagnostics) {
	var diags diag.Diagnostics

	id := data.Get("id")
	logger.Log("DEBUG", "resource ID: %#v", id)

	config_block := ConfigBlock{
		key: vyos_key,
	}

	for parameter_key, parameter_schema := range resource_schema.Schema {
		if terraform_native_config, ok := data.GetOk(parameter_key); ok {
			parameter_config_block := config_block.CreateChild(parameter_key, parameter_schema.Type)

			diags_ret := terraformWalker(ctx, parameter_config_block, parameter_schema, terraform_native_config)
			diags = append(diags, diags_ret...)
		}
	}

	return &config_block, diags
}

func terraformWalker(ctx context.Context, config_block *ConfigBlock, resource_schema interface{}, terraform_native_config interface{}) (diags diag.Diagnostics) {
	// Recursive function to walk terraform config and return a ConfigBlock
	// Attempt to make any type of failure loud and immidiate to help discover edge cases.

	logger.Log("TRACE", "parent_config.key: %#v", config_block.key)
	logger.Log("TRACE", "resource_schema: %#v", resource_schema)

	// Loop over maps/lists/sets, handle values after the switch
	switch resource_schema := resource_schema.(type) {

	case map[string]*schema.Schema:
		// Config block
		logger.Log("TRACE", "resource_schema is map of schema")

		for key_string, parameter_schema := range resource_schema {
			// Convert to map as expected based on schema type
			terraform_config := terraform_native_config.(map[string]interface{})

			// If terraform has this parameter set create config object and populate it
			if terraform_sub_config, ok := terraform_config[key_string]; ok {
				logger.Log("TRACE", "found key: '%s' in terraform_config: %#v", key_string, terraform_config)

				if terraform_sub_config != "" && terraform_sub_config != nil {
					sub_config := config_block.CreateChild(key_string, parameter_schema.Type)
					sub_diags := terraformWalker(ctx, sub_config, parameter_schema, terraform_sub_config)
					diags = append(diags, sub_diags...)
				} else {
					logger.Log("TRACE", "key: '%s' seems to be empty string or nil: %#v", key_string, terraform_sub_config)
				}
			} else {
				logger.Log("TRACE", "terraform_config does not contain key: %s", key_string)

			}
		}

	case *schema.Schema:
		// Config parameters
		logger.Log("TRACE", "resource_schema.Type: %s", resource_schema.Type)
		logger.Log("TRACE", "parent_terraform_native_config: %#v", terraform_native_config)

		switch resource_schema.Type {
		case schema.TypeBool:
			config_block.AddValue(resource_schema.Type, strconv.FormatBool(terraform_native_config.(bool)))

		case schema.TypeInt:
			var i int64

			if j, ok := terraform_native_config.(int); ok {
				i = int64(j)
			} else if j, ok := terraform_native_config.(int64); ok {
				i = j
			} else {
				logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v, not int or int64", config_block.key.Key, resource_schema)
				diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v, not int or int64", config_block.key.Key, resource_schema)...)
			}

			config_block.AddValue(resource_schema.Type, strconv.FormatInt(i, 10))

		case schema.TypeFloat:
			if f, ok := terraform_native_config.(float64); ok {
				config_block.AddValue(resource_schema.Type, strconv.FormatFloat(f, 'f', -1, 64))
			} else {
				logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v, not float", config_block.key.Key, resource_schema)
				diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v, not float64", config_block.key.Key, resource_schema)...)
			}

		case schema.TypeString:

			if terraform_native_config, ok := terraform_native_config.(string); ok {
				if terraform_native_config != "" {
					config_block.AddValue(resource_schema.Type, terraform_native_config)
				}
			} else {
				// Make unhandled cases visible
				logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v, not string", config_block.key.Key, resource_schema)
				diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v, not string", config_block.key.Key, resource_schema)...)
			}

		case schema.TypeMap:
			logger.Log("ERROR", "(key: %s) TODO resource_schema.Type: %s", config_block.key.Key, resource_schema, resource_schema.Type)
			diags = append(diags, diag.Errorf("(key: %s) TODO resource_schema.Type: %s", config_block.key.Key, resource_schema, resource_schema.Type)...)

		case schema.TypeList, schema.TypeSet:

			// List/Set can be a collection of values or collection of nested config blocks

			if resource_schema.Type == schema.TypeSet {
				logger.Log("TRACE", "converting from set to list")
				terraform_native_config = terraform_native_config.(*schema.Set).List()
			}

			if resource_schema_elem, ok := resource_schema.Elem.(*schema.Resource); ok {
				// If this is a config block recurse the block and return result
				resource_schema_elem_schema := resource_schema_elem.Schema

				logger.Log("TRACE", "resource_schema_elem_schema: %#v", resource_schema_elem_schema)

				// Currently have not come across a list/set of sub configs in terraform, if they appear we might need to just create children with index numbers as the map key
				if resource_schema.MaxItems != 1 {
					logger.Log("ERROR", "resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)

					diags = append(diags, diag.Errorf("resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)...)
				} else {
					// Treat list/set as config block
					for key_string, parameter_schema := range resource_schema_elem_schema {

						// Get the singular config block element
						var terraform_native_config_block any

						if len(terraform_native_config.([]interface{})) > 0 {
							terraform_native_config_block = terraform_native_config.([]interface{})[0]
						}

						// Some blocks need to support being empty (eg acceleration qat, this might allow for continual diff if tf is poorly written)
						if terraform_native_config_block == nil {
							terraform_native_config_block = make(map[string]interface{})
						}

						// Convert to map as expected based on schema type
						terraform_config := terraform_native_config_block.(map[string]interface{})

						// If terraform has this parameter set create config object and populate it
						if terraform_sub_config, ok := terraform_config[key_string]; ok {
							logger.Log("TRACE", "found key: '%s' in terraform_config: %#v", key_string, terraform_config)

							if terraform_sub_config != "" && terraform_sub_config != nil {
								sub_config := config_block.CreateChild(key_string, parameter_schema.Type)
								sub_diags := terraformWalker(ctx, sub_config, parameter_schema, terraform_sub_config)
								diags = append(diags, sub_diags...)
							} else {
								logger.Log("TRACE", "key: '%s' seems to be empty string or nil: %#v", key_string, terraform_sub_config)
							}
						} else {
							logger.Log("TRACE", "could not find key: '%s' in terraform_config: %#v", key_string, terraform_config)
						}
					}
				}
			} else {
				// Make unhandled cases visible
				logger.Log("ERROR", "resource_schema.Elem is unhandled: %#v", resource_schema)

				diags = append(diags, diag.Errorf("resource_schema.Elem is unhandled: %#v", resource_schema)...)
			}
		default:
			// Make unhandled cases visible
			logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

			diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)...)
		}

	default:
		// Make unhandled cases visible
		logger.Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
		diags = append(diags, diag.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)...)
	}

	tf_json_data, tf_err := json.Marshal(&config_block)
	logger.Log("DEBUG", "err: %s, tf json data: %s\n", tf_err, tf_json_data)

	return diags
}
