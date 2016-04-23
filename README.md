Confer
======

[![Build Status](https://travis-ci.org/jacobstr/confer.svg)](https://travis-ci.org/jacobstr/confer)

A [viper](http://github.com/spf13/viper)-derived configuration management package.

Significant changes include:

 * Materialized path access of configuration variables.
 * The singleton has been replaced by separate instances, largely for testability.
 * The ability to load and merge multiple configuration files.

Features
========

1. Merging multiple configuration sources.
  ```go
    config.ReadPaths("application.yaml", "environments/production.yaml")`
  ```

2. Materialized path access of nested configuration data.
  ```go
    config.GetInt('app.database.port')
  ```
3. Binding of environment variables to configuration data.

    `APP_DATABASE_PORT=3456 go run app.go`

4. User-defined helper methods.

## Usage

### Initialization
Create your configuration instance:

```go
config := confer.NewConfig()
```

Then set defaults, read paths, set overrides:
```go
config.SetDefault("environment", "development")
config.ReadPaths("application.yaml", "environments/production.yml")
config.Set("environment", "development")
```

**No worries!** Confer will [conveniently merge](https://github.com/jacobstr/confer/confer_test.go#L155)
deeply nested structures for you. My usual configuration setup looks like this:

```
config
  ├── application.development.yml
  ├── application.production.yml
  └── application.yml
```

For example, an application-specific config package like the one below can be used
to drive a core configuration with environment specific overrides:

```go

var App *confer.Config

func init() {
  App = confer.NewConfig()
  appenv := os.Getenv("MYAPP_ENV");
  paths := []string{"application.yml"}

  if (appenv != "") {
    paths = append(paths, fmt.Sprintf("application.%s.yml", appenv))
  }

  if err := App.ReadPaths(paths...); err != nil {
    log.Warn(err)
  }
}
```

### Setting Defaults
Sets a value if it hasn't already been set. Multiple invocations won't clobber
existing values, so you'll likely want to do this before reading from files.

```go
config := confer.NewConfig()
config.ReadPaths("application.yaml")
config.SetDefault("ContentDir", "content")
config.SetDefault("LayoutDir", "layouts")
config.SetDefault("Indexes", map[string]string{"tag": "tags", "category": "categories"})
```

### Setting Keys / Value Pairs
Sets a value. Has lower precedence than environment variables or command line flags.
```go
config.Set("verbose", true)
config.Set("logfile", "/var/log/config.log")
```
### Getting Values
There are a variety of accessors for accessing type-coerced values:
```go
Get(key string) : interface{}
GetBool(key string) : bool
GetFloat64(key string) : float64
GetInt(key string) : int
GetString(key string) : string
GetStringMap(key string) : map[string]interface{}
GetStringMapString(key string) : map[string]string
GetStringSlice(key string) : []string
GetTime(key string) : time.Time
IsSet(key string) : bool
NewConfigSubset(prefix string) : *confer.Config
```

### Deep Configuration Data
*Materialized paths* allow easy access of deeply nested config data:
```go
logger_config := config.GetStringMap("logger.stdout")
```
Because periods aren't valid environment variable characters, when using automatic environment bindings (see below), substitute with underscores:
```
LOGGER_STDOUT=/var/log/myapp go run server.go
```

### Configuration Subsets
An alternative to the `GetStringMap` / *Materialized paths* is a configuration subset. This allows you to create a new *confer.Config object which is a subset of the full configuration, specified by a key `prefix`. This makes passing around a subset of the configuration easy, allowing you to access the values using the getter methods listed above as you would normally. The setter methods are also supported on a configuration subset, as well as nested configuration subsets.

##### Example Config
```yml
---
my-app:
    logging:
        enabled: true
        level: debug
    server:
        workers:
            - 127.0.0.1
            - 127.0.0.2
            - 127.0.0.3
    database:
        bird:
            host:     feathers
            user:     postgres
            password: superl33+
        wind:
            host:     sea-spray
            user:     postgres
            password: easyPZ
```

##### Config Subset

```go
windDbConfig := config.NewConfigSubset("my-app.database.wind")
host := windDbConfig.GetString("host")
```

##### Nested Config Subset

```go
dbConfig := config.NewConfigSubset("my-app.database")
windDbConfig := dbConfig.NewConfigSubset("wind")
host := windDbConfig.GetString("host")
```

### Environment Bindings


##### Automatic Binding
Confer can automatically bind all existing configuration keys to environment variables.

Given some sort of `application.yaml`
```yaml
---
app:
   log: "verbose"
   database:
       host: "localhost"
```

And this pair of calls:

```go
config.ReadPaths("application.yaml")
config.AutomaticEnv()
```

You'll have the following environment variables exposed for configuration:
```
APP_LOG
APP_DATABASE_HOST
```

##### Selective Binding
If this automatic binding is bizarre, you can selectively bind environment variables
with ``BindEnv()`.

```go
config.BindEnv("APP_LOG", "app.log")
```

### Helpers
You can `Set` a `func() interface{}` at a configuration key to provide values dynamically:

```go
config.Set("dbstring", func() interface {} {
  return fmt.Sprintf(
    "user=%s dbname=%s sslmode=%s",
    config.GetString("database.user"),
    config.GetString("database.name"),
    config.GetString("database.sslmode"),
  )
})
assert(config.GetString("dbstring") ==  "user=doug dbname=pruden sslmode=pushups")
```
