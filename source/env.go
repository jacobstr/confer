package source

import (
	"fmt"
	"os"
	"strings"

	jww "github.com/spf13/jwalterweatherman"
)

// A configuration data source that manages environment variable access.
type EnvSource struct {
	index map[string]string
}

// Converts a public-facing Get() key to the corresponding, default environment
// variable key
func envamize(key string) string {
	return strings.Replace(strings.ToUpper(key), ".", "_", -1)
}

func NewEnvSource() *EnvSource {
	return &EnvSource{
		index: make(map[string]string),
	}
}

// Essentially an environment variable specific alias.
func (self *EnvSource) Bind(input ...string) (err error) {
	var key, envkey string

	if len(input) == 0 {
		return fmt.Errorf("BindEnv missing key to bind to")
	}

	if len(input) == 1 {
		key = input[0]
	} else {
		key = input[1]
	}

	envkey = envamize(key)

	jww.TRACE.Println(key, "Bound to", envkey)
	self.index[strings.ToLower(key)] = envkey

	return nil
}

// Sets an environment variable.
func (self *EnvSource) Set(key string) {
	// TODO what does this do?
}

// Gets an environment variable.
func (self *EnvSource) Get(key string) (val interface{}, exists bool) {
	envkey, exists := self.index[key]
	jww.TRACE.Println("index is", self.index)

	if exists {
		jww.TRACE.Println(key, "registered as env var", envkey)
	}

	if val = os.Getenv(envkey); val != "" {
		jww.TRACE.Println(envkey, "found in environment with val:", val)
		return val, true
	} else {
		jww.TRACE.Println(envkey, "env value unset:")
		return nil, false
	}
}
