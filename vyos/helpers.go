package vyos

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func helper_format_id(key_template string, d *schema.ResourceData) string {
	// Keyfilds used to generate the ID, format: "field|field|field"....
	// ID format: "field=value|field=value"...
	// how ever many is needed for the resource  to be uniqly identified
	var id string
	for _, attr := range helper_key_fields_from_template(key_template) {
		val := d.Get(attr).(string)
		//id = id + strings.Replace(attr, "_", "-", -1) + "=" + val + "|"
		id = fmt.Sprintf("%s|%s=%s", id, attr, val)
	}
	id = strings.TrimLeft(id, "|")

	return id
}

func helper_key_fields_from_template(key_template string) []string {
	re := regexp.MustCompile(`\{{[a-z_]+}}`)
	fields := re.FindAllString(key_template, -1)
	for idx, field := range fields {
		fields[idx] = strings.Trim(field, `{}`)
	}
	return fields
}

func helper_key_from_template(key_template string, id string, d *schema.ResourceData) string {
	// key_template is a formatable string eg: "firewall name %s rule %s"

	id_pairs := make(map[string]string)
	var id_keys []string
	var id_values []interface{}

	for _, id_pair := range strings.Split(id, "|") {
		pair := strings.Split(id_pair, "=")
		id_key := pair[0]
		id_value := pair[1]

		id_keys = append(id_keys, id_key)
		id_values = append(id_values, id_value)
		id_pairs[id_key] = id_value
	}

	key := key_template
	for _, key_field := range helper_key_fields_from_template(key_template) {
		v := d.Get(key_field).(string)
		log.Printf("[DEBUG] adding field/value: '%s/%s' to template: '%s'", key_field, v, key)
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", key_field), v, 1)
	}

	//key := fmt.Sprintf(key_template, id_values...)

	return key
}

func helper_config_fields_from_schema(key_template string, s map[string]*schema.Schema) []string {
	var fields []string
	for k := range s {
		if !strings.Contains(key_template, fmt.Sprintf("{{%s}}", k)) && !s[k].Computed {
			log.Printf("[DEBUG] adding field: '%s' to config fields", k)
			fields = append(fields, k)
		} else {
			log.Printf("[DEBUG] NOT adding field: '%s' to config fields", k)
		}
	}
	return fields
}
