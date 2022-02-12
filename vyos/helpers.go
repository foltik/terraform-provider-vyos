package vyos

import (
	"fmt"
	"log"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func helper_format_id(key_template string, d *schema.ResourceData) string {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf("[DEBUG] %s: key_template: '%s'", func_name, key_template)

	// Keyfilds used to generate the ID, format: "field|field|field"....
	// ID format: "field=value|field=value"...
	// how ever many is needed for the resource  to be uniqly identified

	var id string

	for _, attr := range helper_key_fields_from_template(key_template) {
		log.Printf("[DEBUG] %s: attr: '%s'", func_name, attr)

		val := fmt.Sprintf("%v", d.Get(attr))

		// val_raw := d.Get(attr)
		// var val string

		// switch val_raw.(type) {
		// default:
		// 	diag.Errorf("Unhandled type: '%T' for attr: '%s'.", val_raw, val_raw)
		// case string:
		// 	val = val_raw.(string)
		// case int:
		// 	val = fmt.Sprintf("%v", val_raw)
		// }

		//id = id + strings.Replace(attr, "_", "-", -1) + "=" + val + "|"
		id = fmt.Sprintf("%s|%s=%s", id, attr, val)
		log.Printf("[DEBUG] %s: id: '%s'", func_name, id)
	}
	id = strings.TrimLeft(id, "|")

	log.Printf("[DEBUG] %s: complete id: '%s'", func_name, id)

	return id
}

func helper_key_fields_from_template(key_template string) []string {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf("[DEBUG] %s: key_tempalte: '%s'", func_name, key_template)

	re := regexp.MustCompile(`\{{[a-z_]+}}`)
	fields := re.FindAllString(key_template, -1)

	for idx, field := range fields {
		log.Printf("[DEBUG] %s: idx: '%d', field: '%s'", func_name, idx, field)

		fields[idx] = strings.Trim(field, `{}`)
	}
	return fields
}

func helper_key_from_template(key_template string, id string, d *schema.ResourceData) string {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf("[DEBUG] %s: key_template: '%s'", func_name, key_template)

	// key_template is a formatable string eg: "firewall name %s rule %s"

	// id_pairs := make(map[string]string)
	// var id_keys []string
	// var id_values []interface{}

	// for _, id_pair := range strings.Split(id, "|") {
	// 	log.Printf("[DEBUG] %s: id_par: '%v'", func_name, id_pair)

	// 	pair := strings.Split(id_pair, "=")
	// 	id_key := pair[0]
	// 	id_value := pair[1]

	// 	id_keys = append(id_keys, id_key)
	// 	id_values = append(id_values, id_value)
	// 	id_pairs[id_key] = id_value
	// }

	key := key_template
	for _, key_field := range helper_key_fields_from_template(key_template) {
		v := fmt.Sprintf("%v", d.Get(key_field))
		log.Printf("[DEBUG] adding field/value: '%s/%s' to template: '%s'", key_field, v, key)
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", key_field), v, 1)

		log.Printf("[DEBUG] %s: key: '%s'", func_name, key)
	}

	return key
}

func helperRemoveFieldsFromSchema(fields []string, schema map[string]*schema.Schema) map[string]*schema.Schema {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	for attr := range schema {

		// Check if schema attr is in field list using sort pkg
		sort.Strings(fields)
		i := sort.SearchStrings(fields, attr)
		if i < len(fields) && fields[i] == attr {
			log.Printf("[DEBUG] %s: removing key field: '%s' from schema", func_name, attr)
			delete(schema, attr)
		} else {
			log.Printf("[DEBUG] %s: allowing key field: '%s' to remain in schema", func_name, attr)
		}
	}
	return schema
}

func helper_config_fields_from_schema(key_template string, s map[string]*schema.Schema) []string {
	// Dumb, but helpful
	pc, _, _, _ := runtime.Caller(0)
	func_name := runtime.FuncForPC(pc).Name()

	log.Printf("[DEBUG] %s: key_template: '%s'", func_name, key_template)

	var fields []string
	for k := range s {
		if !strings.Contains(key_template, fmt.Sprintf("{{%s}}", k)) && !s[k].Computed {
			log.Printf("[DEBUG] %s: adding field: '%s' to config fields", func_name, k)
			fields = append(fields, k)
		} else {
			log.Printf("[DEBUG] %s: NOT adding field: '%s' to config fields", func_name, k)
		}
	}
	return fields
}
