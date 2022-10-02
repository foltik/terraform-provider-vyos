package schemabased

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func NewConfigFromVyos(ctx context.Context, vyos_key *ConfigKey, resource_schema *schema.Resource, vyos_client *client.Client) (*ConfigBlock, error) {
	/*
		Return ConfigBlock if config is found.
		Return error if issue is detected.
		Return nil for both if no config is found in VyOS
	*/

	Log("DEBUG", "vyos_key: %#v", vyos_key)

	config_block := ConfigBlock{
		key: vyos_key,
	}

	Log("DEBUG", "Asking client to fetch vyos config: %#v", config_block.key)
	vyos_native_config, err := vyos_client.Config.Show(ctx, vyos_key.Key)
	if vyos_native_config == nil && err == nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Create VyOS config struct
	err = vyosWalker(ctx, &config_block, resource_schema.Schema, vyos_native_config)

	return &config_block, err
}

func vyosWalker(ctx context.Context, config_block *ConfigBlock, resource_schema interface{}, parent_vyos_native_config interface{}) error {
	// Recursive function to walk VyOS config and return a ConfigBlock
	// Attempt to make any type of failure loud and immidiate to help discover edge cases.

	Log("TRACE", "{%s} Walking", config_block.key.Key)

	if parent_vyos_native_config == nil {
		Log("TRACE", "{%s} parent_vyos_native_config is nil", config_block.key.Key)
		return nil
	}

	// Loop over maps/lists/sets, handle values after the switch
	switch resource_schema := resource_schema.(type) {

	case map[string]*schema.Schema:
		// Config block
		Log("TRACE", "resource_schema is map of schema")

		for key_string, parameter_schema := range resource_schema {
			// Convert to map as expected based on schema type
			Log("TRACE", "key_string: '%s' parent_vyos_native_config: '%#v'", key_string, parent_vyos_native_config)
			parent_vyos_config := parent_vyos_native_config.(map[string]interface{})
			vyos_key_string := strings.Replace(key_string, "_", "-", -1)

			// If VyOS has this parameter set create config object and populate it
			if vyos_config, ok := parent_vyos_config[vyos_key_string]; ok {
				child_config := config_block.CreateChild(key_string, parameter_schema.Type)
				child_err := vyosWalker(ctx, child_config, parameter_schema, vyos_config)
				if child_err != nil {
					return child_err
				}
			} else {
				Log("DEBUG", "parent_vyos_config does not contain key: %s", vyos_key_string)

			}
		}

	case *schema.Schema:
		// Config parameters
		Log("TRACE", "resource_schema.Type: %s", resource_schema.Type)
		Log("TRACE", "parent_vyos_native_config: %#v", parent_vyos_native_config)

		switch resource_schema.Type {
		case schema.TypeString, schema.TypeInt, schema.TypeFloat:
			// Handle simple native types here

			if vyos_native_config, ok := parent_vyos_native_config.(string); ok {
				config_block.AddValue(resource_schema.Type, vyos_native_config)
			} else {
				// Make unhandled cases visible
				Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

				return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
			}
		case schema.TypeBool:
			// Handle bool here as it shows up differently in vyos
			Log("TRACE", "Should be bool: parent_vyos_native_config: %#v", parent_vyos_native_config)

			if _, ok := parent_vyos_native_config.(map[string]interface{}); ok {
				config_block.AddValue(resource_schema.Type, "true")
			} else {
				// Make unhandled cases visible
				Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

				return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
			}

		case schema.TypeList, schema.TypeMap, schema.TypeSet:

			// List/Set can be a collection of values or collection of nested config blocks
			Log("TRACE", "resource_schema set/list")

			if resource_schema_elem, ok := resource_schema.Elem.(*schema.Resource); ok {
				// If this is a config block recurse the block and return result
				resource_schema_elem_schema := resource_schema_elem.Schema

				Log("TRACE", "resource_schema_elem_schema: %#v", resource_schema_elem_schema)

				// Currently have not come across a list/set of sub configs in VyOS, if they appear we might need to just create children with index numbers as the map key
				if resource_schema.MaxItems != 1 {
					Log("ERROR", "resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)

					return fmt.Errorf("resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)
				} else {
					// Treat list/set as config block
					for key_string, schema := range resource_schema_elem_schema {
						// Convert to map as expected based on schema type
						parent_vyos_config := parent_vyos_native_config.(map[string]interface{})
						vyos_key_string := strings.Replace(key_string, "_", "-", -1)

						// If VyOS has this parameter set create config object and populate it
						if vyos_config, ok := parent_vyos_config[vyos_key_string]; ok {
							sub_config := config_block.CreateChild(key_string, schema.Type)
							sub_err := vyosWalker(ctx, sub_config, schema, vyos_config)
							if sub_err != nil {
								return sub_err
							}
						} else {
							Log("DEBUG", "parent_vyos_config does not contain key: %s", vyos_key_string)

						}
					}
				}
			} else if resource_schema_elem, ok := resource_schema.Elem.(*schema.Schema); ok {
				Log("TRACE", "resource_schema_elem: '%#v'", resource_schema_elem)

				switch resource_schema_elem.Type {
				case schema.TypeBool, schema.TypeFloat, schema.TypeInt, schema.TypeString:
					Log("TRACE", "adding value: parent_vyos_native_config: '%#v'", parent_vyos_native_config)
					for _, v := range parent_vyos_native_config.([]interface{}) {
						config_block.AddValue(resource_schema_elem.Type, v.(string))
					}
				default:
					Log("ERROR", "resource_schema_elem.Type is unhandled: %#v", resource_schema_elem.Type)
					return fmt.Errorf("resource_schema_elem.Type is unhandled: %#v", resource_schema_elem.Type)
				}
			} else {
				// Make unhandled cases visible
				Log("ERROR", "resource_schema.Elem is unhandled: %#v", resource_schema)

				return fmt.Errorf("resource_schema.Elem is unhandled: %#v", resource_schema)
			}
		default:
			// Make unhandled cases visible
			Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

			return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
		}

	default:
		// Make unhandled cases visible
		Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
		return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
	}

	return nil
}

func NewConfigFromTerraform(ctx context.Context, vyos_key *ConfigKey, resource_schema *schema.Resource, d *schema.ResourceData) (*ConfigBlock, error) {
	id := d.Get("id")
	Log("DEBUG", "resource ID: %#v", id)

	config_block := ConfigBlock{
		key: vyos_key,
	}

	for parameter_key, parameter_schema := range resource_schema.Schema {
		if terraform_native_config, ok := d.GetOk(parameter_key); ok {
			parameter_config_block := config_block.CreateChild(parameter_key, parameter_schema.Type)

			err := terraformWalker(ctx, parameter_config_block, parameter_schema, terraform_native_config)
			if err != nil {
				return nil, err
			}
		}
	}

	return &config_block, nil
}

func terraformWalker(ctx context.Context, config_block *ConfigBlock, resource_schema interface{}, terraform_native_config interface{}) error {
	// Recursive function to walk terraform config and return a ConfigBlock
	// Attempt to make any type of failure loud and immidiate to help discover edge cases.

	Log("TRACE", "parent_config.key: %#v", config_block.key)
	Log("TRACE", "resource_schema: %#v", resource_schema)

	// Loop over maps/lists/sets, handle values after the switch
	switch resource_schema := resource_schema.(type) {

	case map[string]*schema.Schema:
		// Config block
		Log("TRACE", "resource_schema is map of schema")

		for key_string, parameter_schema := range resource_schema {
			// Convert to map as expected based on schema type
			terraform_config := terraform_native_config.(map[string]interface{})

			// If terraform has this parameter set create config object and populate it
			if terraform_sub_config, ok := terraform_config[key_string]; ok {
				Log("TRACE", "found key: '%s' in terraform_config: %#v", key_string, terraform_config)

				if terraform_sub_config != "" && terraform_sub_config != nil {
					sub_config := config_block.CreateChild(key_string, parameter_schema.Type)
					sub_err := terraformWalker(ctx, sub_config, parameter_schema, terraform_sub_config)
					if sub_err != nil {
						return sub_err
					}
				} else {
					Log("TRACE", "key: '%s' seems to be empty string or nil: %#v", key_string, terraform_sub_config)
				}
			} else {
				Log("TRACE", "terraform_config does not contain key: %s", key_string)

			}
		}

	case *schema.Schema:
		// Config parameters
		Log("TRACE", "resource_schema.Type: %s", resource_schema.Type)
		Log("TRACE", "parent_terraform_native_config: %#v", terraform_native_config)

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
				Log("ERROR", "(key: %s)resource_schema is unhandled: %#v, not int or int64", config_block.key.Key, resource_schema)
				return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v, not int or int64", config_block.key.Key, resource_schema)
			}

			config_block.AddValue(resource_schema.Type, strconv.FormatInt(i, 10))

		case schema.TypeFloat:
			if f, ok := terraform_native_config.(float64); ok {
				config_block.AddValue(resource_schema.Type, strconv.FormatFloat(f, 'f', -1, 64))
			} else {
				Log("ERROR", "(key: %s)resource_schema is unhandled: %#v, not float", config_block.key.Key, resource_schema)
				return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v, not float64", config_block.key.Key, resource_schema)
			}

		case schema.TypeString:

			if terraform_native_config, ok := terraform_native_config.(string); ok {
				if terraform_native_config != "" {
					config_block.AddValue(resource_schema.Type, terraform_native_config)
				}
			} else {
				// Make unhandled cases visible
				Log("ERROR", "(key: %s)resource_schema is unhandled: %#v, not string", config_block.key.Key, resource_schema)
				return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v, not string", config_block.key.Key, resource_schema)
			}

		case schema.TypeMap:
			Log("ERROR", "(key: %s) TODO resource_schema.Type: %s", config_block.key.Key, resource_schema, resource_schema.Type)
			return fmt.Errorf("(key: %s) TODO resource_schema.Type: %s", config_block.key.Key, resource_schema.Type)

		case schema.TypeList, schema.TypeSet:

			// List/Set can be a collection of values or collection of nested config blocks

			if resource_schema.Type == schema.TypeSet {
				Log("TRACE", "converting from set to list")
				terraform_native_config = terraform_native_config.(*schema.Set).List()
			}

			if resource_schema_elem, ok := resource_schema.Elem.(*schema.Resource); ok {
				// If this is a config block recurse the block and return result
				resource_schema_elem_schema := resource_schema_elem.Schema

				Log("TRACE", "resource_schema_elem_schema: %#v", resource_schema_elem_schema)

				// Currently have not come across a list/set of sub configs in terraform, if they appear we might need to just create children with index numbers as the map key
				if resource_schema.MaxItems != 1 {
					Log("ERROR", "resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)

					return fmt.Errorf("resource_schema has elem indicating it is a config block, but does not have MaxItems set to one, this configuration is currently unhandled: %#v", resource_schema)
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
							Log("TRACE", "found key: '%s' in terraform_config: %#v", key_string, terraform_config)

							if terraform_sub_config != "" && terraform_sub_config != nil {
								sub_config := config_block.CreateChild(key_string, parameter_schema.Type)
								sub_err := terraformWalker(ctx, sub_config, parameter_schema, terraform_sub_config)
								if sub_err != nil {
									return sub_err
								}
							} else {
								Log("TRACE", "key: '%s' seems to be empty string or nil: %#v", key_string, terraform_sub_config)
							}
						} else {
							Log("TRACE", "could not find key: '%s' in terraform_config: %#v", key_string, terraform_config)
						}
					}
				}
			} else if resource_schema_elem, ok := resource_schema.Elem.(*schema.Schema); ok {
				Log("TRACE", "resource_schema_elem: '%#v'", resource_schema_elem)

				switch resource_schema_elem.Type {
				case schema.TypeBool, schema.TypeFloat, schema.TypeInt, schema.TypeString:
					Log("TRACE", "adding value: terraform_native_config: '%#v'", terraform_native_config)
					for _, v := range terraform_native_config.([]interface{}) {
						config_block.AddValue(resource_schema_elem.Type, v.(string))
					}
				default:
					Log("ERROR", "resource_schema_elem.Type is unhandled: %#v", resource_schema_elem.Type)
					return fmt.Errorf("resource_schema_elem.Type is unhandled: %#v", resource_schema_elem.Type)
				}
			} else {
				// Make unhandled cases visible
				Log("ERROR", "resource_schema.Elem is unhandled: %#v", resource_schema)

				return fmt.Errorf("resource_schema.Elem is unhandled: %#v", resource_schema)
			}
		default:
			// Make unhandled cases visible
			Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)

			return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
		}

	default:
		// Make unhandled cases visible
		Log("ERROR", "(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
		return fmt.Errorf("(key: %s)resource_schema is unhandled: %#v", config_block.key.Key, resource_schema)
	}

	tf_json_data, tf_err := json.Marshal(&config_block)
	Log("DEBUG", "err: %s, tf json data: %s\n", tf_err, tf_json_data)

	return nil
}
