package vyos

import (
	"context"
	"encoding/json"
	"log"
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

func helperSchemaBasedConfigLiveToConfigMap(resource_schema map[string]*schema.Schema, live_config map[string]interface{}) map[string]interface{} {
	return_value := make(map[string]interface{})

	for schema_key, schema_value := range resource_schema {
		// Convert schema keys "_" to the VyOS version of the attributet name which should usually be "-""
		live_key := strings.Replace(schema_key, "_", "-", -1)

		log.Printf("[DEBUG] helperSchemaBasedConfigLiveToConfigMap: investigating schema_key '%s', live_key '%s'.", schema_key, live_key)

		if live_value, ok := live_config[live_key]; ok {

			live_value_byte, _ := json.Marshal(live_config[live_key])
			log.Printf(
				"[DEBUG] helperSchemaBasedConfigLiveToConfigMap: investigating live_value '%s', expected to be schema type '%s', current live value '%s'",
				live_value, schema_value.Type.String(), string(live_value_byte),
			)

			switch schema_value.Type.String() {
			case "TypeMap":
				subconfig := live_config[live_key]
				return_value[schema_key] = helperSchemaBasedConfigLiveToConfigMap(schema_value.Elem.(map[string]*schema.Schema), subconfig.(map[string]interface{}))
			case "TypeList":
				subconfig := live_config[live_key]
				return_value[schema_key] = helperSchemaBasedConfigLiveToConfigList(schema_value.Elem.(*schema.Schema), subconfig.([]interface{}))
			case "TypeSet":
				// Do recursive Set call
			default:
				return_value[schema_key] = helperSchemaBasedConfigLiveToConfigPrimitive(schema_value.Elem.(*schema.Schema), live_value)
			}
		}
	}

	return return_value
}

func helperSchemaBasedConfigLiveToConfigList(resource_schema *schema.Schema, live_config []interface{}) []interface{} {
	return_value := []interface{}{}

	for idx, live_value := range live_config {
		live_value_byte, _ := json.Marshal(live_value)
		log.Printf(
			"[DEBUG] helperSchemaBasedConfigLiveToConfigList: investigating idx '%d', expected to be schema type '%s', current live value '%s'",
			idx, resource_schema.Type.String(), string(live_value_byte),
		)

		switch resource_schema.Type.String() {
		case "TypeMap":
			return_value = append(return_value, helperSchemaBasedConfigLiveToConfigMap(resource_schema.Elem.(map[string]*schema.Schema), live_value.(map[string]interface{})))
		case "TypeList":
			return_value = append(return_value, helperSchemaBasedConfigLiveToConfigList(resource_schema.Elem.(*schema.Schema), live_value.([]interface{})))
		case "TypeSet":
			// Do recursive Set call
		default:
			return_value = append(return_value, helperSchemaBasedConfigLiveToConfigPrimitive(resource_schema.Elem.(*schema.Schema), live_value))
		}
	}

	return return_value
}

func helperSchemaBasedConfigLiveToConfigPrimitive(resource_schema *schema.Schema, live_config interface{}) interface{} {

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

func helperSchemaBasedConfigRead(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	log.Printf("[DEBUG] Reading tree at key '%s'", key)
	live_config, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Keep only attributes defined in the schema
	config := helperSchemaBasedConfigLiveToConfigMap(resource_schema, live_config)

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
# Helpers to create live config based on schema
#
#
#################################################
*/

func helper_config_based_on_schema_create(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set the ID for the resource about to be configured
	id := helper_format_id(key_template, d)
	key := helper_key_from_template(key_template, id, d)

	// Check if config already exists
	configs, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	}
	// Dont care about sub config trees
	for k, v := range configs {
		switch v.(type) {
		default:
			continue
		case string:
			return diag.Errorf("Configuration '%s' already exists and has '%s' set, try a resource import instead.", key, k)
		case int:
			return diag.Errorf("Configuration '%s' already exists and has '%s' set, try a resource import instead.", key, k)
		}
	}

	live_attrs := []interface{}{}
	for _, attr := range helper_config_fields_from_schema(key_template, resource_schema) {
		live_attr := strings.Replace(attr, "_", "-", -1)

		value := d.Get(attr)

		config_schema_type := resource_schema[attr].Type
		switch config_schema_type.String() {
		case "TypeBool":
			if value == true {
				live_attrs = append(live_attrs, live_attr)
			} else {
				// Dont set any value if false
				continue
			}
		default:
			live_attrs = append(live_attrs, map[string]interface{}{live_attr: value})
		}
	}

	live_config := map[string]interface{}{
		key: live_attrs,
	}

	err = client.Config.SetTree(live_config)
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

func helper_config_based_on_schema_update(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	var live_delete_attrs []string
	live_set_attrs := []interface{}{}
	for _, config_field := range helper_config_fields_from_schema(key_template, resource_schema) {
		if d.HasChanges(config_field) {
			config_schema_type := resource_schema[config_field].Type

			value, ok := d.GetOk(config_field)
			live_config_field := strings.Replace(config_field, "_", "-", -1)
			if ok {

				switch config_schema_type.String() {
				case "TypeBool":
					if value == true {
						live_set_attrs = append(live_set_attrs, live_config_field)
						continue
					} else {
						live_delete_attrs = append(live_delete_attrs, live_config_field)
						continue
					}
				}
				live_set_attrs = append(live_set_attrs, map[string]interface{}{live_config_field: value})
			} else {
				live_delete_attrs = append(live_delete_attrs, live_config_field)

			}
		}
	}

	live_set_config := map[string]interface{}{
		key: live_set_attrs,
	}
	errSet := client.Config.SetTree(live_set_config)
	if errSet != nil {
		return diag.FromErr(errSet)
	}

	var live_delete_keys []string
	for _, live_delete_attr := range live_delete_attrs {
		live_delete_keys = append(live_delete_keys, key+" "+live_delete_attr)
	}
	errDel := client.Config.Delete(live_delete_keys...)
	if errDel != nil {
		return diag.FromErr(errDel)
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

func helper_config_based_on_schema_delete(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, resource_schema map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	var live_delete_keys []string
	for _, config_field := range helper_config_fields_from_schema(key_template, resource_schema) {
		live_config_field := strings.Replace(config_field, "_", "-", -1)
		live_delete_keys = append(live_delete_keys, key+" "+live_config_field)
	}

	errDel := client.Config.Delete(live_delete_keys...)
	if errDel != nil {
		return diag.FromErr(errDel)
	}

	return diags
}
