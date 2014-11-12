// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Mamba is an offshoot of Viper, written by Steve Francia and
// is an application configuration system that allows your application
// to be configured in a variety of ways
// via flags, ENVIRONMENT variables, configuration files.

// The Black Mamba is the world's most poisonous snake, and as such
// this is Golang's most deadly Config Management Tool.

// Each item takes precedence over the item below it:

// flag
// env
// config
// default

package mamba

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kr/pretty"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v1"
)

type Config struct {
	// A set of paths to look for the config file in
	paths []string

	// Name of file to look for inside the path
	name string

	// extensions Supported
	SupportedExts []string
	file          string
	configType    string
	config        map[string]interface{}
	override      map[string]interface{}
	env           map[string]string
	defaults      map[string]interface{}
	pflags        map[string]*pflag.Flag
}

func NewConfig() *Config {
	return &Config{
		name:          "config",
		SupportedExts: []string{"json", "toml", "yaml", "yml"},
		config:        make(map[string]interface{}),
		override:      make(map[string]interface{}),
		env:           make(map[string]string),
		defaults:      make(map[string]interface{}),
		pflags:        make(map[string]*pflag.Flag),
	}
}

// Explicitly define the path, name and extension of the config file
// Viper will use this and not check any of the config paths
func (self *Config) SetConfigFile(in string) {
	if in != "" {
		self.file = in
	}
}

func (self *Config) ConfigFileUsed() string {
	return self.file
}

// Add a path for viper to search for the config file in.
// Can be called multiple times to define multiple search paths.
func (self *Config) AddConfigPath(in string) {
	if in != "" {
		absin := absPathify(in)
		jww.INFO.Println("adding", absin, "to paths to search")
		if !stringInSlice(absin, self.paths) {
			self.paths = append(self.paths, absin)
		}
	}
}

func (self *Config) GetString(key string) string {
	return cast.ToString(self.Get(key))
}

func (self *Config) GetBool(key string) bool {
	return cast.ToBool(self.Get(key))
}

func (self *Config) GetInt(key string) int {
	return cast.ToInt(self.Get(key))
}

func (self *Config) GetFloat64(key string) float64 {
	return cast.ToFloat64(self.Get(key))
}

func (self *Config) GetTime(key string) time.Time {
	return cast.ToTime(self.Get(key))
}

func (self *Config) GetStringSlice(key string) []string {
	return cast.ToStringSlice(self.Get(key))
}

func (self *Config) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(self.Get(key))
}

func (self *Config) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(self.Get(key))
}

// Takes a single key and marshals it into a Struct
func (self *Config) MarshalKey(key string, rawVal interface{}) error {
	return mapstructure.Decode(self.Get(key), rawVal)
}

// Marshals the config into a Struct
func (self *Config) Marshal(rawVal interface{}) error {
	err := mapstructure.Decode(self.defaults, rawVal)
	if err != nil {
		return err
	}
	err = mapstructure.Decode(self.config, rawVal)
	if err != nil {
		return err
	}
	err = mapstructure.Decode(self.override, rawVal)
	if err != nil {
		return err
	}

	return nil
}

// Bind a specific key to a flag (as used by cobra)
//
//	 serverCmd.Flags().Int("port", 1138, "Port to run Application server on")
//	 viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
//
func (self *Config) BindPFlag(key string, flag *pflag.Flag) (err error) {
	if flag == nil {
		return fmt.Errorf("flag for %q is nil", key)
	}
	self.pflags[key] = flag

	switch flag.Value.Type() {
	case "int", "int8", "int16", "int32", "int64":
		self.SetDefault(key, cast.ToInt(flag.Value.String()))
	case "bool":
		self.SetDefault(key, cast.ToBool(flag.Value.String()))
	default:
		self.SetDefault(key, flag.Value.String())
	}
	return nil
}

