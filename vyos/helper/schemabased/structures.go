package schemabased

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DELETETYPE string

const (
	// Faster if no children exists
	DeleteTypeResource DELETETYPE = "resource"

	// Will only delete what is defined in the resource schema, and will not touch children
	// This can be problematic in some cases, such as firewall rules
	DeleteTypeParameters DELETETYPE = "parameter"
)

/*
	StaticId is used for global resources, where only one can exist at a time.
	Do not use KeyTemplate and StaticId at the same time.
*/
type ResourceInfo struct {
	KeyTemplate             string
	CreateRequiredTemplates []string
	DeleteStrategy          DELETETYPE
	DeleteBlockerTemplates  []string
	StaticId                string
	ResourceSchema          *schema.Resource
}

// These structs will contain an internal representation of configs,
// and unify the workflow when reading terraform and VyOS config
type configBlock struct {
	parent *configBlock
	key    *configKey

	resource_type schema.ValueType

	values   []configValue
	children map[*configKey]*configBlock
}

// Allow easy access from value to config block instead of using a plain string
type configValue struct {
	config_block *configBlock
	value_type   schema.ValueType
	value        string
}

func (value *configValue) getValueNative() interface{} {
	// Return golang native version of value

	var val interface{}

	switch value.value_type {
	case schema.TypeBool:
		if value.value == "true" {
			val = true
		} else if value.value == "false" {
			val = false
		} else {
			logger("ERROR", "value: %s, does not match bool 'true' or 'false'. Setting to 'false'.", value.value)
			val = false
		}
	case schema.TypeFloat:
		if f, err := strconv.ParseFloat(value.value, 32); err == nil {
			val = f
		}
		if f, err := strconv.ParseFloat(value.value, 64); err == nil {
			val = f
		}
	case schema.TypeInt:
		i, err := strconv.ParseInt(value.value, 10, 32)
		if err == nil {
			val = i
		} else {
			logger("ERROR", "value: %s, unable to convert to int.", value.value)
		}
	case schema.TypeString:
		val = value.value
	}

	return val
}

// Config keys
func (cfg *configBlock) GetFullConfigKey() (full_key string) {
	logger("TRACE", "{%s} Getting full key for: %s", cfg.key, full_key)

	if cfg.parent != nil {
		full_key = cfg.parent.GetFullConfigKey() + " "
	}

	return full_key + cfg.key.Key
}

// Config Values
func (cfg *configBlock) AddValue(value_type schema.ValueType, value string) {
	logger("TRACE", "{%s} Adding value: %s", cfg.key, value)

	switch value_type {
	case schema.TypeBool:
	case schema.TypeFloat:
	case schema.TypeInt:
	case schema.TypeString:
	default:
		logger("ERROR", "{%s} Value: %s has type: %d which is unknown and might case issues.", cfg.key, value, value_type)
	}

	config_v := configValue{
		config_block: cfg,
		value_type:   value_type,
		value:        value,
	}

	if cfg.values == nil {
		cfg.values = []configValue{config_v}
	} else {
		cfg.values = append(cfg.values, config_v)
	}
}

// Child configs
func (cfg *configBlock) CreateChild(key string, resource_type schema.ValueType) *configBlock {
	if cfg.children == nil {
		logger("TRACE", "{%s} Initializing child map", cfg.key)
		cfg.children = make(map[*configKey]*configBlock)
	}

	if child, ok := cfg.GetChild(key); ok {
		logger("TRACE", "{%s} Returning existing child: %s", cfg.key, key)
		return child
	}

	logger("TRACE", "{%s} Creating child: %s", cfg.key, key)

	key_obj := configKey{key}

	new_child := &configBlock{
		parent:        cfg,
		key:           &key_obj,
		resource_type: resource_type,
	}

	cfg.children[&key_obj] = new_child

	return new_child
}

