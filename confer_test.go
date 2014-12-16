// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package confer

import (
	"fmt"
	"os"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/jacobstr/confer/reader"
	"github.com/spf13/pflag"
)

var yamlExample = []byte(`Hacker: true
name: steve
hobbies:
- skateboarding
- snowboarding
- go
clothing:
  jacket: leather
  trousers: denim
age: 35
eyes : brown
beard: true
`)

var yamlOverride = []byte(`Hacker: false
name: steve
hobbies:
- skateboarding
- dancing
awesomeness: supreme
`)

var tomlExample = []byte(`
title = "TOML Example"

[owner]
organization = "MongoDB"
Bio = "MongoDB Chief Developer Advocate & Hacker at Large"
dob = 1979-05-27T07:32:00Z # First class dates? Why not?`)

var jsonExample = []byte(`{
"id": "0001",
"type": "donut",
"name": "Cake",
"ppu": 0.55,
"batters": {
        "batter": [
                { "type": "Regular" },
                { "type": "Chocolate" },
                { "type": "Blueberry" },
                { "type": "Devil's Food" }
            ]
    }
}`)

var remoteExample = []byte(`{
"id":"0002",
"type":"cronut",
"newkey":"remote"
}`)

var application_yaml = map[string]interface{}{
	"logging": map[string]interface{}{
		"level": "info",
	},
	"database": map[string]interface{}{
		"host":     "localhost",
		"user":     "postgres",
		"password": "spend_an_hour_tweaking_your_pg_hba_for_this",
	},
	"server": map[string]interface{}{
		"workers": nil,
	},
}

var app_dev_yaml = map[string]interface{}{
	"root":    "/home/ubuntu/killer_project",
	"logging": "debug",
	"database": map[string]interface{}{
		"host":     "localhost",
		"user":     "postgres",
		"password": "spend_an_hour_tweaking_your_pg_hba_for_this",
	},
	"server": map[string]interface{}{
		"workers":       1,
		"static_assets": []interface{}{"css", "js", "img", "fonts"},
	},
}

//stubs for PFlag Values
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Type() string {
	return "string"
}

func (s *stringValue) String() string {
	return fmt.Sprintf("%s", *s)
}

