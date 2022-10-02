package schemabased

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ConfigKeyTemplate struct {
	Template string
}

// Used to clearify child index in ConfigBlock
// Should never contain spaces
type ConfigKey struct {
	Key string
}

func FormatResourceId(template ConfigKeyTemplate, d *schema.ResourceData) string {
	// Format terraform resource ID compliant string

	Log("TRACE", "template: '%s'", template)

	// Keyfilds used to generate the ID, format: "field|field|field"....
	// ID format: "field=value|field=value"...
	// how ever many is needed for the resource  to be uniqly identified

	var id string

	for _, attr := range GetKeyFieldsFromTemplate(template) {
		// Build terraform resource ID from template fields one by one

		Log("TRACE", "adding attr: '%s'", attr)

		val := fmt.Sprintf("%v", d.Get(attr))

		id = fmt.Sprintf("%s|%s=%s", id, attr, val)
		Log("TRACE", "current id: '%s'", id)
	}
	id = strings.TrimLeft(id, "|")

	Log("TRACE", "complete id: '%s'", id)

	return id
}

func FormatKeyFromResource(template ConfigKeyTemplate, d *schema.ResourceData) string {
	// Format VyOS compliant key string from terraform resource data

	Log("TRACE", "template: '%s'", template)

	key := template.Template
	for _, parameter := range GetKeyFieldsFromTemplate(template) {
		// Loop over each templated parameter field

		// Get parameter value for current templated field
		value := fmt.Sprintf("%v", d.Get(parameter))

		Log("TRACE", "replacing templated 'parameter = value': '%s = %s'.", parameter, value)

		// Replace templated key parameter with value one by one
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", parameter), value, 1)

		Log("TRACE", "Current key: '%s'", key)
	}

	Log("TRACE", "Complete key: '%s'", key)

	return key
}

func FormatKeyFromId(template ConfigKeyTemplate, id string) string {
	// Format VyOS compliant key string from reource ID string

	Log("TRACE", "id: '%s'", id)

	key := template.Template
	for field, value := range GetFieldValuePairsFromId(id) {
		// Loop over each templated parameter field

		Log("TRACE", "replacing 'parameter = value': '%s = %s'.", field, value)

		// Replace templated key parameter with value one by one
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", field), value, 1)

		Log("TRACE", "Current key: '%s'", key)
	}

	Log("TRACE", "Complete key: '%s'", key)

	return key
}

func GetKeyFieldsFromTemplate(template ConfigKeyTemplate) []string {
	// Use key template to generate a list of resource ID parameters

	Log("TRACE", "template: '%s'", template)

	re := regexp.MustCompile(`\{{[a-z_]+}}`)
	fields := re.FindAllString(template.Template, -1)

	for idx, field := range fields {
		Log("TRACE", "idx: '%d', field: '%s'", idx, field)

		fields[idx] = strings.Trim(field, `{}`)
	}
	return fields
}

func GetFieldValuePairsFromId(id string) map[string]string {
	// Split resource ID into key value pairs.
	// Required ID format: parameter=value|parameter2=value2|parameter3=value3

	Log("TRACE", "ID: '%s'", id)

	field_value_pairs := make(map[string]string)

	for _, pair_str := range strings.Split(id, "|") {
		pair_slice := strings.Split(pair_str, "=")
		field := pair_slice[0]
		value := pair_slice[1]
		Log("TRACE", "Field: '%s', Value: '%s'", field, value)
		field_value_pairs[field] = value
	}
	return field_value_pairs
}