func (cfg *configBlock) AddChild(child *configBlock) {
	if cfg.children == nil {
		logger("TRACE", "{%s} Initializing child map", cfg.key)
		cfg.children = make(map[*configKey]*configBlock)
	}

	logger("TRACE", "{%s} Adding child: %s", cfg.key, child.key)

	child.parent = cfg
	cfg.children[child.key] = child
}

func (cfg *configBlock) FindChild(full_key string) (child *configBlock, has_child bool) {
	logger("TRACE", "{%s} Looking for child: %s", cfg.key, full_key)

	if cfg.GetFullConfigKey() == full_key {
		return cfg, true
	}

	for _, current_child := range cfg.children {
		if sub_child, found := current_child.FindChild(full_key); found {
			return sub_child, true
		}
	}

	return nil, false
}

func (cfg *configBlock) GetChild(key string) (child *configBlock, has_child bool) {
	logger("TRACE", "{%s} Fetching child: %s", cfg.key, key)

	for child_key, child := range cfg.children {
		if child_key.Key == key {
			return child, true
		}
	}

	return nil, false
}

func (cfg *configBlock) GetChildren() (children map[*configKey]*configBlock, has_child bool) {
	logger("TRACE", "{%s} Fetching all children", cfg.key)

	if len(cfg.children) > 0 {
		logger("TRACE", "{%s} found '%d' children", cfg.key, len(cfg.children))
		return cfg.children, true
	}

	logger("TRACE", "{%s} found no children: '%#v'", cfg.key, cfg.children)

	return nil, false
}

func (cfg *configBlock) GetValues() (values []configValue, has_values bool) {
	logger("TRACE", "{%s} Fetching all values", cfg.key)

	if len(cfg.values) > 0 {
		logger("TRACE", "{%s} found '%d' values", cfg.key, len(cfg.values))
		return cfg.values, true
	}

	logger("TRACE", "{%s} found no values: '%#v'", cfg.key, cfg.values)
	return nil, false
}

func (cfg *configBlock) GetValuesRecursive() (values []configValue, has_values bool) {
	logger("TRACE", "{%s} Fetching all (sub)values", cfg.key)

	var ret []configValue

	ret = append(ret, cfg.values...)

	if children, ok := cfg.GetChildren(); ok {
		for _, child := range children {
			if vals, ok := child.GetValuesRecursive(); ok {
				ret = append(ret, vals...)
			}
		}
	}
	if len(ret) > 0 {
		return ret, true
	}

	return nil, false
}

func (cfg *configBlock) PopChild(key string) (child *configBlock, had_child bool) {
	if found_child, ok := cfg.GetChild(key); ok {
		delete(cfg.children, found_child.key)
		return found_child, true
	}

	return nil, false
}

func (cfg *configBlock) convertTreeToNative() map[string]interface{} {
	logger("TRACE", "{%s} Converting to a tree of golang native types", cfg.key)

	var return_values []interface{}

	// Handle own values
	if self_values, ok := cfg.GetValues(); ok {
		for idx, value := range self_values {
			logger("TRACE", "idx: %d, value: %s", idx, value.value)
			return_values = append(return_values, value.getValueNative())
		}
	}

	// Recurse for children / sub configs
	if children, ok := cfg.GetChildren(); ok {
		for key, child := range children {
			logger("TRACE", "child: %s", key)
			child_values := child.convertTreeToNative()
			return_values = append(return_values, child_values)
		}
	}

	// Set the config key as the map key
	data := map[string]interface{}{
		cfg.key.Key: return_values,
	}

	logger("TRACE", "{%s} returning result '%#v'", cfg.key, data)

	return data
}

