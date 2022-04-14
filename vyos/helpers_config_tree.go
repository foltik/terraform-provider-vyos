package vyos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

/*
TODO: propper doc strings
TODO: is it worth it to create stucts for vyos and terraform style configs so its easier to understand when to use what?
TODO: Add missing comments where applicable
TODO: Migrate resource_config_block over to these helpers when delete function is complete
*/

/*
#################################################
#
#
# Helpers to read live config based on schema
#
#
#################################################
*/

// Recursively convert VyOS configuration to terraform config based on resource schema
// If a list/set type has MaxItems 1 it is considered to be a configuration block and not a real list on the VyOS side.
//
// Takes a VyOS style config
// Returns a Terraform Style config
func helperSchemaBasedVyosToTerraformWalker(resource_schema map[string]*schema.Schema, vyos_config map[string]interface{}) (map[string]interface{}, error) {
	return_value := make(map[string]interface{})

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf("[DEBUG] %s: START", func_name)

	// Walk the schema map
	for schema_key, schema_body := range resource_schema {

		// Convert schema_key that uses "_" to the vyos_key version which uses "-"
		vyos_key := strings.Replace(schema_key, "_", "-", -1)
		log.Printf("[DEBUG] %s: investigating schema_key '%s', vyos_key '%s'.", func_name, schema_key, vyos_key)

		// Check if VyOS has the current parameter/key configured
		if vyos_value, ok := vyos_config[vyos_key]; ok {

			// Un-JSON the data returned from VyOS
			// TODO is this needed, why did we not need it when fetching the vyos_value from the vyos_config above
			vyos_value_byte, _ := json.Marshal(vyos_config[vyos_key])
			log.Printf(
				"[DEBUG] %s: investigating vyos_value '%v', expected to be type: '%s', schematype: '%T', current live value '%s'",
				func_name, vyos_value, schema_body.Type.String(), schema_body, string(vyos_value_byte),
			)

			// Treat each type of parameter according to their type
			switch schema_body.Type.String() {
			case "TypeMap":
				sub_schema_body := schema_body.Elem.(map[string]*schema.Schema)
				vyos_value := vyos_config[vyos_key].(map[string]interface{})

				// Recurse for map
				if ret, err := helperSchemaBasedVyosToTerraformWalker(sub_schema_body, vyos_value); err != nil {
					// Raise errors to the top if encountered during reccrusion
					return nil, err
				} else {
					return_value[schema_key] = ret
				}

			case "TypeList", "TypeSet":

				log.Printf("[DEBUG] %s: schema_key '%s' has MaxItems set to '%v'", func_name, schema_key, schema_body.MaxItems)

				// This is if we have a config block, which means schema should have MaxItems set to 1 (I dont know if this is a workaround or the correct way)
				// The schema would still be a list/set, but VyOS would send back a map, so we recurse.
				//
				// Else treat it as a normal list/set
				if schema_body.MaxItems == 1 {
					sub_schema_body := schema_body.Elem.(*schema.Resource).Schema
					vyos_value := vyos_config[vyos_key].(map[string]interface{})

					// Recurse for map
					if ret, err := helperSchemaBasedVyosToTerraformWalker(sub_schema_body, vyos_value); err != nil {
						// Raise errors to the top if encountered during reccrusion
						return nil, err
					} else {
						// Turn the value back into a list so it matches the schema
						return_value[schema_key] = []interface{}{ret}
					}

				} else {

					// Loop over each vyos_value (list elements returned from VyOS)

					vyos_value := vyos_config[vyos_key]
					log.Printf("[DEBUG] %s: vyos_value [type]: '%T' [value]: '%v'", func_name, vyos_value, vyos_value)

					for vyos_value_index, vyos_value := range vyos_value.([]interface{}) {
						log.Printf("[DEBUG] %s: investigating vyos_value_index '%d'.", func_name, vyos_value_index)

						// Loop over each parameter in schema
						for sub_schema_key, sub_schema_body := range schema_body.Elem.(*schema.Resource).Schema {

							fmt.Printf("[DEBUG] %s: sub_schema_key: '%s' of golang type: '%T' expects schema type: '%s'",
								func_name, sub_schema_key, sub_schema_body.Elem, sub_schema_body.Type.String(),
							)

							// Treat each parameter according to their type
							switch schema_body.Type.String() {
							case "TypeMap":
								sub_schema_body := schema_body.Elem.(map[string]*schema.Schema)
								vyos_value := vyos_config[vyos_key].(map[string]interface{})

								// Recurse for map
								if ret, err := helperSchemaBasedVyosToTerraformWalker(sub_schema_body, vyos_value); err != nil {
									// Raise errors to the top if encountered during reccrusion
									return nil, err
								} else {
									// Append the return_value to the list
									return_value[schema_key] = append(
										return_value[schema_key].([]interface{}),
										ret,
									)
								}
							default:
								// TODO can this be recursed by sending schema and such to self, to avoid having 2 blocks handeling default <-> primitive types

								// Append any primitive type to return_value list
								switch sub_schema_body.Type.String() {
								case "TypeBool":
									// Do bool check since API returns "key:{}" for true and nil for false, instead of a nice and usefull true/false
									if vyos_value != nil {
										vyos_value = true
									} else {
										vyos_value = false
									}
								default:
								}
								//ret := helperSchemaBasedVyosToTerraformSetter(sub_schema_body, vyos_value)
								return_value[schema_key] = append(
									return_value[schema_key].([]interface{}),
									vyos_value,
								)

							}
						}
					}
				}

			default:
				// Append any primitive type to return_value list
				switch schema_body.Type.String() {
				case "TypeBool":
					// Do bool check since API returns "key:{}" for true and nil for false, instead of a nice and usefull true/false
					if vyos_value != nil {
						vyos_value = true
					} else {
						vyos_value = false
					}
				default:
				}

				return_value[schema_key] = vyos_value
			}
		}
	}

	log.Printf(
		"[DEBUG] %s: return_value: '%v'",
		func_name, return_value,
	)

	return return_value, nil
}

