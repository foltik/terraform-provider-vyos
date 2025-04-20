package vyos

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/foltik/vyos-client-go/client"
)

func resourceConfigBlockTree() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource is useful when a single command is not enough for a valid config commit and children paths are needed.",
		CreateContext: resourceConfigBlockTreeCreate,
		ReadContext:   resourceConfigBlockTreeRead,
		UpdateContext: resourceConfigBlockTreeUpdate,
		DeleteContext: resourceConfigBlockTreeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The resource ID, same as the `path`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"path": {
				Description:      "Config path seperated by spaces.",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				ForceNew:         true,
			},
			"configs": {
				Description: "Key/Value map of config parameters. Value can be a jsonencode list",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:         true,
				DiffSuppressFunc: configDiffSuppressFunc,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(10 * time.Minute),
			Read:    schema.DefaultTimeout(10 * time.Minute),
			Update:  schema.DefaultTimeout(10 * time.Minute),
			Delete:  schema.DefaultTimeout(10 * time.Minute),
			Default: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func configDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {

	multivalueOld := []string{}
	err := json.Unmarshal([]byte(old), &multivalueOld)
	if err != nil {
		return false
	}
	sort.Strings(multivalueOld)

	multivalueNew := []string{}
	err = json.Unmarshal([]byte(new), &multivalueNew)
	if err != nil {
		return false
	}
	sort.Strings(multivalueNew)

	return reflect.DeepEqual(multivalueOld, multivalueNew)
}

// Covert configs to a set of vyos client commands.
// If expand_slice is set, then list values (json encoded) are expanded in multiple vyos client commands
// If expand_slice is not set then the values in the map might contain slices
func getCommandsForConfig(path string, config interface{}, expand_slice bool) (commands map[string]interface{}) {

	commands = map[string]interface{}{}
	for key, value := range config.(map[string]interface{}) {
		// prefix with path - Note that key might be empty
		if len(key) > 0 {
			key = path + " " + key
		} else {
			key = path
		}

		// Try to decode the string as json list
		value := value.(string)
		multivalue := []string{}
		err := json.Unmarshal([]byte(value), &multivalue)
		if err == nil {
			if expand_slice {
				for _, subvalue := range multivalue {
					commands[key+" "+subvalue] = ""
				}
			} else {
				commands[key] = multivalue
			}
		} else {
			// Could not decode json string - assume single value string
			if expand_slice {
				commands[key] = value
			} else {
				commands[key] = []string{value}
			}
		}
	}
	return
}

func resourceConfigBlockTreeCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	client := *p.client
	path := d.Get("path").(string)

	// Get commands needed to create resource in Vyos
	commands := getCommandsForConfig(path, d.Get("configs"), true)

	err := client.Config.SetTree(ctx, commands)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(path)
	p.conditionalSave(ctx)
	return diags
}

func resourceConfigBlockTreeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client
	path := d.Id()

	configsTree, err := c.Config.ShowAny(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	flat, err := client.Flatten(configsTree)
	if err != nil {
		return diag.FromErr(err)
	}

	// Convert Vyos commands to Terraform schema
	configs := map[string]interface{}{}
	for _, config := range flat {
		key := config[0]
		value := config[1]
		existing_value, ok := configs[key]
		if ok {
			// This is command with multiple values
			switch existing_value := existing_value.(type) {
			case string:
				// Second value for command found - convert to slice
				configs[key] = []string{existing_value, value}
			case []string:
				// N value for command found - append to slice
				configs[key] = append(existing_value, value)
			}
		} else {
			configs[key] = value
		}
	}

	// If there are slices then covert them to json strings
	for key, value := range configs {
		switch value := value.(type) {
		case []string:
			jsonBytes, _ := json.Marshal(value)
			configs[key] = string(jsonBytes)
		}
	}

	// Easiest way to allow ImportStatePassthroughContext to work is to set the path
	if d.Get("path") == "" {
		if err := d.Set("path", path); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("configs", configs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceConfigBlockTreeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client

	path := d.Get("path").(string)
	o, n := d.GetChange("configs")
	old_configs := o.(map[string]interface{})
	new_configs := n.(map[string]interface{})

	// Get commands needed to create old a new config
	old_comands := getCommandsForConfig(path, old_configs, false)
	new_comands := getCommandsForConfig(path, new_configs, false)

	// NOTE: it is important to apply new settings before deleting to
	//       avoid errors. This is because delete and set are 2
	//       different API calls and this might result in invalid
	//       intermediary configs if we delete first.

	// Calculate new commands (new config minus old config)
	set_commands := map[string]interface{}{}
	for command, new_value := range new_comands {
		old_value, ok := old_comands[command]
		if !ok || !reflect.DeepEqual(new_value, old_value) {
			switch new_value := new_value.(type) {
			case []string:
				// List value - multiple subcommands
				for _, value := range new_value {
					set_commands[command+" "+value] = ""
				}
			case string:
				set_commands[command] = new_value
			}
		}
	}
	if len(new_comands) > 0 {
		errSet := c.Config.SetTree(ctx, new_comands)
		if errSet != nil {
			return diag.FromErr(errSet)
		}
	}

	// Calculate delete commands (old config minus new config)
	delete_commands := map[string]interface{}{}
	for command, old_value := range old_comands {
		new_value, ok := new_comands[command]
		if !ok {
			// Not found in new config - delete commpletly
			if len(command) > len(path) {
				// Do not delete path
				delete_commands[command] = ""
			}
		} else {
			// Compare old and new values for this command
			for _, old_value_part := range old_value.([]string) {
				found := false
				for _, new_value_part := range new_value.([]string) {
					if old_value_part == new_value_part {
						found = true
						break
					}
				}
				// Only delete if old value not in new config AND this
				// command is bellow the resource path
				if !found && len(command) > len(path) {
					delete_commands[command] = old_value_part
				}
			}
		}
	}
	// Remove orphan nodes as well
	// An orphan is a node that does not have any entries in the new config
	//
	// Example: "system static-host-mapping host-name foo inet" => 1.2.3.4"
	//          This will fail if we delete only "system static-host-mapping host-name foo inet"
	//          and do not delete "system static-host-mapping host-name foo" as well
	for key, _ := range delete_commands {
		key_parts := strings.Split(key, " ")
		parent := ""
	out:
		for _, key_part := range key_parts {
			if len(parent) > 0 {
				parent += " "
			}
			parent += key_part
			found := false
			for key, _ := range new_comands {
				if strings.Contains(key, parent) {
					// parent still exists in new config - keep parent
					found = true
					break
				}
			}
			if !found && len(parent) > len(path) {
				// Delete parent if not done already
				if _, ok := delete_commands[parent]; !ok {
					delete_commands[parent] = ""
				}
				break out
			}
		}
	}
	if len(delete_commands) > 0 {
		errDel := c.Config.DeleteTree(ctx, delete_commands)
		if errDel != nil {
			return diag.FromErr(errDel)
		}
	}

	p.conditionalSave(ctx)
	return diags
}

func resourceConfigBlockTreeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	p := m.(*ProviderClass)
	c := *p.client
	path := d.Get("path").(string)

	err := c.Config.Delete(ctx, path)
	if err != nil {
		return diag.FromErr(err)
	}

	p.conditionalSave(ctx)
	return diags
}