func (cfg *configBlock) MarshalJSON() ([]byte, error) {
	// Make object JSON marshalable

	logger("TRACE", "{%s} MarshalJSON", cfg.key)

	j, err := json.Marshal(cfg.convertTreeToNative())
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (cfg *configBlock) convertTreeToTerraform() interface{} {
	logger("TRACE", "{%s} Converting to a tree of golang native types in tf structure", cfg.key)
	logger("TRACE", "{%s} of type schema.Type '%d'", cfg.key, cfg.resource_type)

	switch cfg.resource_type {
	case schema.TypeBool, schema.TypeFloat, schema.TypeInt, schema.TypeString:
		// Basic values
		if len(cfg.values) == 0 {
			logger("ERROR", "{%s} of type schema.Type '%d', has '%d' values, expected 1.", cfg.key, cfg.resource_type, len(cfg.values))
		} else {
			if len(cfg.values) > 1 {
				logger("ERROR", "{%s} of type schema.Type '%d', has '%d' values, expected 1, only including first value.", cfg.key, cfg.resource_type, len(cfg.values))
			} else {
				value := cfg.values[0].getValueNative()
				logger("TRACE", "{%s} with value: '%v'", cfg.key, value)
				return value
			}
		}
	case schema.TypeList:
		response := []interface{}{}

		// Check for values
		if values, ok := cfg.GetValues(); ok {
			logger("TRACE", "{%s} of type schema.Type '%d', found '%d' values", cfg.key, cfg.resource_type, len(values))
			for _, value := range values {
				response = append(response, value.getValueNative())
			}
		}

		// Recurse
		if children, ok := cfg.GetChildren(); ok {
			logger("TRACE", "{%s} of type schema.Type '%d', found '%d' children", cfg.key, cfg.resource_type, len(children))
			block := make(map[string]interface{})
			for key, child := range children {
				result := child.convertTreeToTerraform()
				block[key.Key] = result
			}
			response = []interface{}{block}
		}

		return response

	case schema.TypeSet:
		response := []interface{}{}

		// Check for values
		if values, ok := cfg.GetValues(); ok {
			logger("TRACE", "{%s} of type schema.Type '%d', found '%d' values", cfg.key, cfg.resource_type, len(values))
			for _, value := range values {
				response = append(response, value.getValueNative())
			}
		}

		// Recurse
		if children, ok := cfg.GetChildren(); ok {
			logger("TRACE", "{%s} of type schema.Type '%d', found '%d' children", cfg.key, cfg.resource_type, len(children))
			block := make(map[string]interface{})
			for key, child := range children {
				result := child.convertTreeToTerraform()
				block[key.Key] = result
			}
			response = []interface{}{block}
		}

		return response

	case schema.TypeMap:
		if values, ok := cfg.GetValues(); ok {
			logger("TRACE", "{%s} of type schema.Type '%d', found '%d' values", cfg.key, cfg.resource_type, len(values))
			logger("ERROR", "{%s} of type schema.Type '%d', has '%d' values, expected zero. Values: %#v", cfg.key, cfg.resource_type, len(values), values)
		}

		// Recurse
		response := map[string]interface{}{}
		if children, ok := cfg.GetChildren(); ok {
			logger("TRACE", "{%s} of type schema.Type '%d', found '%d' children", cfg.key, cfg.resource_type, len(children))
			for key, child := range children {
				result := child.convertTreeToTerraform()
				response[key.Key] = result
			}
		}
		return response

	}

	logger("ERROR", "{%s} Unable to return anything, this should never happen!", cfg.key)
	panic("Unable to return anything, this should never happen!")
}

func (cfg *configBlock) MarshalTerraform() map[string]interface{} {
	// Return an object that can be used with schema.ResourceData.set() function

	logger("TRACE", "{%s} MarshalTerraform", cfg.key)

	response := make(map[string]interface{})

	if children, ok := cfg.GetChildren(); ok {
		for key, child := range children {
			response[key.Key] = child.convertTreeToTerraform()
		}
	}

	return response
}

func (cfg *configBlock) convertTreeToVyos() map[string]interface{} {
	// Needs to remove config parameters that are boolean == false

	logger("TRACE", "{%s} Converting to a tree of vyos client compatible style golang types", cfg.key)

	var return_values []interface{}

	// Handle own values (skip bool = false)
	if self_values, ok := cfg.GetValues(); ok {
		for idx, value := range self_values {
			logger("TRACE", "idx: %d, type: %d, value: %s", idx, value.value_type, value.value)
			var val interface{}
			switch value.value_type {
			case schema.TypeBool:
				if value.value == "true" {
					val = true
				} else if value.value == "false" {
					continue
				} else {
					logger("ERROR", "idx: %d, value: %s, does not match bool 'true' or 'false'. Setting to 'false'.", idx, value.value)
					continue
				}

			default:
				val = value.value
			}

			return_values = append(return_values, val)
		}
	}

	// Recurse for children (skip nil as that would be bool = false)
	if children, ok := cfg.GetChildren(); ok {
		for key, child := range children {
			logger("TRACE", "child: %s", key)

			if child.resource_type == schema.TypeBool {
				if child.values[0].value == "true" {
					return_values = append(return_values, strings.Replace(key.Key, "_", "-", -1))
				}
			} else {
				child_values := child.convertTreeToVyos()
				return_values = append(return_values, child_values)
			}
		}
	}

	// Do not return a map if return valies = nil since that means we were a bool = false
	if return_values != nil {
		data := map[string]interface{}{
			strings.Replace(cfg.key.Key, "_", "-", -1): return_values,
		}
		logger("TRACE", "{%s} returning result '%#v'", cfg.key, data)
		return data
	} else {
		logger("TRACE", "{%s} returning nil", cfg.key)
		return nil
	}
}

func (cfg *configBlock) MarshalVyos() (key string, config any) {
	// Return object that can be used with vyos client to create / set configs

	logger("TRACE", "{%s} MarshalVyos", cfg.key)

	vy := cfg.convertTreeToVyos()

	// should only ever contain a single key = config
	for key, config := range vy {
		return key, config
	}

	return "", nil
}

func (cfg *configBlock) GetDifference(compare_config *configBlock) (changed *configBlock, missing *configBlock) {
	// Return config that represents changed (+ new) and missing parameters of this config when compared to compare_config
	// Best used as changed, deleted := cfg_from_terraform.GetDifference(cfg_from_vyos)

	logger("TRACE", "{%s}", cfg.key)

	// Keep track if we have added anything with these horrible extra booleans
	has_changed := false
	changed = &configBlock{
		key: cfg.key,
	}

	has_missing := false
	missing = &configBlock{
		key: cfg.key,
	}

	// Handle lists of values
	if len(compare_config.values) > 1 || len(cfg.values) > 1 {
		logger("TRACE", "{%s} find missing values", cfg.key)
		matched_c_idxes := []int{}
		matched_s_idxes := []int{}

		// Find missing values
		for cidx, cval := range compare_config.values {
			logger("TRACE", "{%s} cidx %i, cval %s", cfg.key, cidx, cval.value)
			for sidx, sval := range cfg.values {
				logger("TRACE", "{%s} sidx %i, sval %s", cfg.key, sidx, sval.value)
				previously_matched := false
				for _, msidx := range matched_s_idxes {
					if sidx == msidx {
						logger("TRACE", "{%s} Already matched sidx %i", cfg.key, sidx)
						previously_matched = true
						break
					}
				}
				if !previously_matched && cval.value == sval.value {
					logger("TRACE", "{%s} New match cidx %i sidx %i", cfg.key, cidx, sidx)
					matched_c_idxes = append(matched_c_idxes, cidx)
					matched_s_idxes = append(matched_s_idxes, sidx)
				}
			}
		}
		for cidx, cval := range compare_config.values {
			matched := false
			for _, mcidx := range matched_c_idxes {
				if cidx == mcidx {
					logger("TRACE", "{%s} Match cidx %i", cfg.key, cidx)
					matched = true
					break
				}
			}
			if !matched {
				logger("TRACE", "{%s} Missing val '%s'", cfg.key, cval.value)
				missing.AddValue(cval.value_type, cval.value)
				has_missing = true
			}
		}

		// Find new values
		logger("TRACE", "{%s} find new values", cfg.key)
		matched_c_idxes = []int{}
		matched_s_idxes = []int{}
		for sidx, sval := range cfg.values {
			logger("TRACE", "{%s} sidx %i, sval %s", cfg.key, sidx, sval.value)
			for cidx, cval := range compare_config.values {
				logger("TRACE", "{%s} cidx %i, cval %s", cfg.key, cidx, cval.value)
				previously_matched := false
				for _, mcidx := range matched_c_idxes {
					if cidx == mcidx {
						logger("TRACE", "{%s} Already matched cidx %i", cfg.key, cidx)
						previously_matched = true
						break
					}
				}
				if !previously_matched && cval.value == sval.value {
					logger("TRACE", "{%s} New match cidx %i sidx %i", cfg.key, cidx, sidx)
					matched_c_idxes = append(matched_c_idxes, cidx)
					matched_s_idxes = append(matched_s_idxes, sidx)
				}
			}
		}
		for sidx, sval := range cfg.values {
			matched := false
			for _, msidx := range matched_s_idxes {
				if sidx == msidx {
					logger("TRACE", "{%s} Match sidx %i", cfg.key, sidx)
					matched = true
					break
				}
			}
			if !matched {
				logger("TRACE", "{%s} New val '%s'", cfg.key, sval.value)
				changed.AddValue(sval.value_type, sval.value)
				has_changed = true
			}
		}

	} else {

		// Find missing and changed values
		logger("TRACE", "{%s} find missing values", cfg.key)
		if compare_vals, ok := compare_config.GetValues(); ok {

			// Keep track of indexes we already matched against.
			matched_idxes := []int{}
			for _, compare_val := range compare_vals {
				found_val := false

				// Check each value
				for idx, self_val := range cfg.values {
					already_matched := false
					for _, matched_idx := range matched_idxes {
						if idx == matched_idx {
							already_matched = true
							break
						}
					}
					if already_matched {
						logger("TRACE", "{%s} with value '%s' already matched.", cfg.key, self_val.value)
						continue
					}
					if (compare_val.value_type == self_val.value_type) && (cfg.resource_type >= schema.TypeBool || cfg.resource_type == schema.TypeFloat || cfg.resource_type == schema.TypeInt || cfg.resource_type == schema.TypeString) {
						matched_idxes = append(matched_idxes, idx)
						if compare_val.value == self_val.value {
							logger("TRACE", "{%s} Is of native type, and both has equal value: '%s', skipping.", cfg.key, self_val.value)
						} else {
							logger("TRACE", "{%s} Is of native type, and both has a value, assuming change '%s' => '%s'", cfg.key, compare_val.value, self_val.value)
							found_val = true
							changed.AddValue(self_val.value_type, self_val.value)
							has_changed = true
							break
						}
					}

					if (compare_val.value_type == self_val.value_type) && (compare_val.value == self_val.value) {
						matched_idxes = append(matched_idxes, idx)
						logger("TRACE", "{%s} Both has value '%s'", cfg.key, self_val.value)
						found_val = true
						break
					}
				}

				if !found_val {
					logger("TRACE", "{%s} Missing val '%s'", cfg.key, compare_val.value)
					missing.AddValue(compare_val.value_type, compare_val.value)
					has_missing = true
				}
			}
		}

		// Find new values
		logger("TRACE", "{%s} find new values", cfg.key)
		if self_vals, ok := cfg.GetValues(); ok {
			for _, self_val := range self_vals {
				found_val := false

				for _, compare_val := range compare_config.values {
					if (compare_val.value_type == self_val.value_type) && (cfg.resource_type == schema.TypeBool || cfg.resource_type == schema.TypeFloat || cfg.resource_type == schema.TypeInt || cfg.resource_type == schema.TypeString) {
						logger("TRACE", "{%s} Is of native type, and both has a value, assuming '%s' => '%s', should have been handled together with missing values, skipping.", cfg.key, compare_val.value, self_val.value)
						found_val = true
						break
					}

					if (self_val.value_type == compare_val.value_type) && (self_val.value == compare_val.value) {
						logger("TRACE", "{%s} Both has value '%s'", cfg.key, self_val.value)
						found_val = true
						break
					}
				}

				if !found_val {
					logger("TRACE", "{%s} New val '%s'", cfg.key, self_val.value)
					changed.AddValue(self_val.value_type, self_val.value)
					has_changed = true
				}
			}
		}
	}

	// Help debugging by printing childrens keys
	self_keys := make([]string, len(cfg.children))
	for k := range cfg.children {
		self_keys = append(self_keys, k.Key)
	}
	logger("DEBUG", "{%s} self children: %v", cfg.key, self_keys)

	compare_keys := make([]string, len(compare_config.children))
	for k := range compare_config.children {
		compare_keys = append(compare_keys, k.Key)
	}
	logger("DEBUG", "{%s} compare children: %v", cfg.key, compare_keys)

	// Find missing children
	logger("TRACE", "{%s} find missing children", cfg.key)
	if compare_children, ok := compare_config.GetChildren(); ok {
		for compare_child_key, compare_child_val := range compare_children {

			found_child := false

			if self_child, ok := cfg.GetChild(compare_child_key.Key); ok {
				logger("TRACE", "{%s} Both has child '%s'", cfg.key, self_child.key)
				found_child = true
				continue
			}

			if !found_child {
				logger("TRACE", "{%s} Missing child '%s'", cfg.key, compare_child_key)
				missing.AddChild(compare_child_val)
				has_missing = true
			}
		}
	}

	// Find new children
	logger("TRACE", "{%s} find new children", cfg.key)
	if self_children, ok := cfg.GetChildren(); ok {
		for self_child_key, self_child_val := range self_children {

			found_child := false

			if compare_child, ok := compare_config.GetChild(self_child_key.Key); ok {
				logger("TRACE", "{%s} Both has child '%s'", cfg.key, compare_child.key)
				found_child = true
				continue
			}

			if !found_child {
				logger("TRACE", "{%s} New config child found '%s'", cfg.key, self_child_key)
				changed.AddChild(self_child_val)
				has_changed = true
			}
		}
	}

	// Recurse into own children, compare children should either be in this list, or in the missing list
	logger("TRACE", "{%s} recurse operation into children", cfg.key)
	if self_children, ok := cfg.GetChildren(); ok {
		for self_child_key, self_child_val := range self_children {
			if _, is_missing := missing.GetChild(self_child_key.Key); is_missing {
				logger("TRACE", "{%s} Child already marked as missing '%s'", cfg.key, self_child_key.Key)
				continue
			}

			if compare_child, ok := compare_config.GetChild(self_child_key.Key); ok {

				child_changed, child_missing := self_child_val.GetDifference(compare_child)

				if child_changed != nil {
					changed.AddChild(child_changed)
					has_changed = true
				}

				if child_missing != nil {
					missing.AddChild(child_missing)
					has_missing = true
				}
			}
		}
	}

	if !has_changed {
		logger("TRACE", "{%s} No changes found", cfg.key)
		changed = nil
	}
	if !has_missing {
		logger("TRACE", "{%s} No missing found", cfg.key)
		missing = nil
	}

	logger("TRACE", "{%s} Changed: %v", cfg.key, changed)
	logger("TRACE", "{%s} Missing: %v", cfg.key, missing)

	return changed, missing
}

func (cfg *configBlock) GlobalResourceRemoveSuperfluous(resource_info *ResourceInfo) error {
	/*
		Readies global config by removing extra children.
		Returns error if there are any values at the top level
	*/

	// There should not be any top level values on global resources
	if _, has_values := cfg.GetValues(); has_values {
		return fmt.Errorf("Global resources with top level values are currently not supported")
	}

	// Remove parameters and children not defined in schema
	children, _ := cfg.GetChildren()
	for child_key := range children {
		if _, ok := resource_info.ResourceSchema.Schema[child_key.Key]; ok == false {
			cfg.PopChild(child_key.Key)
		}
	}

	return nil
}
