Confer
======

A [viper](http://gihub.com/spf13/viper) derived configuration management package. It started out as a fork but I started doing things in a dramatically different manner that it would be unmergable.

Significant changes include:

 * Materialized path access of configuration variables.
 * The singleton has been replaced by separate instances, largely for tesability.
 * The ability to load and merge multiple configuration files. Inspired in part by
   rails, each subsequent file has it's configuration data recursively merged into
   the existing configuration:
   
   * config/application.yml
   * config/environments/production.yml

Features
========

1. Merging multiple configuration sources.

   `config.ReadPaths("application.yaml", "environments/production.yaml")`

2. Materialized path access of nested configuration data.

   `config.GetInt('app.database.port')`

3. Binding of environment variables to configuration data.

	`APP_DATABASE_PORT=3456 go run app.go`


## Usage

### Initialization

    app = confer.NewConfiguration()
    app.ReadPaths("application.yaml")

### Setting Defaults
Sets a value if it hasn't already been set. Multiple invocations won't clobber
existing defaults.

    app = confer.NewConfiguration()
    app.ReadPaths("application.yaml")
    app.SetDefault("ContentDir", "content")
    app.SetDefault("LayoutDir", "layouts")
    app.SetDefault("Indexes", map[string]string{"tag": "tags", "category": "categories"})

### Setting Arbitary Values
Sets a value whether or not it's been set. Will clobber the current configuration key
value.

    app.Set("verbose", true)
    app.Set("logfile", "/var/log/app.log")

### Getting Values

    app.GetString("logFiLe") // case insensitive Setting & Getting
    if app.GetBool("verbose") {
      fmt.Println("verbose enabled")
    }

### Deep Configuration Data

	// Materialized paths allow for deep traversal of nested config data.
	logger_config := app.GetStringMap("logger.stdout")
	// Or, go even deeper.
	logger_base_path := app.GetString("logger.stdout.base_path")

	// Periods are not valid environment variable names, replace
	// materialized path periods with underscores.
	LOGGER_STDOUT_BASE_PATH=/var/log/myapp go run server.go

