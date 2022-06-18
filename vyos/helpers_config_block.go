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

func helper_config_block_read(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, s map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	log.Printf("[DEBUG] Reading tree at key '%s'", key)
	live_config, err := client.Config.ShowTree(ctx, key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Keep only attributes defined in config_fields
	config := make(map[string]interface{})
	for _, attr := range helper_config_fields_from_schema(key_template, s) {
		live_attr := strings.Replace(attr, "_", "-", -1)

		config_schema_type := s[attr].Type
		live_attr_byte, _ := json.Marshal(live_config[live_attr])
		log.Printf("[DEBUG] helper_config_block_read: investigating live_attr '%s', expected to be schema type '%s', current live value '%s'", live_attr, config_schema_type.String(), string(live_attr_byte))

		// Conversion operations needed when reading
		switch config_schema_type.String() {
		case "TypeBool":
			if live_config[live_attr] != nil {
				live_config[live_attr] = true
			} else {
				live_config[live_attr] = false
			}
		}

		if live_config[live_attr] != nil {
			config[attr] = live_config[live_attr]
		}
	}

	// Easiest way to allow ImportStatePassthroughContext to work is to set the keys needed for the ID
	for k, v := range config {
		// Convert live keys - to the terraform version of the attributet name which should use _
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func helper_config_block_create(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, s map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set the ID for the resource about to be configured
	id := helper_format_id(key_template, d)
	key := helper_key_from_template(key_template, id, d)

	// Check if config already exists
	configs, err := client.Config.ShowTree(key)
	if err != nil {
		return diag.FromErr(err)
	}
	// Dont care about sub config blocks
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

	live_attrs := []interface{}{} //make(map[string]interface{})
	for _, attr := range helper_config_fields_from_schema(key_template, s) {
		live_attr := strings.Replace(attr, "_", "-", -1)

		value := d.Get(attr)

		config_schema_type := s[attr].Type
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

func helper_config_block_update(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, s map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	var live_delete_attrs []string
	live_set_attrs := []interface{}{}
	for _, config_field := range helper_config_fields_from_schema(key_template, s) {
		if d.HasChanges(config_field) {
			config_schema_type := s[config_field].Type

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

func helper_config_block_delete(ctx context.Context, client *client.Client, key_template string, d *schema.ResourceData, s map[string]*schema.Schema) diag.Diagnostics {
	var diags diag.Diagnostics

	key := helper_key_from_template(key_template, d.Id(), d)

	var live_delete_keys []string
	for _, config_field := range helper_config_fields_from_schema(key_template, s) {
		live_config_field := strings.Replace(config_field, "_", "-", -1)
		live_delete_keys = append(live_delete_keys, key+" "+live_config_field)
	}

	errDel := client.Config.Delete(live_delete_keys...)
	if errDel != nil {
		return diag.FromErr(errDel)
	}

	return diags
}
