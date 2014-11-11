viper
=====

Go configuration with fangs

## What is Viper?

A configuration management module that handles:

1. Merging multiple configuration sources.

   `config.ReadPaths("application.yaml", "environments/production.yaml")`

2. Materialized path access of nested configuration data.

   `config.GetInt('app.database.port')`

3. Binding of environment variables to configuration data.

  `APP_DATABASE_PORT=3456 go run app.go`

## Why Viper?

When building a modern application you don’t want to have to worry about
configuration file formats, you want to focus on building awesome software.
Viper is here to help with that.

Viper does the following for you:

1. Find, load and marshall a configuration file in YAML, TOML or JSON.
2. Provide a mechanism to setDefault values for your different configuration options
3. Provide a mechanism to setOverride values for options specified through command line flags.
4. Make it easy to tell the difference between when a user has provided a command line or config file which is the same as the default.

Viper believes that:

1. command line flags take precedence over options set in config files
2. config files take precedence over options set in remote key/value stores
3. remote key/value stores take precedence over defaults

Viper configuration keys are case insensitive.

## Usage

### Initialization

    app = viper.NewConfiguration()
    app.ReadPaths("application.yaml")

### Setting Defaults

    app = viper.NewConfiguration()
    app.ReadPaths("application.yaml")
    app.SetDefault("ContentDir", "content")
    app.SetDefault("LayoutDir", "layouts")
    app.SetDefault("Indexes", map[string]string{"tag": "tags", "category": "categories"})

### Setting Overrides

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

## Q & A

Q: Why not INI files?

A: Ini files are pretty awful. There’s no standard format and they are hard to
validate. Viper is designed to work with YAML, TOML or JSON files. If someone
really wants to add this feature, I’d be happy to merge it. It’s easy to
specify which formats your application will permit.

Q: Why is it called "viper"?

A: Viper is designed to be a companion to
[Cobra](http://github.com/spf13/cobra). While both can operate completely
independently, together they make a powerful pair to handle much of your
application foundation needs.

Q: Why is it called "Cobra"?

A: Is there a better name for a commander?
