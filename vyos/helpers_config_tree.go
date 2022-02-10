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
#################################################
#
#
# Helpers to read live config based on schema
#
#
#################################################
*/

func helperSchemaBasedVyosToTerraformWalker(resource_schema map[string]*schema.Schema, live_config map[string]interface{}) (map[string]interface{}, error) {
	var err error
	return_value := make(map[string]interface{})

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	for schema_key, schema_value := range resource_schema {
		// Convert schema keys "_" to the VyOS version of the attributet name which should usually be "-""
		live_key := strings.Replace(schema_key, "_", "-", -1)

		log.Printf("[DEBUG] %s: investigating schema_key '%s', live_key '%s'.",
			func_name, schema_key, live_key)

		if live_value, ok := live_config[live_key]; ok {

			live_value_byte, _ := json.Marshal(live_config[live_key])
			log.Printf(
				"[DEBUG] %s: investigating live_value '%v', expected to be type: '%s', schematype: '%T', current live value '%s'",
				func_name, live_value, schema_value.Type.String(), schema_value, string(live_value_byte),
			)

			switch schema_value.Type.String() {
			case "TypeMap":
				subconfig := live_config[live_key]
				return_value[schema_key], err = helperSchemaBasedVyosToTerraformWalker(
					schema_value.Elem.(map[string]*schema.Schema),
					subconfig.(map[string]interface{}),
				)
			case "TypeList", "TypeSet":
				subconfig := live_config[live_key]

				// This is if we have a config block (kind of a workaround), which means schema should have MaxItems set to 1
				// The schema would still be a List, but VyOS would send back a map, so just treat it as such.
				// hoever the schema needs some fixing up.
				if schema_value.MaxItems == 1 {
					sub_schema := schema_value.Elem.(*schema.Resource).Schema
					var v map[string]interface{}
					v, err = helperSchemaBasedVyosToTerraformWalker(
						sub_schema,
						subconfig.(map[string]interface{}),
					)

					// turn the value back into a list so it matches the shema
					return_value[schema_key] = []interface{}{v}

				} else {

					//return_value[schema_key] = helperSchemaBasedConfigLiveToConfigList(schema_value.Elem.(*schema.Schema), subconfig.([]interface{}))

					for subconfig_idx, subconfig_element := range subconfig.([]interface{}) {
						log.Printf(
							"[DEBUG] %s: investigating index '%d'.",
							func_name, subconfig_idx,
						)

						var sub_schema map[string]*schema.Schema
						is_primitive := false

						switch schema_value.Elem.(type) {
						case map[string]*schema.Schema:
							sub_schema = schema_value.Elem.(map[string]*schema.Schema)
						case *schema.Resource:
							sub_schema = schema_value.Elem.(*schema.Resource).Schema
						case *schema.Schema:
							is_primitive = true
						default:
							return nil, fmt.Errorf("[DEBUG] %s: schema for key: '%s' of type: '%T' not handled", func_name, schema_key, schema_value.Elem)

						}

						if is_primitive {
							return_value[schema_key] = helperSchemaTerraformToVyosSetter(
								schema_value.Elem.(*schema.Schema),
								subconfig_element,
							)
						} else {
							return_value[schema_key], err = helperSchemaBasedTerraformToVyosWalker(
								sub_schema,
								subconfig_element.(map[string]interface{}),
							)
						}
					}

				}
			default:
				return_value[schema_key] = helperSchemaBasedVyosToTerraformSetter(schema_value, live_value)
			}
		}
	}

	log.Printf(
		"[DEBUG] %s: return_value: '%v'",
		func_name, return_value,
	)

	return return_value, err
}