// Binds a viper key to a ENV variable
// ENV variables are case sensitive
// If only a key is provided, it will use the env key matching the key, uppercased.
func (self *Config) BindEnv(input ...string) (err error) {
	var key, envkey string
	if len(input) == 0 {
		return fmt.Errorf("BindEnv missing key to bind to")
	}

	key = input[0]

	if len(input) == 1 {
		envkey = strings.ToUpper(key)
	} else {
		envkey = input[1]
	}

	self.env[key] = envkey

	return nil
}

func (self *Config) find(key string) interface{} {
	var val interface{}
	var exists bool

	flag, exists := self.pflags[key]
	if exists {
		if flag.Changed {
			jww.TRACE.Println(key, "found in override (via pflag):", val)
			return flag.Value.String()
		}
	}

	val, exists = self.override[key]
	if exists {
		jww.TRACE.Println(key, "found in override:", val)
		return val
	}

	envkey, exists := self.env[key]
	if exists {
		jww.TRACE.Println(key, "registered as env var", envkey)
		if val = os.Getenv(envkey); val != "" {
			jww.TRACE.Println(envkey, "found in environement with val:", val)
			return val
		} else {
			jww.TRACE.Println(envkey, "env value unset:")
		}
	}

	val, exists = self.config[key]
	if exists {
		jww.TRACE.Println(key, "found in config:", val)
		return val
	}

	val, exists = self.defaults[key]
	if exists {
		jww.TRACE.Println(key, "found in defaults:", val)
		return val
	}

	return nil
}

// Get returns an interface..
// Must be typecast or used by something that will typecast
func (self *Config) Get(key string) interface{} {
	key = strings.ToLower(key)
	v := self.find(key)

	if v == nil {
		return nil
	}

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

func (self *Config) IsSet(key string) bool {
	t := self.Get(key)
	return t != nil
}

// Have viper check ENV variables for all
// keys set in config, default & flags
func (self *Config) AutomaticEnv() {
	for _, x := range self.AllKeys() {
		self.BindEnv(x)
	}
}

func (self *Config) InConfig(key string) bool {
	_, exists := self.config[key]
	return exists
}

// Set the default value for this key.
// Default only used when no value is provided by the user via flag, config or ENV.
func (self *Config) SetDefault(key string, value interface{}) {
	self.defaults[strings.ToLower(key)] = value
}

// The user provided value (via flag)
// Will be used instead of values obtained via config file, ENV or default
func (self *Config) Set(key string, value interface{}) {
	self.override[key] = value
}

// Viper will discover and load the configuration file from disk
// searching in one of the defined paths.
func (self *Config) ReadInConfig() error {
	jww.INFO.Println("Attempting to read in config file")
	if !stringInSlice(self.getConfigType(), self.SupportedExts) {
		return UnsupportedConfigError(self.getConfigType())
	}

	file, err := ioutil.ReadFile(self.getConfigFile())
	if err != nil {
		return err
	}

	self.MarshallReader(bytes.NewReader(file))
	return nil
}

func (self *Config) MarshallReader(in io.Reader) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(in)

	switch self.getConfigType() {
	case "yaml", "yml":
		if err := yaml.Unmarshal(buf.Bytes(), &self.config); err != nil {
			jww.ERROR.Fatalf("Error parsing config: %s", err)
		}

	case "json":
		if err := json.Unmarshal(buf.Bytes(), &self.config); err != nil {
			jww.ERROR.Fatalf("Error parsing config: %s", err)
		}

	case "toml":
		if _, err := toml.Decode(buf.String(), &self.config); err != nil {
			jww.ERROR.Fatalf("Error parsing config: %s", err)
		}
	}
}

func (self *Config) AllKeys() []string {
	m := map[string]struct{}{}

	for key, _ := range self.defaults {
		m[key] = struct{}{}
	}

	for key, _ := range self.config {
		m[key] = struct{}{}
	}

	for key, _ := range self.override {
		m[key] = struct{}{}
	}

	a := []string{}
	for x, _ := range m {
		a = append(a, x)
	}

	return a
}

