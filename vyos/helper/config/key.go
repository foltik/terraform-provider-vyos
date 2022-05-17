package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ConfigKeyTemplate struct {
	Template string
}

// Used to clearify child index in ConfigBlock
// Should never contain spaces
// TODO can we verify "no spaces" in some way? maybe a private property with a setter or something?
type ConfigKey struct {
	Key string
}

func FormatResourceId(template ConfigKeyTemplate, d *schema.ResourceData) string {
	// Format terraform resource ID compliant string

	logger.Log("TRACE", "template: '%s'", template)

	// Keyfilds used to generate the ID, format: "field|field|field"....
	// ID format: "field=value|field=value"...
	// how ever many is needed for the resource  to be uniqly identified

	var id string

	for _, attr := range getKeyFields(template) {
		// Build terraform resource ID from template fields one by one

		logger.Log("TRACE", "adding attr: '%s'", attr)

		val := fmt.Sprintf("%v", d.Get(attr))

		id = fmt.Sprintf("%s|%s=%s", id, attr, val)
		logger.Log("TRACE", "current id: '%s'", id)
	}
	id = strings.TrimLeft(id, "|")

	logger.Log("TRACE", "complete id: '%s'", id)

	return id
}

func FormatKey(template ConfigKeyTemplate, id string, d *schema.ResourceData) string {
	// Format VyOS compliant key string

	logger.Log("TRACE", "template: '%s', id: '%s'", template, id)

	key := template.Template
	for _, parameter := range getKeyFields(template) {
		// Loop over each templated parameter field

		// Get parameter value for current templated field
		value := fmt.Sprintf("%v", d.Get(parameter))

		logger.Log("TRACE", "replacing templated 'parameter = value': '%s = %s'.", parameter, value)

		// Replace templated key parameter with value one by one
		key = strings.Replace(key, fmt.Sprintf("{{%s}}", parameter), value, 1)

		logger.Log("TRACE", "Current key: '%s'", key)
	}

	logger.Log("TRACE", "Complete key: '%s'", key)

	return key
}

func getKeyFields(template ConfigKeyTemplate) []string {
	// Use key template to generate a list of resource ID parameters

	logger.Log("TRACE", "template: '%s'", template)

	re := regexp.MustCompile(`\{{[a-z_]+}}`)
	fields := re.FindAllString(template.Template, -1)

	for idx, field := range fields {
		logger.Log("TRACE", "idx: '%d', field: '%s'", idx, field)

		fields[idx] = strings.Trim(field, `{}`)
	}
	return fields
}
