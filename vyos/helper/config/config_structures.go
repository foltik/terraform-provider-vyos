package config

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Implement some interfaces to help sort it all
type ConfigParentInterface interface {
	AddChild(key ConfigKey, values []string) (child *ConfigBlock)
	GetChild(key ConfigKey) (child *ConfigBlock, has_child bool)
	FindChild(full_key string) (child *ConfigBlock, has_child bool)
	GetChildren() (children map[ConfigKey]*ConfigBlock, has_child bool)
}

type ConfigChildInterface interface {
	GetParent() (parent *ConfigBlock, has_parent bool)
}

type ConfigBlockInterface interface {
	GetFullConfigKey() (full_key string)
	GetValues() (values []string, has_values bool)
}

// These structs will contain an internal representation of configs,
// and unify the workflow when reading terraform and VyOS config
type ConfigBlock struct {
	parent *ConfigBlock
	key    *ConfigKey

	resource_type schema.ValueType

	values   []ConfigValue
	children map[*ConfigKey]*ConfigBlock
}

// Allow easy access from value to config block instead of using a plain string
type ConfigValue struct {
	config_block *ConfigBlock
	value_type   schema.ValueType
	value        string
}

func (value *ConfigValue) getValueNative() interface{} {
	// Return golang native version of value

	var val interface{}

	switch value.value_type {
	case schema.TypeBool:
		if value.value == "true" {
			val = true
		} else if value.value == "false" {
			val = false
		} else {
			logger.Log("ERROR", "value: %s, does not match bool 'true' or 'false'. Setting to 'false'.", value.value)
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
			logger.Log("ERROR", "value: %s, unable to convert to int.", value.value)
		}
	case schema.TypeString:
		val = value.value
	}

	return val
}

// Config keys
func (cfg *ConfigBlock) GetFullConfigKey() (full_key string) {
	logger.Log("TRACE", "{%s} Getting full key for: %s", cfg.key, full_key)

	if cfg.parent != nil {
		full_key = cfg.parent.GetFullConfigKey() + " "
	}

	return full_key + cfg.key.Key
}

// Config Values
func (cfg *ConfigBlock) AddValue(value_type schema.ValueType, value string) {
	logger.Log("TRACE", "{%s} Adding value: %s", cfg.key, value)

	switch value_type {
	case schema.TypeBool:
	case schema.TypeFloat:
	case schema.TypeInt:
	case schema.TypeString:
	default:
		logger.Log("ERROR", "{%s} Value: %s has type: %d which is unknown and might case issues.", cfg.key, value, value_type)
	}

	config_v := ConfigValue{
		config_block: cfg,
		value_type:   value_type,
		value:        value,
	}

	if cfg.values == nil {
		cfg.values = []ConfigValue{config_v}
	} else {
		cfg.values = append(cfg.values, config_v)
	}
}

// Child configs
func (cfg *ConfigBlock) CreateChild(key *ConfigKey, resource_type schema.ValueType) *ConfigBlock {
	if cfg.children == nil {
		logger.Log("TRACE", "{%s} Initializing child map", cfg.key)
		cfg.children = make(map[*ConfigKey]*ConfigBlock)
	}

	logger.Log("TRACE", "{%s} Creating child: %s", cfg.key, key)

	new_child := &ConfigBlock{
		parent:        cfg,
		key:           key,
		resource_type: resource_type,
	}

	cfg.children[key] = new_child

	return new_child
}

func (cfg *ConfigBlock) AddChild(child *ConfigBlock) {
	if cfg.children == nil {
		logger.Log("TRACE", "{%s} Initializing child map", cfg.key)
		cfg.children = make(map[*ConfigKey]*ConfigBlock)
	}

	logger.Log("TRACE", "{%s} Adding child: %s", cfg.key, child.key)

	child.parent = cfg
	cfg.children[child.key] = child
}

