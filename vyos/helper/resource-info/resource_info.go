package resourceInfo

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type DELETETYPE string

const (
	// Faster if no children exists
	DeleteTypeResource DELETETYPE = "resource"

	// Will only delete what is defined in the resource schema, and will not touch children
	// This can be problematic in some cases, such as firewall rules
	DeleteTypeParameters DELETETYPE = "parameter"
)

type ResourceInfo struct {
	KeyTemplate             string
	CreateRequiredTemplates []string
	DeleteStrategy          DELETETYPE
	DeleteBlockerTemplates  []string
	StaticId                string
	ResourceSchema          *schema.Resource
}

func ValidateStringKeyField() schema.SchemaValidateFunc {
	return validation.StringDoesNotContainAny("_")
}

func ValidateDiagStringKeyField() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(ValidateStringKeyField())
}