func TestSpec(t *testing.T) {
	Convey("Confer", t, func() {
		config := NewConfig()

		Convey("Getting a default", func() {
			config.SetDefault("age", 45)
			So(config.Get("age"), ShouldEqual, 45)
		})

		Convey("Marhsalling", func() {
			Convey("Yaml", func() {
				yaml, _ := reader.ReadBytes(yamlExample, "yaml")
				config.MergeAttributes(yaml)

				Convey("Existence checks", func() {
					So(config.InConfig("name"), ShouldEqual, true)
					So(config.InConfig("state"), ShouldEqual, false)
				})

				Convey("Strings", func() {
					So(config.Get("name"), ShouldEqual, "steve")
				})

				Convey("Arrays", func() {
					So(
						config.Get("hobbies"),
						ShouldResemble,
						[]interface{}{"skateboarding", "snowboarding", "go"},
					)
				})

				Convey("Integers", func() {
					So(config.Get("age"), ShouldEqual, 35)
				})

				Convey("Merging", func() {
					yaml, _ := reader.ReadFile("test/fixtures/merging.yaml")

					Convey("An initial map", func() {
						root := map[string]interface{}{"users": yaml.(map[interface{}]interface{})["mapusers"]}
						config.MergeAttributes(root)
						So(config.GetStringMap("users"), ShouldResemble, map[string]interface{}{"bob": "/home/bob", "jim": "/home/jim"})

						Convey("Should be clobbered by an integer", func() {
							root := map[string]interface{}{"users": yaml.(map[interface{}]interface{})["intusers"]}
							config.MergeAttributes(root)
							So(config.Get("users"), ShouldResemble, 5)

							Convey("Should be clobbered back to a map", func() {
								root := map[string]interface{}{"users": yaml.(map[interface{}]interface{})["mapusers"]}
								config.MergeAttributes(root)
								So(
									config.GetStringMap("users"),
									ShouldResemble, map[string]interface{}{"bob": "/home/bob", "jim": "/home/jim"},
								)
							})
						})

						Convey("Should be clobbered by an array", func() {
							root := map[string]interface{}{"users": yaml.(map[interface{}]interface{})["arrayusers"]}
							config.MergeAttributes(root)
							So(config.Get("users"), ShouldResemble, []interface{}{"bob", "jim"})

							Convey("And arrays should always clobber each other", func() {
								root := map[string]interface{}{"users": yaml.(map[interface{}]interface{})["morearrayusers"]}
								config.MergeAttributes(root)
								So(
									config.Get("users"),
									ShouldResemble, []interface{}{"andy"},
								)
							})
						})

						Convey("Should be extended by another map", func() {
							root := map[string]interface{}{"users": yaml.(map[interface{}]interface{})["moreusers"]}
							config.MergeAttributes(root)
							So(
								config.GetStringMap("users"),
								ShouldResemble,
								map[string]interface{}{"bob": "/home/bob", "jim": "/home/jim", "andy": "/home/andy"},
							)
						})
					})
				})
			})

			Convey("Toml", func() {
				toml, _ := reader.ReadBytes(tomlExample, "toml")
				config.MergeAttributes(toml)
				So(config.Get("owner.organization"), ShouldEqual, "MongoDB")
			})

			Convey("Json", func() {
				json, _ := reader.ReadBytes(jsonExample, "json")
				config.MergeAttributes(json)
				So(config.Get("ppu"), ShouldEqual, 0.55)
			})
		})

		Convey("Defaults, Overrides, Files", func() {
			Convey("Defaults", func() {
				config.SetDefault("clothing.jacket", "poncho")
				config.SetDefault("age", 99)

				So(config.Get("clothing.jacket"), ShouldEqual, "poncho")
				So(config.Get("age"), ShouldEqual, 99)

				Convey("Files should clobber defaults", func() {
					yaml, _ := reader.ReadBytes(yamlExample, "yaml")
					config.MergeAttributes(yaml)

					So(config.Get("clothing.jacket"), ShouldEqual, "leather")
					So(config.Get("age"), ShouldEqual, 35)

					Convey("Overrides should clobber files", func() {
						config.Set("clothing.jacket", "peacoat")
						config.Set("age", 30)
						So(config.Get("clothing.jacket"), ShouldEqual, "peacoat")
						So(config.Get("age"), ShouldEqual, 30)

						So(config.GetStringMap("clothing")["jacket"], ShouldEqual, "peacoat")
					})

					Convey("All three sources should appear in AllKeys()", func() {
						keys := config.AllKeys()
						sort.Strings(keys)
						So(
							keys,
							ShouldResemble,
							[]string{
								"age",
								"beard",
								"clothing.jacket",
								"clothing.trousers",
								"eyes",
								"hacker",
								"hobbies",
								"name",
							})
					})
				})
			})
		})

		Convey("PFlags", func() {
			testString := "testing"
			testValue := newStringValue(testString, &testString)

			flag := &pflag.Flag{
				Name:    "testflag",
				Value:   testValue,
				Changed: false,
			}

			Convey("Should not appear in AllKeys() initially", func() {
				So(config.AllKeys(), ShouldResemble, []string{})
			})

			// Initial assertions after binding.
			config.BindPFlag("testflag", flag)
			So(config.Get("testflag"), ShouldEqual, "testing")

			Convey("Should appear in AllKeys()", func() {
				So(config.AllKeys(), ShouldResemble, []string{"testflag"})
			})

			Convey("Insensitivity before mutation", func() {
				So(config.Get("testFlag"), ShouldEqual, "testing")
			})

			flag.Value.Set("testing_mutate")
			flag.Changed = true //hack for pflag usage
			So(config.Get("testflag"), ShouldEqual, "testing_mutate")

			Convey("Insensitivity after mutation", func() {
				So(config.Get("testFlag"), ShouldEqual, "testing_mutate")
			})
		})

		Convey("ReadPaths", func() {

			Convey("Single Path", func() {
				config.ReadPaths("test/fixtures/application.yaml")
				So(config.GetStringMap("app"), ShouldResemble, application_yaml)
			})

			Convey("Multiple Paths", func() {
				Convey("With A Missing File", func() {
					config.ReadPaths("test/fixtures/application.yaml", "test/fixtures/missing.yaml")
					So(config.GetStringMap("app"), ShouldResemble, application_yaml)
				})

				Convey("With An Augmented Environment", func() {
					config.ReadPaths("test/fixtures/application.yaml", "test/fixtures/environments/development.yaml")
					So(config.GetStringMap("app"), ShouldResemble, app_dev_yaml)

					Convey("Deep access", func() {
						So(config.GetString("app.database.host"), ShouldEqual, "localhost")
					})
				})
			})

			Convey("Rooted paths", func() {
				config.SetRootPath("test/fixtures")
				config.ReadPaths("application.yaml")
				So(config.GetStringMap("app"), ShouldResemble, application_yaml)
			})
		})

		Convey("Environment Variables", func() {
			Convey("Automatic Env", func() {
				config.ReadPaths("test/fixtures/application.yaml")
				os.Setenv("APP_LOGGING_LEVEL", "trace")
				config.AutomaticEnv()
				So(config.Get("app.logging.level"), ShouldEqual, "trace")
			})

			Convey("Underscore translation", func() {
				config.ReadPaths("test/fixtures/env_underscores.yaml")
				os.Setenv("AWESOME_SAUCE_HEAT_LEVEL_IS_RADICAL", "yep!")
				config.AutomaticEnv()
				So(config.Get("awesome_sauce.heat_level.is_radical"), ShouldEqual, "yep!")
			})
		})

		Convey("Case Sensitivity", func() {
			config.ReadPaths("test/fixtures/application.yaml")
			funky := "aPp.DatAbase.host"
			regular := "app.database.host"
			So(config.GetString(funky), ShouldResemble, "localhost")

			Convey("Should manage case-insensitive key collissions", func() {
				config.Set(funky, "woot")
				So(config.GetString(funky), ShouldEqual, "woot")
				So(config.GetString(regular), ShouldEqual, "woot")

				config.Set(regular, "localhost")
				So(config.GetString(funky), ShouldEqual, "localhost")
				So(config.GetString(regular), ShouldEqual, "localhost")
			})
		})

		Convey("Helpers", func() {
			Convey("Returning an integer", func() {
				config.Set("port", func() interface{} {
					return 5
				})
				So(config.GetInt("port"), ShouldEqual, 5)
			})

			Convey("Returning a stringmap", func() {
				config.Set("database", func() interface{} {
					return map[string]string{"host": "localhost"}
				})
				So(config.GetStringMapString("database"), ShouldResemble, map[string]string{"host": "localhost"})
			})

			Convey("Dbstring example", func() {
				config.Set("database.user", "doug")
				config.Set("database.dbname", "pruden")
				config.Set("database.sslmode", "pushups")

				config.Set("dbstring", func() interface{} {
					return fmt.Sprintf(
						"user=%s dbname=%s sslmode=%s",
						config.GetString("database.user"),
						config.GetString("database.dbname"),
						config.GetString("database.sslmode"),
					)
				})
				So(config.GetString("dbstring"), ShouldEqual, "user=doug dbname=pruden sslmode=pushups")
			})
		})

		Convey("AllSettings", func() {
			Convey("Should only include leaves", func() {
				config.ReadPaths("test/fixtures/application.yaml")
				So(config.AllSettings(), ShouldResemble, map[string]interface{} {
					"app.logging.level" : "info",
					"app.database.host" : "localhost",
					"app.database.user" : "postgres",
					"app.database.password" : "spend_an_hour_tweaking_your_pg_hba_for_this",
					"app.server.workers" : nil,
				})
			})

			Convey("Should include stubbed deep values", func() {
				config.Set("api.credentials.secret", "password")
				So(config.AllSettings(), ShouldResemble, map[string]interface{} {
					"api.credentials.secret" : "password",
				})

				Convey("And retain them when we merge data", func() {
					config.MergeAttributes(map[string]interface{} { "api": map[string]interface {} { "user" : "stallman1337@hotmail.com"} })
					So(config.AllSettings(), ShouldResemble, map[string]interface{} {
						"api.credentials.secret" : "password",
						"api.user" : "stallman1337@hotmail.com",
					})
				})
			})
		})
	})
}

// About 9500 ns / op on my system.
func BenchmarkIntAccess(b *testing.B) {
	configAttrs := make(map[string]interface{})

	for i := 0; i < b.N; i++ {
		configAttrs[fmt.Sprintf("attr%s", i)] = i
	}

	config := NewConfig()
	config.MergeAttributes(configAttrs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.GetInt(fmt.Sprintf("attr%s", i))
	}
}

// About 9500 ns / op on my system.
func BenchmarkHelperAccess(b *testing.B) {
	configAttrs := make(map[string]interface{})

	for i := 0; i < b.N; i++ {
		configAttrs[fmt.Sprintf("attr%s", i)] = func(c *Config) interface{} {
			return 5
		}
	}

	config := NewConfig()
	config.MergeAttributes(configAttrs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.GetInt(fmt.Sprintf("attr%s", i))
	}
}
