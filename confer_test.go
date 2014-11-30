// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package confer

import (
	"fmt"
	"os"
	"testing"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/spf13/pflag"
	"github.com/jacobstr/confer/reader"
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


var application_yaml = map[string]interface{} {
	"logging": map[string]interface{} {
		"level" : "info",
	},
	"database": map[string]interface{} {
		"host" : "localhost",
		"user" : "postgres",
		"password" : "spend_an_hour_tweaking_your_pg_hba_for_this",
	},
	"server": map[string]interface{} {
		"workers" : nil,
	},
}

var app_dev_yaml = map[string]interface{} {
	"root": "/home/ubuntu/killer_project",
	"logging": "debug",
	"database": map[string]interface{} {
		"host" : "localhost",
		"user" : "postgres",
		"password" : "spend_an_hour_tweaking_your_pg_hba_for_this",
	},
	"server": map[string]interface{} {
		"workers" : 1,
		"static_assets": []interface{}{ "css", "js", "img" , "fonts" },
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
	Convey("Map-Like Config Sources", t, func() {
		config := NewConfiguration()

		Convey("Getting a default", func() {
			config.SetDefault("age", 45)
			So(config.Get("age"), ShouldEqual, 45)
		})

		Convey("Marhsalling", func() {
			Convey("Yaml", func() {
				yaml, _ := reader.Readbytes(yamlExample, "yaml")
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
						[]interface{} {"skateboarding", "snowboarding", "go"},
					)
				})

				Convey("Integers", func() {
					So(config.Get("age"), ShouldEqual, 35)
				})

				Convey("Merging", func() {
					yaml, _ := reader.Readbytes(yamlOverride, "yaml")
					config.MergeAttributes(yaml)
					So(config.Get("awesomeness"), ShouldEqual, "supreme")
					So(config.Get("hobbies"), ShouldResemble, []interface{} { "skateboarding", "dancing" })
				})
			})

			Convey("Toml", func() {
				toml, _ := reader.Readbytes(tomlExample, "toml")
				config.MergeAttributes(toml)
				So(config.Get("owner.organization"), ShouldEqual, "MongoDB")
			})

			Convey("Json", func() {
				json, _ := reader.Readbytes(jsonExample, "json")
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
					yaml, _ := reader.Readbytes(yamlExample, "yaml")
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

			// Initial assertions after binding.
			config.BindPFlag("testvalue", flag)
			So(config.Get("testvalue"), ShouldEqual, "testing")

			Convey("Insensitivity before mutation", func() {
				So(config.Get("testValue"), ShouldEqual, "testing")
			})

			flag.Value.Set("testing_mutate")
			flag.Changed = true //hack for pflag usage
			So(config.Get("testvalue"), ShouldEqual, "testing_mutate")

			Convey("Insensitivity after mutation", func() {
				So(config.Get("testValue"), ShouldEqual, "testing_mutate")
			})
		})
	})

	Convey("ReadPaths", t, func(){

		Convey("Single Path", func() {
			config := NewConfiguration()
			config.ReadPaths("test/fixtures/application.yaml")
			So(config.GetStringMap("app"), ShouldResemble, application_yaml)
		})

		Convey("Multiple Paths", func() {
			config := NewConfiguration()
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
			config := NewConfiguration()
			config.SetRootPath("test/fixtures")
			config.ReadPaths("application.yaml")
			So(config.GetStringMap("app"), ShouldResemble, application_yaml)
		})
	})

	Convey("Environment Variables", t, func() {
		config := NewConfiguration()
		config.ReadPaths("test/fixtures/application.yaml")
		Convey("Automatic Env", func() {
			os.Setenv("APP_LOGGING_LEVEL", "trace")
			config.AutomaticEnv()
			So(config.Get("app.logging.level"), ShouldEqual, "trace")
		})
	})

	Convey("Case Sensitivity", t, func() {
		config := NewConfiguration()
		config.ReadPaths("test/fixtures/application.yaml")
		So(config.GetString("aPp.DatAbase.host"), ShouldResemble, "localhost")
	})
}