func (self *Config) AllSettings() map[string]interface{} {
	m := map[string]interface{}{}
	for _, x := range self.AllKeys() {
		m[x] = self.Get(x)
	}

	return m
}

// Name for the config file.
// Does not include extension.
func (self *Config) SetName(in string) {
	if in != "" {
		self.name = in
	}
}

func (self *Config) SetConfigType(in string) {
	if in != "" {
		self.configType = in
	}
}

func (self *Config) getConfigType() string {
	if self.configType != "" {
		return self.configType
	}

	cf := self.getConfigFile()
	ext := path.Ext(cf)

	if len(ext) > 1 {
		return ext[1:]
	} else {
		return ""
	}
}

func (self *Config) getConfigFile() string {
	// if explicitly set, then use it
	if self.file != "" {
		return self.file
	}

	cf, err := self.findConfigFile()
	if err != nil {
		return ""
	}

	self.file = cf
	return self.getConfigFile()
}

func (self *Config) searchInPath(in string) (filename string) {
	jww.DEBUG.Println("Searching for config in ", in)
	for _, ext := range self.SupportedExts {

		jww.DEBUG.Println("Checking for", path.Join(in, self.name+"."+ext))
		if b, _ := exists(path.Join(in, self.name+"."+ext)); b {
			jww.DEBUG.Println("Found: ", path.Join(in, self.name+"."+ext))
			return path.Join(in, self.name+"."+ext)
		}
	}

	return ""
}

func (self *Config) findConfigFile() (string, error) {
	jww.INFO.Println("Searching for config in ", self.paths)

	for _, cp := range self.paths {
		file := self.searchInPath(cp)
		if file != "" {
			return file, nil
		}
	}
	cwd, _ := findCWD()

	file := self.searchInPath(cwd)
	if file != "" {
		return file, nil
	}

	return "", fmt.Errorf("config file not found in: %s", self.paths)
}

func (self *Config) Debug() {
	fmt.Println("Config:")
	pretty.Println(self.config)
	fmt.Println("Env:")
	pretty.Println(self.env)
	fmt.Println("Defaults:")
	pretty.Println(self.defaults)
	fmt.Println("Override:")
	pretty.Println(self.override)
}

func (self *Config) Reset() {
	self = NewConfig()
}

// HELPERS
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func absPathify(inPath string) string {
	jww.INFO.Println("Trying to resolve absolute path to", inPath)

	if strings.HasPrefix(inPath, "$HOME") {
		inPath = userHomeDir() + inPath[5:]
	}

	if strings.HasPrefix(inPath, "$") {
		end := strings.Index(inPath, string(os.PathSeparator))
		inPath = os.Getenv(inPath[1:end]) + inPath[end:]
	}

	if filepath.IsAbs(inPath) {
		return filepath.Clean(inPath)
	}

	p, err := filepath.Abs(inPath)
	if err == nil {
		return filepath.Clean(p)
	} else {
		jww.ERROR.Println("Couldn't discover absolute path")
		jww.ERROR.Println(err)
	}
	return ""
}

type UnsupportedConfigError string

func (str UnsupportedConfigError) Error() string {
	return fmt.Sprintf("Unsupported Config Type %q", string(str))
}

// Check if File / Directory Exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func findCWD() (string, error) {
	serverFile, err := filepath.Abs(os.Args[0])

	if err != nil {
		return "", fmt.Errorf("Can't get absolute path for executable: %v", err)
	}

	path := filepath.Dir(serverFile)
	realFile, err := filepath.EvalSymlinks(serverFile)

	if err != nil {
		if _, err = os.Stat(serverFile + ".exe"); err == nil {
			realFile = filepath.Clean(serverFile + ".exe")
		}
	}

	if err == nil && realFile != serverFile {
		path = filepath.Dir(realFile)
	}

	return path, nil
}
