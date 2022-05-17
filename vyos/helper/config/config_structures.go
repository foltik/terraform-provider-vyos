package config

import (
	"encoding/json"

	"github.com/foltik/terraform-provider-vyos/vyos/helper/logger"
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

	values   []ConfigValue
	children map[*ConfigKey]*ConfigBlock
}

// Allow easy access from value to config block instead of using a plain string
type ConfigValue struct {
	config_block *ConfigBlock
	value        *string
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
func (cfg *ConfigBlock) AddValue(value string) {
	logger.Log("TRACE", "{%s} Adding value: %s", cfg.key, value)

	config_v := ConfigValue{
		config_block: cfg,
		value:        &value,
	}

	if cfg.values == nil {
		cfg.values = []ConfigValue{config_v}
	} else {
		cfg.values = append(cfg.values, config_v)
	}
}

// Child configs
func (cfg *ConfigBlock) AddChild(key *ConfigKey, values ...string) *ConfigBlock {
	if cfg.children == nil {
		logger.Log("TRACE", "{%s} Initializing child map", cfg.key)
		cfg.children = make(map[*ConfigKey]*ConfigBlock)
	}

	logger.Log("TRACE", "{%s} Adding child: %s", cfg.key, key)

	new_child := &ConfigBlock{
		parent: cfg,
		key:    key,
	}

	for _, val := range values {
		new_child.AddValue(val)
	}

	cfg.children[key] = new_child

	return new_child
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

func (cfg *ConfigBlock) GetChild(key *ConfigKey) (child *ConfigBlock, has_child bool) {
	logger.Log("TRACE", "{%s} Fetching child: %s", cfg.key, key)

	if child, has_child = cfg.children[key]; has_child {
		return child, has_child
	} else {
		return nil, false
	}
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

// Json compatability
func (cfg *ConfigBlock) convertTreeToNative() map[string]interface{} {
	logger.Log("TRACE", "{%s} Converting to a tree of golang native types", cfg.key)

	var return_values []interface{}

	if self_values, ok := cfg.GetValues(); ok {
		for idx, value := range self_values {
			logger.Log("TRACE", "idx: %d, value: %s", idx, value.value)
			return_values = append(return_values, value.value)
		}
	}

	if children, ok := cfg.GetChildren(); ok {
		for key, child := range children {
			logger.Log("TRACE", "child: %s", key)
			child_values := child.convertTreeToNative()
			return_values = append(return_values, child_values)
		}
	}

	data := map[string]interface{}{
		cfg.key.Key: return_values,
	}

	logger.Log("TRACE", "{%s} returning result '%#v'", cfg.key, data)

	return data
}

func (cfg *ConfigBlock) MarshalJSON() ([]byte, error) {
	logger.Log("TRACE", "{%s} MarshalJSON", cfg.key)

	j, err := json.Marshal(cfg.convertTreeToNative())
	if err != nil {
		return nil, err
	}
	return j, nil
}
