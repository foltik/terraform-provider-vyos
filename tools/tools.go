//go:build tools
// +build tools

// Ref: https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md

package tools

import (
	// document generation
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
