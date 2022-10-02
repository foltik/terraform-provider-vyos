package schemabased

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ValidateStringKeyField() schema.SchemaValidateFunc {
	return validation.StringDoesNotContainAny("_ =|")
}

func ValidateDiagStringKeyField() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(ValidateStringKeyField())
}