func helperSchemaBasedVyosToTerraformSetter(resource_schema *schema.Schema, live_config interface{}) interface{} {

	var return_value interface{}

	switch resource_schema.Type.String() {
	case "TypeBool":
		// Do bool check since API returns "key:{}" for true and "" for false, instead of a nice and usefull true/false
		if live_config != nil {
			return_value = true
		} else {
			return_value = false
		}
	default:
		return_value = live_config
	}

	return return_value
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

func helperSchemaBasedTerraformToVyosWalker(resource_schema map[string]*schema.Schema, config map[string]interface{}) (map[string]interface{}, error) {
	var err error
	return_value := make(map[string]interface{})

	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	for schema_key, schema_value := range resource_schema {
		// Convert schema keys "_" to the VyOS version of the attributet name which should usually be "-""
		//live_key := strings.Replace(schema_key, "_", "-", -1)

		log.Printf("[DEBUG] %s: schema_key '%s'", func_name, schema_key)

		if subconfig, ok := config[schema_key]; ok {
			log.Printf(
				"[DEBUG] %s: investigating subconfig '%s' of type: '%s', schematype: '%T', with value: '%v'.",
				func_name, schema_key, schema_value.Type.String(), subconfig, subconfig,
			)

			switch schema_value.Type.String() {
			case "TypeMap":
				return_value[schema_key], err = helperSchemaBasedTerraformToVyosWalker(
					schema_value.Elem.(map[string]*schema.Schema),
					subconfig.(map[string]interface{}),
				)
			case "TypeList", "TypeSet":

				if schema_value.Type.String() == "TypeList" {

					subconfig = subconfig.([]interface{})
				} else if schema_value.Type.String() == "TypeSet" {

					subconfig = subconfig.(*schema.Set).List()
				}

				for subconfig_idx, subconfig_element := range subconfig.([]interface{}) {
					log.Printf(
						"[DEBUG] %s: investigating index '%d'.",
						func_name, subconfig_idx,
					)

					var sub_schema map[string]*schema.Schema
					is_primitive := false

					switch schema_value.Elem.(type) {
					case map[string]*schema.Schema:
						sub_schema = schema_value.Elem.(map[string]*schema.Schema)
					case *schema.Resource:
						sub_schema = schema_value.Elem.(*schema.Resource).Schema
					case *schema.Schema:
						is_primitive = true
					default:
						return nil, fmt.Errorf("[DEBUG] %s: schema for key: '%s' of type: '%T' not handled", func_name, schema_key, schema_value.Elem)

					}

					if is_primitive {
						return_value[schema_key] = helperSchemaTerraformToVyosSetter(
							schema_value.Elem.(*schema.Schema),
							subconfig_element,
						)
					} else {
						return_value[schema_key], err = helperSchemaBasedTerraformToVyosWalker(
							sub_schema,
							subconfig_element.(map[string]interface{}),
						)
					}

				}
			default:
				return_value[schema_key] = helperSchemaTerraformToVyosSetter(
					schema_value,
					subconfig,
				)
			}
		}
	}

	return return_value, err
}

func helperSchemaTerraformToVyosSetter(resource_schema *schema.Schema, config interface{}) interface{} {
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

		// Do bool check since API returns "key:{}" for true and "" for false, instead of a nice and usefull true/false
		if config == true {
			return_value = "true"
		} else {
			return_value = ""
		}
	case "TypeInt":
		// ? why did nothing else seem to work, this is such a over engineered converstion of int -> string
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
	return return_value
}

func helperSchemaDiff(old_config map[string]interface{}, new_config map[string]interface{}) (map[string]interface{}, []string, error) {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	// return objects
	updates := make(map[string]interface{})
	deletes := []string{}

	// Merge old and new keys to get a complete key list
	keys := []string{}
	for key := range old_config {
		keys = append(keys, key)
	}

	for key := range new_config {
		sort.Strings(keys)
		i := sort.SearchStrings(keys, key)
		if i < len(keys) && keys[i] == key {
			log.Printf("[DEBUG] %s: (keys to inspect) already have key: '%s'", func_name, key)
		} else {
			log.Printf("[DEBUG] %s: (keys to inspect) adding key: '%s'", func_name, key)
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		old_value, old_ok := old_config[key]
		new_value, new_ok := new_config[key]

		if old_ok && new_ok {
			// if in old and new: update
			log.Printf("[DEBUG] %s: (UPDATE) key: '%s', old_value type: '%T', new_value type: '%T'", func_name, key, old_value, new_value)

			switch old_value.(type) {

			case map[string]interface{}:
				log.Printf("[DEBUG] %s: (UPDATE) key: '%s', recurse for map", func_name, key)

				ups, dels, err := helperSchemaDiff(old_value.(map[string]interface{}), new_value.(map[string]interface{}))

				if err != nil {
					return nil, nil, fmt.Errorf("[ERROR] %s: error walking through key: '%s'", func_name, key)
				}

				updates[key] = ups
				for _, del := range dels {
					deletes = append(deletes, fmt.Sprintf("%s %s", key, del))
				}

				continue

			case []interface{}, *schema.Set:
				log.Printf("[DEBUG] %s: (UPDATE) key: '%s', loop over list", func_name, key)

				if o_value, ok := old_value.(*schema.Set); ok {
					old_value = o_value.List()
					log.Printf("[DEBUG] %s: (UPDATE) key: '%s', converting old_value from *schema.Set to list", func_name, key)
				}
				if n_value, ok := new_value.(*schema.Set); ok {
					new_value = n_value.List()
					log.Printf("[DEBUG] %s: (UPDATE) key: '%s', converting new_value from *schema.Set to list", func_name, key)
				}

				// deleted values
				for idx, old_sub_value := range old_value.([]interface{}) {

					switch old_sub_value.(type) {

					case map[string]interface{}:
						log.Printf("[DEBUG] %s: (UPDATE) key: '%s', old_sub_value type: '%T', assume new_value has only 1 element and recurse as map", func_name, key, old_sub_value)
						ups, dels, err := helperSchemaDiff(old_sub_value.(map[string]interface{}), new_value.([]interface{})[0].(map[string]interface{}))

						if err != nil {
							return nil, nil, fmt.Errorf("[ERROR] %s: error walking through key: '%s'", func_name, key)
						}

						updates[key] = ups
						for _, del := range dels {
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
			log.Printf("[DEBUG] %s: (DELETE) key: '%s', old_value: '%v'", func_name, key, old_value)

			deletes = append(deletes, key)
			continue

		} else if (!old_ok) && new_ok {
			// if in new but not old: new parameter added
			log.Printf("[DEBUG] %s: (ADD) key: '%s', new_value: '%v'", func_name, key, new_value)

			updates[key] = new_value
			continue

		} else {
			return nil, nil, fmt.Errorf("[ERROR] %s: dont know how to handle key: '%s', old_config: '%v', new_config: '%v'", func_name, key, old_config, new_config)
		}
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

	live_config, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Keep only attributes defined in the schema
	config, err := helperSchemaBasedVyosToTerraformWalker(resource_schema, live_config)
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

func helperSchemaBasedConfigCreate(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
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

	// Check if config already exists
	log.Printf("[DEBUG] %s: Reading tree at key '%s'", func_name, key)
	live_config, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	}

	if live_config != nil {
		return diag.Errorf("[ERROR] %s: Config path '%s' already exists, try a resource import instead.", func_name, key)
	}

	tf_config := make(map[string]interface{})

	for k, _ := range resource_schema {
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
	// ! TODO
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

	updates, dels, err := helperSchemaDiff(old_changed, new_changed)
	deleted = append(deleted, dels...)

	if err != nil {
		return diag.FromErr(err)
	}

	config := map[string]interface{}{
		key: updates,
	}

	err = client.Config.SetTree(config)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Config.Delete(deleted...)
	if err != nil {
		return diag.FromErr(err)
	}

	// var live_delete_attrs []string
	// live_set_attrs := []interface{}{}
	// for _, config_field := range helper_config_fields_from_schema(key_template, resource_schema) {
	// 	if d.HasChanges(config_field) {
	// 		config_schema_type := resource_schema[config_field].Type

	// 		value, ok := d.GetOk(config_field)
	// 		live_config_field := strings.Replace(config_field, "_", "-", -1)
	// 		if ok {

	// 			switch config_schema_type.String() {
	// 			case "TypeBool":
	// 				if value == true {
	// 					live_set_attrs = append(live_set_attrs, live_config_field)
	// 					continue
	// 				} else {
	// 					live_delete_attrs = append(live_delete_attrs, live_config_field)
	// 					continue
	// 				}
	// 			}
	// 			live_set_attrs = append(live_set_attrs, map[string]interface{}{live_config_field: value})
	// 		} else {
	// 			live_delete_attrs = append(live_delete_attrs, live_config_field)

	// 		}
	// 	}
	// }

	// live_set_config := map[string]interface{}{
	// 	key: live_set_attrs,
	// }
	// errSet := client.Config.SetTree(live_set_config)
	// if errSet != nil {
	// 	return diag.FromErr(errSet)
	// }

	// var live_delete_keys []string
	// for _, live_delete_attr := range live_delete_attrs {
	// 	live_delete_keys = append(live_delete_keys, key+" "+live_delete_attr)
	// }
	// errDel := client.Config.Delete(live_delete_keys...)
	// if errDel != nil {
	// 	return diag.FromErr(errDel)
	// }

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
	// ! TODO add check for children / configs outside of the schema (recursive? yes!)
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	errDel := client.Config.Delete(key)
	if errDel != nil {
		return diag.FromErr(errDel)
	}

	return diags
}
