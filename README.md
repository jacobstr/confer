Mamba
=====

Go configuration with fangs

## What is Mamba?

Mamba is a complete configuration solution. Designed to work within an
application to handle file based configuration and seamlessly marry that with
command line flags which can also be used to control application behavior.

We're slowly adding features that will bring us closer to Symfony's [Config
Component](http://symfony.com/doc/current/components/config/introduction.html)
because it's awesome and Go should have sweet tools.

## Why Mamba?

When building a modern application you don’t want to have to worry about
configuration file formats, you want to focus on building awesome software.
Mamba is here to help with that.

Mamba provides the following features:

1. Find, load and marshall a configuration file in YAML, TOML or JSON.
2. Provide a mechanism to setDefault values for your different configuration options
3. Provide a mechanism to setOverride values for options specified through command line flags.
4. Make it easy to tell the difference between when a user has provided a command line or config file which is the same as the default.

These are also noteworthy:

1. command line flags take precedence over options set in config files
2. config files take precedence over defaults

Config files often can be found in multiple locations. Viper allows you to set
multiple paths to search for the config file in.

Mamba configuration keys are case insensitive.

## Usage

### Initialization

	config := mamba.NewConfig()
	config.SetConfigName("config") // name of config file (without extension)
	config.AddConfigPath("/etc/appname/")   // path to look for the config file in
	config.AddConfigPath("$HOME/.appname")  // call multiple times to add many search paths
	config.ReadInConfig() // Find and read the config file

### Setting Defaults

	config.SetDefault("ContentDir", "content")
	config.SetDefault("LayoutDir", "layouts")
	config.SetDefault("Indexes", map[string]string{"tag": "tags", "category": "categories"})

### Setting Overrides

    config.Set("Verbose", true)
    config.Set("LogFile", LogFile)

### Registering and Using Aliases

    config.Set("verbose", true) // same result as next line
    config.Set("loud", true)   // same result as prior line

    config.GetBool("loud") // true
    config.GetBool("verbose") // true

### Getting Values

    config.GetString("logfile") // case insensitive Setting & Getting
	if config.GetBool("verbose") {
	    fmt.Println("verbose enabled")
	}


## Q & A

Q: Why not INI files?

A: Ini files are pretty awful. There’s no standard format and they are hard to
validate. Mamba is designed to work with YAML, TOML or JSON files. If someone
really wants to add this feature, I’d be happy to merge it. It’s easy to
specify which formats your application will permit.

Q: Where did this package come from?

A: We overhauled [Viper](https://github.com/spf13/viper) by Steve Francia.
We really liked the initial idea but wanted the freedom to move the config in
our own direction which meant dropping some of the features that we thought,
while nice, didn't contribute to writting clean code within this tool, and
withing your own codebase.
