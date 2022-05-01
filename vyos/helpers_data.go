package vyos

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

// Used to clearify child index in ConfigBlock
// Should never contain spaces
// TODO can we verify that in some way, maybe a private property with a setter or something?
type ConfigKey struct {
	string
}

// These structs will contain an internal representation of configs,
// and unify the workflow when reading terraform and VyOS config
type ConfigBlock struct {
	parent *ConfigBlock
	key    *ConfigKey

	values   []ConfigValue
	children map[ConfigKey]ConfigBlock
}

// Allow easy access from value to config block instead of using a plain string
type ConfigValue struct {
	config_block *ConfigBlock
	value        string
}

// Config keys
func (cfg *ConfigBlock) GetFullConfigKey() (full_key string) {
	logF("TRACE", "{%s} Getting full key for: %s", cfg.key, full_key)

	if cfg.parent != nil {
		full_key = cfg.parent.GetFullConfigKey() + " "
	}

	return full_key + cfg.key.string
}

// Config Values
func (cfg *ConfigBlock) AddValue(value string) (config_value *ConfigValue) {
	logF("TRACE", "{%s} Adding value: %s", cfg.key, value)

	config_v := ConfigValue{
		config_block: cfg,
		value:        value,
	}

	cfg.values = append(cfg.values, config_v)
	return &config_v
}

// Child configs
func (cfg *ConfigBlock) AddChild(key *ConfigKey, values ...string) (child *ConfigBlock) {
	if cfg.children == nil {
		logF("TRACE", "{%s} Initializing child map", cfg.key)
		cfg.children = make(map[ConfigKey]ConfigBlock)
	}

	logF("TRACE", "{%s} Adding child: %s", cfg.key, key)

	new_child := ConfigBlock{
		parent: cfg,
		key:    key,
	}

	for _, val := range values {
		new_child.values = append(
			new_child.values,
			ConfigValue{config_block: &new_child, value: val},
		)
	}

	cfg.children[*key] = new_child

	return &new_child
}

func (cfg *ConfigBlock) FindChild(full_key string) (child *ConfigBlock, has_child bool) {
	logF("TRACE", "{%s} Looking for child: %s", cfg.key, full_key)

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
	logF("TRACE", "{%s} Fetching child: %s", cfg.key, key)

	if *child, has_child = cfg.children[*key]; has_child {
		return child, has_child
	} else {
		return nil, false
	}
}

func (cfg *ConfigBlock) GetChildren() (children *map[ConfigKey]ConfigBlock, has_child bool) {
	logF("TRACE", "{%s} Fetching all children", cfg.key)

	if len(cfg.children) > 0 {
		return &cfg.children, true
	}

	return nil, false
}

// Values
func (cfg *ConfigBlock) GetValues() (values []ConfigValue, has_values bool) {
	logF("TRACE", "{%s} Fetching all values", cfg.key)

	if len(cfg.values) > 0 {
		return cfg.values, true
	}

	return nil, false
}
