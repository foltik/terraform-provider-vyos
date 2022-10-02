package schemabased

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type configKeyTemplate struct {
	Template string
}

// Used to clearify child index in ConfigBlock
// Should never contain spaces
type configKey struct {
	Key string
}

func formatResourceId(template configKeyTemplate, d *schema.ResourceData) string {
	// Format terraform resource ID compliant string

	logger("TRACE", "template: '%s'", template)

	// Keyfilds used to generate the ID, format: "field|field|field"....
	// ID format: "field=value|field=value"...
	// how ever many is needed for the resource  to be uniqly identified

	var id string

	for _, attr := range getKeyFieldsFromTemplate(template) {
		// Build terraform resource ID from template fields one by one

		logger("TRACE", "adding attr: '%s'", attr)

		val := fmt.Sprintf("%v", d.Get(attr))

		id = fmt.Sprintf("%s|%s=%s", id, attr, val)
		logger("TRACE", "current id: '%s'", id)
	}
	id = strings.TrimLeft(id, "|")

	logger("TRACE", "complete id: '%s'", id)

	return id
}

func formatKeyFromResource(template configKeyTemplate, d *schema.ResourceData) string {
	// Format VyOS compliant key string from terraform resource data

	logger("TRACE", "template: '%s'", template)

	key := template.Template
	for _, parameter := range getKeyFieldsFromTemplate(template) {
		// Loop over each templated parameter field

		// Get parameter value for current templated field
		value := fmt.Sprintf("%v", d.Get(parameter))

		logger("TRACE", "replacing templated 'parameter = value': '%s = %s'.", parameter, value)

		// Replace templated key parameter with value one by one
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", parameter), value, 1)

		logger("TRACE", "Current key: '%s'", key)
	}

	logger("TRACE", "Complete key: '%s'", key)

	return key
}

func formatKeyFromId(template configKeyTemplate, id string) string {
	// Format VyOS compliant key string from reource ID string

	logger("TRACE", "id: '%s'", id)

	key := template.Template
	for field, value := range getFieldValuePairsFromId(id) {
		// Loop over each templated parameter field

		logger("TRACE", "replacing 'parameter = value': '%s = %s'.", field, value)

		// Replace templated key parameter with value one by one
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", field), value, 1)

		logger("TRACE", "Current key: '%s'", key)
	}

	logger("TRACE", "Complete key: '%s'", key)

	return key
}

func getKeyFieldsFromTemplate(template configKeyTemplate) []string {
	// Use key template to generate a list of resource ID parameters

	logger("TRACE", "template: '%s'", template)

	re := regexp.MustCompile(`\{{[a-z_]+}}`)
	fields := re.FindAllString(template.Template, -1)

	for idx, field := range fields {
		logger("TRACE", "idx: '%d', field: '%s'", idx, field)

		fields[idx] = strings.Trim(field, `{}`)
	}
	return fields
}

func getFieldValuePairsFromId(id string) map[string]string {
	// Split resource ID into key value pairs.
	// Required ID format: parameter=value|parameter2=value2|parameter3=value3

	logger("TRACE", "ID: '%s'", id)

	field_value_pairs := make(map[string]string)

	for _, pair_str := range strings.Split(id, "|") {
		pair_slice := strings.Split(pair_str, "=")
		field := pair_slice[0]
		value := pair_slice[1]
		logger("TRACE", "Field: '%s', Value: '%s'", field, value)
		field_value_pairs[field] = value
	}
	return field_value_pairs
}

func keyAndTemplate(d *schema.ResourceData, resourceInfo *ResourceInfo) (configKey, configKeyTemplate) {
	/*
		Useful for read, update and delete functions.
		Create function does not have an ID to rely on and can currently not use this to get the key and template
	*/
	key_template := configKeyTemplate{Template: resourceInfo.KeyTemplate}
	key_string := formatKeyFromId(key_template, d.Id())
	key := configKey{Key: key_string}

	return key, key_template
}