func (cfg *ConfigBlock) FindChild(full_key string) (child *ConfigBlock, has_child bool) {
	logger.Log("TRACE", "{%s} Looking for child: %s", cfg.key, full_key)

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

func (cfg *ConfigBlock) GetChild(key string) (child *ConfigBlock, has_child bool) {
	logger.Log("TRACE", "{%s} Fetching child: %s", cfg.key, key)

	for child_key, child := range cfg.children {
		if child_key.Key == key {
			return child, true
		}
	}

	return nil, false
}

func (cfg *ConfigBlock) GetChildren() (children map[*ConfigKey]*ConfigBlock, has_child bool) {
	logger.Log("TRACE", "{%s} Fetching all children", cfg.key)

	if len(cfg.children) > 0 {
		logger.Log("TRACE", "{%s} found '%d' children", cfg.key, len(cfg.children))
		return cfg.children, true
	}

	logger.Log("TRACE", "{%s} found no children: '%#v'", cfg.key, cfg.children)

	return nil, false
}

// Values
func (cfg *ConfigBlock) GetValues() (values []ConfigValue, has_values bool) {
	logger.Log("TRACE", "{%s} Fetching all values", cfg.key)

	if len(cfg.values) > 0 {
		logger.Log("TRACE", "{%s} found '%d' values", cfg.key, len(cfg.values))
		return cfg.values, true
	}

	logger.Log("TRACE", "{%s} found no values: '%#v'", cfg.key, cfg.values)
	return nil, false
}

// Values
func (cfg *ConfigBlock) GetValuesRecursive() (values []ConfigValue, has_values bool) {
	logger.Log("TRACE", "{%s} Fetching all (sub)values", cfg.key)

	var ret []ConfigValue

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

func (cfg *ConfigBlock) PopChild(key string) (child *ConfigBlock, had_child bool) {
	if found_child, ok := cfg.GetChild(key); ok {
		delete(cfg.children, found_child.key)
		return found_child, true
	}

	return nil, false
}

func (cfg *ConfigBlock) convertTreeToNative() map[string]interface{} {
	logger.Log("TRACE", "{%s} Converting to a tree of golang native types", cfg.key)

	var return_values []interface{}

	// Handle own values
	if self_values, ok := cfg.GetValues(); ok {
		for idx, value := range self_values {
			logger.Log("TRACE", "idx: %d, value: %s", idx, value.value)
			return_values = append(return_values, value.getValueNative())
		}
	}

	// Recurse for children / sub configs
	if children, ok := cfg.GetChildren(); ok {
		for key, child := range children {
			logger.Log("TRACE", "child: %s", key)
			child_values := child.convertTreeToNative()
			return_values = append(return_values, child_values)
		}
	}

	// Set the config key as the map key
	data := map[string]interface{}{
		cfg.key.Key: return_values,
	}

	logger.Log("TRACE", "{%s} returning result '%#v'", cfg.key, data)

	return data
}

func (cfg *ConfigBlock) MarshalJSON() ([]byte, error) {
	// Make object JSON marshalable

	logger.Log("TRACE", "{%s} MarshalJSON", cfg.key)

	j, err := json.Marshal(cfg.convertTreeToNative())
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (cfg *ConfigBlock) convertTreeToTerraform() interface{} {
	logger.Log("TRACE", "{%s} Converting to a tree of golang native types in tf structure", cfg.key)

	switch cfg.resource_type {
	case schema.TypeBool, schema.TypeFloat, schema.TypeInt, schema.TypeString:
		// Basic values
		if len(cfg.values) == 0 {
			logger.Log("ERROR", "{%s} of type schema.Type '%d', has '%d' values, expected 1.", cfg.key, cfg.resource_type, len(cfg.values))
		} else {
			if len(cfg.values) > 1 {
				logger.Log("ERROR", "{%s} of type schema.Type '%d', has '%d' values, expected 1, only including first value.", cfg.key, cfg.resource_type, len(cfg.values))
			} else {
				return cfg.values[0].getValueNative()
			}
		}
	case schema.TypeList:
		response := []interface{}{}

		// Check for values
		if values, ok := cfg.GetValues(); ok {
			for _, value := range values {
				response = append(response, value.getValueNative())
			}
		}

		// Recurse
		if children, ok := cfg.GetChildren(); ok {
			for key, child := range children {
				result := child.convertTreeToTerraform()
				response = append(response, map[string]interface{}{key.Key: result})
			}
		}

		return response

	case schema.TypeSet:
		response := []interface{}{}

		// Check for values
		if values, ok := cfg.GetValues(); ok {
			for _, value := range values {
				response = append(response, value.getValueNative())
			}
		}

		// Recurse
		if children, ok := cfg.GetChildren(); ok {
			for key, child := range children {
				result := child.convertTreeToTerraform()
				response = append(response, map[string]interface{}{key.Key: result})
			}
		}

		return response

	case schema.TypeMap:
		if values, ok := cfg.GetValues(); ok {
			logger.Log("ERROR", "{%s} of type schema.Type '%d', has '%d' values, expected zero. Values: %#v", cfg.key, cfg.resource_type, len(values), values)
		}

		// Recurse
		response := map[string]interface{}{}
		if children, ok := cfg.GetChildren(); ok {
			for key, child := range children {
				result := child.convertTreeToTerraform()
				response[key.Key] = result
			}
		}
		return response

	}

	logger.Log("ERROR", "{%s} Unable to return anything, this should never happen!", cfg.key)
	panic("Unable to return anything, this should never happen!")
}

func (cfg *ConfigBlock) MarshalTerraform() map[string]interface{} {
	// Return an object that can be used with schema.ResourceData.set() function

	logger.Log("TRACE", "{%s} MarshalTerraform", cfg.key)

	response := make(map[string]interface{})

	if children, ok := cfg.GetChildren(); ok {
		for key, child := range children {
			response[key.Key] = child.convertTreeToTerraform()
		}
	}

	return response
}

func (cfg *ConfigBlock) convertTreeToVyos() map[string]interface{} {
	// Needs to remove config parameters that are boolean == false

	logger.Log("TRACE", "{%s} Converting to a tree of vyos client compatible style golang types", cfg.key)

	var return_values []interface{}

	// Handle own values (skip bool = false)
	if self_values, ok := cfg.GetValues(); ok {
		for idx, value := range self_values {
			logger.Log("TRACE", "idx: %d, type: %d, value: %s", idx, value.value_type, value.value)
			var val interface{}
			switch value.value_type {
			case schema.TypeBool:
				if value.value == "true" {
					val = true
				} else if value.value == "false" {
					continue
				} else {
					logger.Log("ERROR", "idx: %d, value: %s, does not match bool 'true' or 'false'. Setting to 'false'.", idx, value.value)
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
			logger.Log("TRACE", "child: %s", key)

			if child.resource_type == schema.TypeBool {
				if child.values[0].value == "true" {
					return_values = append(return_values, key.Key)
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
		logger.Log("TRACE", "{%s} returning result '%#v'", cfg.key, data)
		return data
	} else {
		logger.Log("TRACE", "{%s} returning nil", cfg.key)
		return nil
	}
}

func (cfg *ConfigBlock) MarshalVyos() map[string]interface{} {
	// Return object that can be used with vyos client to create / set configs

	logger.Log("TRACE", "{%s} MarshalVyos", cfg.key)

	vy := cfg.convertTreeToVyos()
	return vy
}

func (cfg *ConfigBlock) GetDifference(compare_config *ConfigBlock) (changed *ConfigBlock, missing *ConfigBlock) {
	// Return config that represents changed (+ new) and missing parameters of this config when compared to compare_config

	logger.Log("TRACE", "{%s} GetDifference", cfg.key)

	// Keep track if we have added anything with these horrible extra booleans
	changed_used := false
	if changed == nil {
		changed = &ConfigBlock{
			key: cfg.key,
		}
	}

	missing_used := false
	if missing == nil {
		missing = &ConfigBlock{
			key: cfg.key,
		}
	}

	// Find missing values
	logger.Log("TRACE", "{%s} GetDifference, find missing values", cfg.key)
	if compare_vals, ok := compare_config.GetValues(); ok {
		for _, compare_val := range compare_vals {
			found_val := false

			for _, self_val := range cfg.values {

				if (compare_val.value_type == self_val.value_type) && (compare_val.value == self_val.value) {
					logger.Log("TRACE", "Both has value {%s}", self_val.value)
					found_val = true
					break
				}
			}

			if !found_val {
				logger.Log("TRACE", "Missing val {%s}", compare_val.value)
				missing.AddValue(compare_val.value_type, compare_val.value)
				missing_used = true
			}
		}
	}

	// Find new values
	logger.Log("TRACE", "{%s} GetDifference, find new values", cfg.key)
	if self_vals, ok := cfg.GetValues(); ok {
		for _, self_val := range self_vals {
			found_val := false

			for _, compare_val := range compare_config.values {

				if (self_val.value_type == compare_val.value_type) && (self_val.value == compare_val.value) {
					logger.Log("TRACE", "Both has value {%s}", self_val.value)
					found_val = true
					break
				}
			}

			if !found_val {
				logger.Log("TRACE", "New val {%s}", self_val.value)
				changed.AddValue(self_val.value_type, self_val.value)
				changed_used = true
			}
		}
	}

	// Help debugging by printing childrens keys
	self_keys := make([]string, len(cfg.children))
	for k := range cfg.children {
		self_keys = append(self_keys, k.Key)
	}
	logger.Log("DEBUG", "self children: %v", self_keys)

	compare_keys := make([]string, len(compare_config.children))
	for k := range compare_config.children {
		compare_keys = append(compare_keys, k.Key)
	}
	logger.Log("DEBUG", "compare children: %v", compare_keys)

	// Find missing children
	logger.Log("TRACE", "{%s} GetDifference, find missing children", cfg.key)
	if compare_children, ok := compare_config.GetChildren(); ok {
		for compare_child_key, compare_child_val := range compare_children {

			found_child := false

			if self_child, ok := cfg.GetChild(compare_child_key.Key); ok {
				logger.Log("TRACE", "Both has child {%s}", self_child.key)
				found_child = true
				continue
			}

			if !found_child {
				logger.Log("TRACE", "Missing child {%s}", compare_child_key)
				missing.AddChild(compare_child_val)
				missing_used = true
			}
		}
	}

	// Find new children
	logger.Log("TRACE", "{%s} GetDifference, find new children", cfg.key)
	if self_children, ok := cfg.GetChildren(); ok {
		for self_child_key, self_child_val := range self_children {

			found_child := false

			if compare_child, ok := compare_config.GetChild(self_child_key.Key); ok {
				logger.Log("TRACE", "Both has child {%s}", compare_child.key)
				found_child = true
				continue
			}

			if !found_child {
				logger.Log("TRACE", "New child {%s}", self_child_key)
				changed.AddChild(self_child_val)
				changed_used = true
			}
		}
	}

	// Recurse into own children, compare children should either be in this list, or in the missing list
	logger.Log("TRACE", "{%s} GetDifference, recurse operation into children", cfg.key)
	if self_children, ok := cfg.GetChildren(); ok {
		for self_child_key, self_child_val := range self_children {
			if compare_child, ok := compare_config.GetChild(self_child_key.Key); ok {

				child_changed, child_missing := self_child_val.GetDifference(compare_child)

				if child_changed != nil {
					changed.AddChild(child_changed)
					changed_used = true
				}

				if child_missing != nil {
					missing.AddChild(child_missing)
					missing_used = true
				}
			}
		}
	}

	if !changed_used {
		changed = nil
	}
	if !missing_used {
		missing = nil
	}
	return changed, missing
}