/*
#################################################
#
#
# Helpers to convert config based on schema to live version
#
#
#################################################
*/

// Recursively convert terraform configuration to VyOS client config tree based on resource schema
// If a list/set type has MaxItems 1 it is considered to be a configuration block and not a real list on the VyOS side.
//
// Takes a Terraform Style config
// Retruns a VyOS style config
func helperSchemaBasedTerraformToVyosWalker(resource_schema map[string]*schema.Schema, terraform_config map[string]interface{}) (map[string]interface{}, error) {
	return_value := make(map[string]interface{})

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf("[DEBUG] %s: START", func_name)

	// Walk the schema map
	for schema_key, schema_body := range resource_schema {

		// Convert schema_key that uses "_" to the vyos_key version which uses "-"
		vyos_key := strings.Replace(schema_key, "_", "-", -1)
		log.Printf("[DEBUG] %s: investigating schema_key '%s', vyos_key '%s'.", func_name, schema_key, vyos_key)

		// Check if VyOS has the current parameter/key configured
		if terraform_value, ok := terraform_config[schema_key]; ok {
			log.Printf(
				"[DEBUG] %s: investigating terraform_value '%s' of type: '%s', schematype: '%T', with value: '%v'.",
				func_name, schema_key, schema_body.Type.String(), terraform_value, terraform_value,
			)

			switch schema_body.Type.String() {
			case "TypeMap":
				sub_schema_body := schema_body.Elem.(map[string]*schema.Schema)
				terraform_value_map := terraform_value.(map[string]interface{})

				// Recurse for map
				if ret, err := helperSchemaBasedTerraformToVyosWalker(sub_schema_body, terraform_value_map); err != nil {
					// Raise errors to the top if encountered during reccrusion
					return nil, err
				} else {
					return_value[vyos_key] = ret
				}

			case "TypeList", "TypeSet":

				var terraform_value_list []interface{}

				// Convert set into list
				if terraform_value_tmp, ok := terraform_value.(*schema.Set); ok {
					log.Printf("[DEBUG] %s: (UPDATE) key: '%s', converting terraform_value from *schema.Set to list", func_name, schema_key)
					terraform_value_list = terraform_value_tmp.List()
				} else {
					terraform_value_list = terraform_value.([]interface{})
				}

				log.Printf("[DEBUG] %s: schema_key '%s' has MaxItems set to '%v'", func_name, schema_key, schema_body.MaxItems)

				// This is if we have a config block, which means schema should have MaxItems set to 1 (I dont know if this is a workaround or the correct way)
				// The schema would still be a list/set, but VyOS would expect a map/config block, so we recurse.
				//
				// Else treat it as a normal list/set
				if schema_body.MaxItems == 1 {
					sub_schema_body := schema_body.Elem.(*schema.Resource).Schema
					log.Printf("[DEBUG] %s: schema_key '%s' terraform_value_tmp '%v'", func_name, schema_key, terraform_value_list)

					// Verify that parameter is set, if not set we get index out of range error
					if len(terraform_value_list) >= 1 {
						terraform_sub_value := terraform_value_list[0].(map[string]interface{})

						// Recurse for map
						if ret, err := helperSchemaBasedTerraformToVyosWalker(sub_schema_body, terraform_sub_value); err != nil {
							// Raise errors to the top if encountered during reccrusion
							return nil, err
						} else {
							// Do NOT turn the value back into a list so it matches what VyOS expects
							return_value[vyos_key] = ret
						}
					}

				} else {
					sub_return_value := make(map[string]interface{})

					// Loop over each terraform_value (list elements returned configured)
					log.Printf("[DEBUG] %s: terraform_value_list '%v'.", func_name, terraform_value_list)
					for terraform_value_index, terraform_sub_value := range terraform_value_list {
						log.Printf("[DEBUG] %s: terraform_value_index: [type]: '%T' [value]: '%v'", func_name, terraform_value_index, terraform_value_index)
						log.Printf("[DEBUG] %s: terraform_sub_value: [type]: '%T' [value]: '%v'", func_name, terraform_sub_value, terraform_sub_value)

						// Loop over each parameter in schema
						sub_schema := schema_body.Elem.(*schema.Resource).Schema
						log.Printf("[DEBUG] %s: sub_schema: [type]: '%T' [value]: '%v'", func_name, sub_schema, sub_schema)
						for sub_schema_key, sub_schema_body := range sub_schema {

							log.Printf("[DEBUG] %s: sub_schema_key: '%s', expects schema type: '%s'",
								func_name, sub_schema_key, sub_schema_body.Type.String(),
							)
							terraform_sub_value := terraform_sub_value.(map[string]interface{})

							// Convert schema_key that uses "_" to the vyos_key version which uses "-"
							sub_vyos_key := strings.Replace(sub_schema_key, "_", "-", -1)
							log.Printf("[DEBUG] %s: investigating sub_schema_key '%s', sub_vyos_key '%s'.", func_name, sub_schema_key, sub_vyos_key)

							// Treat each parameter according to their type
							switch sub_schema_body.Type.String() {
							case "TypeMap":

								// Recurse for map
								if ret, err := helperSchemaBasedTerraformToVyosWalker(sub_schema_body.Elem.(*schema.Resource).Schema, terraform_sub_value); err != nil {
									// Raise errors to the top if encountered during reccrusion
									return nil, err
								} else {
									// Append the sub_return_value to the list
									sub_return_value[sub_vyos_key] = append(
										sub_return_value[sub_vyos_key].([]interface{}),
										ret,
									)
								}
							default:
								// TODO can this be recursed by sending schema and such to self, to avoid having 2 blocks handeling default <-> primitive types

								// Append any primitive type to sub_return_value list
								ret := helperSchemaBasedTerraformToVyosSetter(sub_schema_body, terraform_sub_value[sub_schema_key])
								if ret != nil {

									// Cant typecast inside append since the sub_return_value[schema_key] might be nil
									var return_slot []interface{}
									if sub_return_value[sub_vyos_key] != nil {
										return_slot = sub_return_value[sub_vyos_key].([]interface{})
									}
									switch sub_schema_body.Type.String() {
									case "TypeBool":
										if ret != nil {
											sub_return_value[sub_vyos_key] = append(
												return_slot,
												"",
											)
										}
									default:
										sub_return_value[sub_vyos_key] = append(
											return_slot,
											ret,
										)
									}
								}
							}
						}
					}
					return_value[vyos_key] = sub_return_value
				}
			default:
				// Append any primitive type to return_value list
				ret := helperSchemaBasedTerraformToVyosSetter(schema_body, terraform_value)
				if ret != nil {

					// Cant typecast inside append since the return_value[schema_key] might be nil
					var return_slot []interface{}
					if return_value[vyos_key] != nil {
						return_slot = return_value[vyos_key].([]interface{})
					}

					switch schema_body.Type.String() {
					case "TypeBool":

						return_value[vyos_key] = append(
							return_slot,
							"",
						)

					default:
						return_value[vyos_key] = append(
							return_slot,
							ret,
						)
					}
				}
			}
		}
	}

	return return_value, nil
}

