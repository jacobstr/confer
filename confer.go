// Copyright © 2014 Steve Francia <spf@spf13.com>.
// Copyright © 2014 Jacob Straszysnki <jacobstr@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Viper is a application configuration system.
// It believes that applications can be configured a variety of ways
// via flags, ENVIRONMENT variables, configuration files retrieved
// from the file system, or a remote key/value store.

// Each item takes precedence over the item below it:

// flag
// env
// config
// default

package confer

import (
	"fmt"
	"strings"
	"time"

	"github.com/kr/pretty"
	"github.com/spf13/cast"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/pflag"

	. "github.com/jacobstr/confer/source"
	"github.com/jacobstr/confer/reader"

	errors "github.com/jacobstr/confer/errors"
	"github.com/jacobstr/confer/maps"
)

// extensions Supported
var SupportedExts []string = []string{"json", "toml", "yaml", "yml"}
var SupportedRemoteProviders []string = []string{"etcd", "consul"}
var configFile string
var configType string

// Manages key/value access and aliasing across multiple configuration sources.
type ConfigManager struct {
	attributes *ConfigSource

	// These ought to just be Configgers, but they're somewhat specialized.
	pflags *PFlagSource
	env    *EnvSource
}

func NewConfiguration() *ConfigManager {
	manager := &ConfigManager{}
	manager.pflags = NewPFlagSource()
	manager.attributes = NewConfigSource()
	manager.env = NewEnvSource()

	return manager
}

func (self *ConfigManager) Find(key string) interface{} {
	var val interface{}
	var exists bool

	// PFlag Override first
	val, exists = self.pflags.Get(key)
	if exists {
		jww.TRACE.Println(key, "found in override (via pflag):", val)
		return val
	}

	// Periods are not supported. Allow the usage of double underscores to specify
	// nested configuration options.
	val, exists = self.env.Get(key)
	if exists {
		jww.TRACE.Println(key, "Found in environment with value:", val)
		return val
	}

	val, exists = self.attributes.Get(key)
	if exists {
		jww.TRACE.Println(key, "Found in config:", val)
		return val
	}

	return nil
}

func (manager *ConfigManager) GetString(key string) string {
	return cast.ToString(manager.Get(key))
}

func (manager *ConfigManager) GetBool(key string) bool {
	return cast.ToBool(manager.Get(key))
}

func (manager *ConfigManager) GetInt(key string) int {
	return cast.ToInt(manager.Get(key))
}

func (manager *ConfigManager) GetFloat64(key string) float64 {
	return cast.ToFloat64(manager.Get(key))
}

func (manager *ConfigManager) GetTime(key string) time.Time {
	return cast.ToTime(manager.Get(key))
}

func (manager *ConfigManager) GetStringSlice(key string) []string {
	return cast.ToStringSlice(manager.Get(key))
}

func (manager *ConfigManager) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(manager.Get(key))
}

func (manager *ConfigManager) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(manager.Get(key))
}

// Bind a specific key to a flag (as used by cobra)
//
//	 serverCmd.Flags().Int("port", 1138, "Port to run Application server on")
//	 confer.BindPFlag("port", serverCmd.Flags().Lookup("port"))
//
func (manager *ConfigManager) BindPFlag(key string, flag *pflag.Flag) (err error) {
	if flag == nil {
		return fmt.Errorf("flag for %q is nil", key)
	}

	manager.pflags.Set(key, flag)

	switch flag.Value.Type() {
	case "int", "int8", "int16", "int32", "int64":
		manager.SetDefault(key, cast.ToInt(flag.Value.String()))
	case "bool":
		manager.SetDefault(key, cast.ToBool(flag.Value.String()))
	default:
		manager.SetDefault(key, flag.Value.String())
	}
	return nil
}

// Binds a confer key to a ENV variable
// ENV variables are case sensitive
// If only a key is provided, it will use the env key matching the key, uppercased.
func (manager *ConfigManager) BindEnv(input ...string) (err error) {
	return manager.env.Bind(input...)
}

