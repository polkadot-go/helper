// config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Field struct {
	Default     interface{}
	Required    bool
	Description string
	Validator   func(interface{}) error
}

type Schema map[string]Field

var (
	registry = make(map[string]Schema)
	mu       sync.RWMutex
	instance *Config
	once     sync.Once
)

type Config struct {
	data      map[string]map[string]interface{}
	loaded    bool
	filename  string
	listeners []func(string, string, interface{})
}

func Register(section string, schema Schema) {
	mu.Lock()
	defer mu.Unlock()
	registry[section] = schema
}

func Get() *Config {
	once.Do(func() {
		instance = &Config{
			data:      make(map[string]map[string]interface{}),
			listeners: make([]func(string, string, interface{}), 0),
		}
	})
	return instance
}

func Load(filename string) error {
	return Get().LoadFile(filename)
}

func (c *Config) LoadFile(filename string) error {
	mu.Lock()
	defer mu.Unlock()

	c.filename = filename
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			c.loadDefaults()
			c.loaded = true
			return nil
		}
		return fmt.Errorf("reading config file: %w", err)
	}

	rawData := make(map[string]interface{})
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("parsing json: %w", err)
	}

	c.loadDefaults()
	if err := c.overlayData(rawData); err != nil {
		return err
	}

	if err := c.validate(); err != nil {
		return err
	}

	c.loaded = true
	return nil
}

func (c *Config) loadDefaults() {
	for section, schema := range registry {
		if c.data[section] == nil {
			c.data[section] = make(map[string]interface{})
		}
		for field, def := range schema {
			if def.Default != nil {
				c.data[section][field] = def.Default
			}
		}
	}
}

func (c *Config) overlayData(rawData map[string]interface{}) error {
	for section, sectionData := range rawData {
		if _, ok := registry[section]; !ok {
			continue
		}

		sectionMap, ok := sectionData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("section %s must be an object", section)
		}

		if c.data[section] == nil {
			c.data[section] = make(map[string]interface{})
		}

		for field, value := range sectionMap {
			old := c.data[section][field]
			c.data[section][field] = value
			c.notifyListeners(section, field, old, value)
		}
	}
	return nil
}

func (c *Config) validate() error {
	for section, schema := range registry {
		for field, def := range schema {
			value := c.data[section][field]

			if def.Required && (value == nil || value == "") {
				return fmt.Errorf("required field missing: %s.%s", section, field)
			}

			if def.Validator != nil && value != nil {
				if err := def.Validator(value); err != nil {
					return fmt.Errorf("validation failed for %s.%s: %w", section, field, err)
				}
			}
		}
	}
	return nil
}

func (c *Config) Get(section, key string) interface{} {
	mu.RLock()
	defer mu.RUnlock()

	if s, ok := c.data[section]; ok {
		return s[key]
	}
	return nil
}

func (c *Config) GetString(section, key string) string {
	if v := c.Get(section, key); v != nil {
		switch val := v.(type) {
		case string:
			return val
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func (c *Config) GetInt(section, key string) int {
	if v := c.Get(section, key); v != nil {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case float32:
			return int(val)
		}
	}
	return 0
}

func (c *Config) GetBool(section, key string) bool {
	if v := c.Get(section, key); v != nil {
		switch val := v.(type) {
		case bool:
			return val
		}
	}
	return false
}

func (c *Config) GetFloat(section, key string) float64 {
	if v := c.Get(section, key); v != nil {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0.0
}

func (c *Config) GetSection(section string) map[string]interface{} {
	mu.RLock()
	defer mu.RUnlock()

	if s, ok := c.data[section]; ok {
		copy := make(map[string]interface{})
		for k, v := range s {
			copy[k] = v
		}
		return copy
	}
	return nil
}

func (c *Config) GetDuration(section, key string) time.Duration {
	if v := c.Get(section, key); v != nil {
		switch val := v.(type) {
		case string:
			if d, err := time.ParseDuration(val); err == nil {
				return d
			}
		case int:
			return time.Duration(val) * time.Second
		case int64:
			return time.Duration(val) * time.Second
		case float64:
			return time.Duration(val * float64(time.Second))
		}
	}
	return 0
}

func (c *Config) GetStringSlice(section, key string) []string {
	if v := c.Get(section, key); v != nil {
		switch val := v.(type) {
		case []string:
			return val
		case []interface{}:
			result := make([]string, len(val))
			for i, item := range val {
				result[i] = fmt.Sprintf("%v", item)
			}
			return result
		case string:
			return strings.Split(val, ",")
		}
	}
	return nil
}

func (c *Config) Set(section, key string, value interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if c.data[section] == nil {
		c.data[section] = make(map[string]interface{})
	}
	old := c.data[section][key]
	c.data[section][key] = value
	c.notifyListeners(section, key, old, value)
}

func (c *Config) AddListener(listener func(section, key string, value interface{})) {
	mu.Lock()
	defer mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

func (c *Config) notifyListeners(section, key string, oldValue, newValue interface{}) {
	for _, listener := range c.listeners {
		listener(section, key, newValue)
	}
}

func (c *Config) Reload() error {
	if c.filename == "" {
		return fmt.Errorf("no filename set")
	}
	return c.LoadFile(c.filename)
}

func (c *Config) Watch(interval time.Duration) {
	if c.filename == "" {
		return
	}

	go func() {
		lastMod := time.Now()
		for {
			time.Sleep(interval)
			stat, err := os.Stat(c.filename)
			if err != nil {
				continue
			}
			if stat.ModTime().After(lastMod) {
				lastMod = stat.ModTime()
				c.Reload()
			}
		}
	}()
}

func GenerateTemplate() ([]byte, error) {
	mu.RLock()
	defer mu.RUnlock()

	template := make(map[string]map[string]interface{})

	for section, schema := range registry {
		template[section] = make(map[string]interface{})
		for field, def := range schema {
			template[section][field] = def.Default
		}
	}

	return json.MarshalIndent(template, "", "  ")
}

func SaveTemplate(filename string) error {
	data, err := GenerateTemplate()
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func MustLoad(filename string) {
	if err := Load(filename); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
}

func (c *Config) MustGet(section, key string) interface{} {
	v := c.Get(section, key)
	if v == nil {
		panic(fmt.Sprintf("config value not found: %s.%s", section, key))
	}
	return v
}

func (c *Config) Exists(section, key string) bool {
	mu.RLock()
	defer mu.RUnlock()

	if s, ok := c.data[section]; ok {
		_, exists := s[key]
		return exists
	}
	return false
}