func helperSchemaBasedTerraformToVyosSetter(resource_schema *schema.Schema, config interface{}) interface{} {
	var return_value interface{}

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf(
		"[DEBUG] %s: [IN] schema type: '%s', value type: '%T', value: '%v'",
		func_name,
		resource_schema.Type.String(),
		config,
		config,
	)

	switch resource_schema.Type.String() {
	case "TypeBool":

		// ! This needs to be ad-hoc handled where function is called from.... terrible design
		if config == true {
			return_value = "true"
		} else {
			return_value = ""
		}
	case "TypeInt":
		// TODO why did nothing else seem to work, this is such a over engineered converstion of int -> string
		return_value = strconv.FormatInt(int64(config.(int)), 10)
	default:
		return_value = fmt.Sprintf("%s", config)
	}

	log.Printf(
		"[DEBUG] %s: [OUT] return type: '%T', value: '%v'",
		func_name,
		return_value,
		return_value,
	)

	// If the setting is not present return nil so we dont try to configure empty values
	if return_value == "" {
		return_value = nil
	}
	return return_value
}

// Takes two terraform style configs
// Return the difference between the configs as a Terraform Style config, and a []string with deleted parameters
func helperSchemaDiff(old_config map[string]interface{}, new_config map[string]interface{}) (map[string]interface{}, []string, error) {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	// return objects
	updates := make(map[string]interface{})
	deletes := []string{}

	// Merge old and new keys to get a complete key list
	// Append "old" keys
	keys := []string{}
	for key := range old_config {
		log.Printf("[DEBUG] %s: (keys to inspect) adding (old) key: '%s'", func_name, key)
		keys = append(keys, key)
	}

	// Append "new" keys, if they are not already added
	for key := range new_config {
		sort.Strings(keys)
		i := sort.SearchStrings(keys, key)
		if i < len(keys) && keys[i] == key {
			log.Printf("[DEBUG] %s: (keys to inspect) already have key: '%s'", func_name, key)
		} else {
			log.Printf("[DEBUG] %s: (keys to inspect) adding (new) key: '%s'", func_name, key)
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		old_value, old_ok := old_config[key]
		new_value, new_ok := new_config[key]

		// is there a better way to detect "empty" string values from the terraform resrouce?
		if old_value == "" {
			old_ok = false
		}
		if new_value == "" {
			new_ok = false
		}

		if old_ok && new_ok {
			// if in old and new: update
			log.Printf("[DEBUG] %s: (UPDATE) key: '%s', old_value type: '%T', new_value type: '%T'", func_name, key, old_value, new_value)
			log.Printf("[DEBUG] %s: (UPDATE) key: '%s', old_value value: '%v', new_value value: '%v'", func_name, key, old_value, new_value)

			switch old_value.(type) {

			case map[string]interface{}:
				log.Printf("[DEBUG] %s: (UPDATE) key: '%s', recurse for map", func_name, key)

				ups, dels, err := helperSchemaDiff(old_value.(map[string]interface{}), new_value.(map[string]interface{}))

				if err != nil {
					return nil, nil, fmt.Errorf("[ERROR] %s: error walking through key: '%s', '%v'", func_name, key, err)
				}

				updates[key] = ups
				for _, del := range dels {
					log.Printf("[DEBUG] %s: appending delete:'%s %s'", func_name, key, del)
					deletes = append(deletes, fmt.Sprintf("%s %s", key, del))
				}

				continue

			case []interface{}, *schema.Set:
				log.Printf("[DEBUG] %s: (UPDATE) key: '%s', loop over list", func_name, key)

				if o_value, ok := old_value.(*schema.Set); ok {
					log.Printf("[DEBUG] %s: (UPDATE) key: '%s', converting old_value from *schema.Set to list", func_name, key)
					old_value = o_value.List()
				}
				if n_value, ok := new_value.(*schema.Set); ok {
					log.Printf("[DEBUG] %s: (UPDATE) key: '%s', converting new_value from *schema.Set to list", func_name, key)
					new_value = n_value.List()
				}

				// deleted values
				for idx, old_sub_value := range old_value.([]interface{}) {

					switch old_sub_value.(type) {

					case map[string]interface{}:
						log.Printf("[DEBUG] %s: (UPDATE) key: '%s', old_sub_value type: '%T', assume new_value has only 1 element and recurse as map", func_name, key, old_sub_value)

						new_value_list := new_value.([]interface{})
						ups, dels, err := helperSchemaDiff(
							old_sub_value.(map[string]interface{}),
							new_value_list[0].(map[string]interface{}))

						if err != nil {
							return nil, nil, fmt.Errorf("[ERROR] %s: error walking through key: '%s', '%v'", func_name, key, err)
						}

						// Cant typecast inside append since the return_value[schema_key] might be nil
						var update_slot []interface{}
						if updates[key] != nil {
							update_slot = updates[key].([]interface{})
						}

						updates[key] = append(update_slot, ups)

						for _, del := range dels {
							log.Printf("[DEBUG] %s: appending delete:'%s %s'", func_name, key, del)
							deletes = append(deletes, fmt.Sprintf("%s %s", key, del))
						}

						// continue loop, this should in theory always complete the loop, but idk how to break out from the loop from within a switch
						continue
					}

					sub_value_found := false

					for _, new_sub_value := range new_value.([]interface{}) {
						log.Printf("[DEBUG] %s: (UPDATE) [DEL] key: '%s', old_sub_value: '%v', new_sub_value: '%v'", func_name, key, old_sub_value, new_sub_value)

						if new_sub_value == old_sub_value {
							log.Printf("[DEBUG] %s: (UPDATE) [DEL] unchanged! key: '%s', index: '%d', value: '%v'", func_name, key, idx, old_value)
							sub_value_found = true
							break
						}
					}

					if !sub_value_found {
						log.Printf("[DEBUG] %s: (UPDATE) [DEL] removed! key: '%s', index: '%d', value: '%v'", func_name, key, idx, old_value)
						deletes = append(deletes, fmt.Sprintf("%s %s", key, old_sub_value))
					}
				}

				// new values
				for idx, new_sub_value := range new_value.([]interface{}) {
					switch new_sub_value.(type) {

					case map[string]interface{}:
						log.Printf("[DEBUG] %s: (UPDATE) key: '%s', new_sub_value type: '%T', this should already have been handled above", func_name, key, new_sub_value)

						continue
					}

					sub_value_found := false

					for _, old_sub_value := range old_value.([]string) {
						if old_sub_value == new_sub_value {
							log.Printf("[DEBUG] %s: (UPDATE) [ADD] unchanged! key: '%s', index: '%d', value: '%v'", func_name, key, idx, new_value)
							sub_value_found = true
							break
						}
					}

					if !sub_value_found {
						log.Printf("[DEBUG] %s: (UPDATE) [ADD] added! key: '%s', index: '%d', value: '%v'", func_name, key, idx, new_value)
						deletes = append(deletes, fmt.Sprintf("%s %s", key, new_sub_value))
					}
				}

			default:
				if old_value == new_value {
					log.Printf("[DEBUG] %s: (UPDATE) no change detected. key: '%s', value: '%v'", func_name, key, old_value)
					continue
				}

				log.Printf("[DEBUG] %s: (UPDATE) CHANGE detected. key: '%s', old_value: '%v', new_value: '%v'", func_name, key, old_value, new_value)

				// Add the new configuration value to the return object
				updates[key] = new_value
			}

			continue

		} else if old_ok && (!new_ok) {
			// if in old but not new: deleted
			log.Printf("[DEBUG] %s: (DELETE) key: '%s', old_value: '%v'", func_name, key, old_value) //TODO whole path needed

			deletes = append(deletes, key+" "+old_value.(string))
			continue

		} else if (!old_ok) && new_ok {
			// if in new but not old: new parameter added
			log.Printf("[DEBUG] %s: (ADD) key: '%s', new_value: '%v'", func_name, key, new_value)

			updates[key] = new_value
			continue

		} else if (!old_ok) && (!new_ok) {
			log.Printf("[DEBUG] %s: (DO NOTHING) key: '%s', new_value: '%v', old_value: '%v'", func_name, key, new_value, old_value)
			continue
		} else {
			return nil, nil, fmt.Errorf("[ERROR] %s: dont know how to handle key: '%s', old_config: '%v', new_config: '%v'", func_name, key, old_config, new_config)
		}
	}

	log.Printf("[DEBUG] %s: returning deletes:", func_name)
	for _, del := range deletes {
		log.Printf("[DEBUG] %s: '%s'", func_name, del)
	}

	return updates, deletes, nil
}

/*
#################################################
#
#
# Helper to read live config based on schema
#
#
#################################################
*/

func helperSchemaBasedConfigRead(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	id := d.Id()
	key := helper_key_from_template(key_template, id, d)

	log.Printf("[DEBUG] %s: Reading tree at key '%s'", func_name, key)

	vyos_config, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Keep only attributes defined in the schema
	config, err := helperSchemaBasedVyosToTerraformWalker(resource_schema, vyos_config)
	if err != nil {
		return diag.FromErr(err)
	}

	// Easiest way to allow ImportStatePassthroughContext to work is to set the keys needed for the ID
	for k, v := range config {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

/*
#################################################
#
#
# Helper to create live config based on schema
#
#
#################################################
*/

func helperSchemaBasedConfigCreate(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema, prerequsite_key_templates ...string) diag.Diagnostics {
	var diags diag.Diagnostics

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	// Set the ID for the resource about to be configured
	id := helper_format_id(key_template, d)
	key := helper_key_from_template(key_template, id, d)

	remove_fields := helper_key_fields_from_template(key_template)
	remove_fields = append(remove_fields, "id")

	resource_schema = helperRemoveFieldsFromSchema(remove_fields, resource_schema)

	// Check if required configs / parents / structure exists
	for _, prereq_template := range prerequsite_key_templates {
		prereq := helper_key_from_template(prereq_template, id, d)
		log.Printf("[DEBUG] %s: Looking for pre-requisite key '%s'", func_name, prereq)
		vyos_prereq_config, err := client.Config.ShowTree(prereq)
		if vyos_prereq_config == nil {
			return diag.Errorf("[ERROR] %s: Could not find pre-requisite key '%s'", func_name, prereq)
		} else if err != nil {
			return diag.FromErr(err)
		}
	}

	// Check if config already exists
	log.Printf("[DEBUG] %s: Reading tree at key '%s'", func_name, key)
	vyos_config, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	} else if vyos_config != nil {
		return diag.Errorf("[ERROR] %s: Config path '%s' already exists, try a resource import instead.", func_name, key)
	}

	tf_config := make(map[string]interface{})

	for k := range resource_schema {
		tf_config[k] = d.Get(k)
	}

	converted_config, err := helperSchemaBasedTerraformToVyosWalker(resource_schema, tf_config)
	if err != nil {
		return diag.FromErr(err)
	}

	config := map[string]interface{}{
		key: converted_config,
	}

	err = client.Config.SetTree(config)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

/*
#################################################
#
#
# Helpers to update live config based on schema
#
#
#################################################
*/

func helperSchemaBasedConfigUpdate(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	key := helper_key_from_template(key_template, d.Id(), d)

	// Placeholder for keys that has been deteled
	deleted := []string{}

	// Placeholders for what needs investigating to find the exact changes
	old_changed := make(map[string]interface{})
	new_changed := make(map[string]interface{})

	for schema_key := range resource_schema {
		if d.HasChanges(schema_key) {
			_, ok := d.GetOk(schema_key)

			if ok {
				log.Printf("[DEBUG] %s: detected change for schema_key: '%s'", func_name, schema_key)

				// Add attribute to be inspected for what is changed.
				old_value, new_value := d.GetChange(schema_key)
				old_changed[schema_key] = old_value
				new_changed[schema_key] = new_value
			} else {
				log.Printf("[DEBUG] %s: detected removal of schema_key: '%s'", func_name, schema_key)

				// Delete whole toplevel parameter/attribute
				deleted = append(deleted, key+" "+schema_key)
			}
		} else {
			log.Printf("[DEBUG] %s: no change for schema_key: '%s'", func_name, schema_key)
		}
	}

	log.Printf("[DEBUG] %s: old_changed dump: '%v'", func_name, old_changed)
	log.Printf("[DEBUG] %s: new_changed dump: '%v'", func_name, new_changed)

	updates_terraform, dels, err := helperSchemaDiff(old_changed, new_changed)
	if err != nil {
		return diag.FromErr(err)
	}

	updates_vyos, err := helperSchemaBasedTerraformToVyosWalker(resource_schema, updates_terraform)
	if err != nil {
		return diag.FromErr(err)
	}

	deleted_tf := append(deleted, dels...)

	// Convert schema_key that uses "_" to the vyos_key version which uses "-"
	for _, del_tf := range deleted_tf {
		del_vyos := strings.Replace(del_tf, "_", "-", -1)
		log.Printf("[DEBUG] %s: converting delete param from tf: '%s', to vyos: '%s'.", func_name, del_tf, del_vyos)
		deleted = append(deleted, del_vyos)
	}

	config := map[string]interface{}{
		key: updates_vyos,
	}

	err = client.Config.SetTree(config)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Config.Delete(deleted...)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

/*
#################################################
#
#
# Helpers to delete live config based on schema
#
#
#################################################
*/

func helperSchemaBasedConfigDelete(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	// TODO add check for children / configs outside of the schema (recursively)

	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	// Convert schema_key that uses "_" to the vyos_key version which uses "-"
	// for _, del_tf := range deleted_tf {
	// 	del_vyos := strings.Replace(del_tf, "_", "-", -1)
	// 	log.Printf("[DEBUG] %s: converting delete param from tf: '%s', to vyos: '%s'.", func_name, del_tf, del_vyos)
	// 	deleted = append(deleted, del_vyos)
	// }

	errDel := client.Config.Delete(key)
	if errDel != nil {
		return diag.FromErr(errDel)
	}

	return diags
}