// Get returns an interface..
// Must be typecast or used by something that will typecast
func (manager *ConfigManager) Get(key string) interface{} {
	jww.TRACE.Println("Looking for", key)

	v := manager.Find(key)

	if v == nil {
		return nil
	}

	jww.TRACE.Println("Found value", v)
	switch v.(type) {
	case bool:
		return cast.ToBool(v)
	case string:
		return cast.ToString(v)
	case int64, int32, int16, int8, int:
		return cast.ToInt(v)
	case float64, float32:
		return cast.ToFloat64(v)
	case time.Time:
		return cast.ToTime(v)
	case []string:
		return v
	}
	return v
}

func (manager *ConfigManager) IsSet(key string) bool {
	t := manager.Get(key)
	return t != nil
}

// Have confer check ENV variables for all
// keys set in config, default & flags
func (manager *ConfigManager) AutomaticEnv() {
	for _, x := range manager.AllKeys() {
		manager.BindEnv(x)
	}
}

func (manager *ConfigManager) InConfig(key string) bool {
	// if the requested key is an alias, then return the proper key
	_, exists := manager.attributes.Get(key)
	return exists
}

// Set the default value for this key.
// Default only used when no value is provided by the user via flag, config or ENV.
func (manager *ConfigManager) SetDefault(key string, value interface{}) {
	if (!manager.IsSet(key)) {
		manager.attributes.Set(key, value)
	}
}

// Explicitly sets a value. This is order dependent, e.g. it will override the current
// value but may be overriden itself e.g. if one subsequently reads a YAML file. Therefore,
// precedence is simply established by order in which you execute your configuration
// instructions.
func (manager *ConfigManager) Set(key string, value interface{}) {
	manager.attributes.Set(key, value)
}

// Loads and sequentially + recursively merges the provided config arguments. Returns
// an error if any of the files fail to load, though this may be expecte
// in the case of search paths.
func (manager *ConfigManager) ReadPaths(paths ...string) error {
	var err error
	var loaded interface{}

	merged_config := manager.attributes.ToStringMap()
	errs := []error{}

	for _, path := range paths {
		loaded, err = reader.Readfile(path)

		if err != nil {
			errs = append(errs, err)
			break
		}

		// In-place recursive coercion to stringmap.
		coerced := cast.ToStringMap(loaded)
		maps.ToStringMapRecursive(coerced)

		if merged_config == nil {
			merged_config = coerced
		} else {
			merged_config = maps.Merge(
				merged_config,
				coerced,
			)
		}

		manager.attributes.FromStringMap(merged_config)
	}

	if len(errs) > 0 {
		return &errors.LoadError{Errors: errs}
	} else {
		return nil
	}
}

// Merges data into the our attributes configuration tier from a struct.
func (manager *ConfigManager) MergeAttributes(val interface{}) error {
	merged_config := maps.Merge(
		manager.attributes.ToStringMap(),
		cast.ToStringMap(val),
	)

	manager.attributes.FromStringMap(merged_config)
	return nil
}

func (manager *ConfigManager) AllKeys() []string {
	m := map[string]struct{}{}

	for key, _ := range manager.attributes.AllKeys() {
		m[key] = struct{}{}
	}

	a := []string{}
	for x, _ := range m {
		// LowerCase the key for backwards-compatibility.
		a = append(a, strings.ToLower(x))
	}

	return a
}

func (manager *ConfigManager) AllSettings() map[string]interface{} {
	m := map[string]interface{}{}
	for _, x := range manager.AllKeys() {
		m[x] = manager.Get(x)
	}

	return m
}

func (manager *ConfigManager) Debug() {
	fmt.Println("Config file attributes:")
	pretty.Println(manager.attributes)
	fmt.Println("Env:")
	pretty.Println(manager.env)
}
