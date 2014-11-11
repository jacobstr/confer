// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package viper

import (
	"fmt"
	"os"
	// "sort"
	"testing"
	// "time"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/davecgh/go-spew/spew"

	"github.com/spf13/pflag"
	"github.com/jacobstr/viper/reader"
	// "github.com/stretchr/testify/assert"
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
		config.SetDefault("age", 45)

		Convey("Getting a default", func() {
			So(config.Get("age"), ShouldEqual, 45)
		})

		Convey("Marhsalling", func() {
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
				// TODO assertions??
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

		application_yaml := map[string]interface{} {
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

		app_dev_yaml := map[string]interface{} {
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

		Convey("Single Path", func() {
			config := NewConfiguration()
			config.ReadPaths("test/fixtures/application.yaml")
			So(config.GetStringMap("app"), ShouldResemble, application_yaml)
		})

		Convey("Multiple Paths", func() {
			config := NewConfiguration()
			Convey("With A Missing File", func() {
				config.ReadPaths("test/fixtures/application.yaml", "test/fixtures/missing.yaml")
				fmt.Println("241 viper_test", config.GetStringMap("app"))
				So(config.GetStringMap("app"), ShouldResemble, application_yaml)
				fmt.Println(app_dev_yaml)
			})

			Convey("With An Augmented Environment", func() {
				config.ReadPaths("test/fixtures/application.yaml", "test/fixtures/environments/development.yaml")
				fmt.Println(config.GetStringMap("app"))
				spew.Dump(config.GetStringMap("app"))
				spew.Dump(app_dev_yaml)
				So(config.GetStringMap("app"), ShouldResemble, app_dev_yaml)

				Convey("Deep access", func() {
					So(config.GetString("app.database.host"), ShouldEqual, "localhost")
				})
			})
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
}


// func TestJSON(t *testing.T) {
//   SetConfigType("json")
//   r := bytes.NewReader(jsonExample)

//   MarshallReader(r, manager.filesystem)
//   assert.Equal(t, "0001", Get("id"))
// }

// func TestTOML(t *testing.T) {
//   SetConfigType("toml")
//   r := bytes.NewReader(tomlExample)

//   MarshallReader(r, manager.filesystem)
//   assert.Equal(t, "TOML Example", Get("title"))
// }

// func TestEnv(t *testing.T) {
//   SetConfigType("json")
//   r := bytes.NewReader(jsonExample)
//   MarshallReader(r, manager.filesystem)
//   BindEnv("id")
//   BindEnv("f", "FOOD")

//   os.Setenv("ID", "13")
//   os.Setenv("FOOD", "apple")
//   os.Setenv("NAME", "crunk")

//   assert.Equal(t, "13", Get("id"))
//   // assert.Equal(t, "apple", Get("f"))
//   // assert.Equal(t, "Cake", Get("name"))

//   // AutomaticEnv()

//   // assert.Equal(t, "crunk", Get("name"))
// }

// // func TestAllKeys(t *testing.T) {
// //   ks := sort.StringSlice{"title", "newkey", "owner", "name", "beard", "ppu", "batters", "hobbies", "clothing", "age", "hacker", "id", "type", "eyes"}
// //   dob, _ := time.Parse(time.RFC3339, "1979-05-27T07:32:00Z")
// //   all := map[string]interface{}{"hacker": true, "beard": true, "newkey": "remote", "batters": map[string]interface{}{"batter": []interface{}{map[string]interface{}{"type": "Regular"}, map[string]interface{}{"type": "Chocolate"}, map[string]interface{}{"type": "Blueberry"}, map[string]interface{}{"type": "Devil's Food"}}}, "hobbies": []interface{}{"skateboarding", "snowboarding", "go"}, "ppu": 0.55, "clothing": map[interface{}]interface{}{"jacket": "leather", "trousers": "denim"}, "name": "crunk", "owner": map[string]interface{}{"organization": "MongoDB", "Bio": "MongoDB Chief Developer Advocate & Hacker at Large", "dob": dob}, "id": "13", "title": "TOML Example", "age": 35, "type": "donut", "eyes": "brown"}

// //   var allkeys sort.StringSlice
// //   allkeys = AllKeys()
// //   allkeys.Sort()
// //   ks.Sort()

// //   assert.Equal(t, ks, allkeys)
// //   assert.Equal(t, all, AllSettings())
// // }

// // func TestCaseInSensitive(t *testing.T) {
// //   assert.Equal(t, true, Get("hacker"))
// //   Set("Title", "Checking Case")
// //   assert.Equal(t, "Checking Case", Get("tItle"))
// // }

// // func TestAliasesOfAliases(t *testing.T) {
// //   RegisterAlias("Foo", "Bar")
// //   RegisterAlias("Bar", "Title")
// //   assert.Equal(t, "Checking Case", Get("FOO"))
// // }

// // func TestRecursiveAliases(t *testing.T) {
// //   RegisterAlias("Baz", "Roo")
// //   RegisterAlias("Roo", "baz")
// // }

// // func TestMarshal(t *testing.T) {
// //   SetDefault("port", 1313)
// //   Set("name", "Steve")

// //   type config struct {
// //     Port int
// //     Name string
// //   }

// //   var C config

// //   err := Marshal(&C)
// //   if err != nil {
// //     t.Fatalf("unable to decode into struct, %v", err)
// //   }

// //   assert.Equal(t, &C, &config{Name: "Steve", Port: 1313})

// //   Set("port", 1234)
// //   err = Marshal(&C)
// //   if err != nil {
// //     t.Fatalf("unable to decode into struct, %v", err)
// //   }
// //   assert.Equal(t, &C, &config{Name: "Steve", Port: 1234})
// // }

// // func TestDeepAccess(t *testing.T) {
// //   assert.Equal(t, "leather", Get("clothing.jacket"))
// // }

// // func TestDeepBindEnv(t *testing.T) {
// //   BindEnv("clothing.jacket")
// //   os.Setenv("CLOTHING__JACKET", "peacoat")
// //   assert.Equal(t, "peacoat", Get("clothing.jacket"))
// // }

// // func TestDeepAutomaticEnv(t *testing.T) {
// //   AutomaticEnv()
// //   os.Setenv("CLOTHING__JACKET", "jean")
// //   assert.Equal(t, "jean", Get("clothing.jacket"))
// // }

// // func TestBoundCaseSensitivity(t *testing.T) {

// //   assert.Equal(t, "brown", Get("eyes"))

// //   BindEnv("eYEs", "TURTLE_EYES")
// //   os.Setenv("TURTLE_EYES", "blue")

// //   assert.Equal(t, "blue", Get("eyes"))

// //   var testString = "green"
// //   var testValue = newStringValue(testString, &testString)

// //   flag := &pflag.Flag{
// //     Name:    "eyeballs",
// //     Value:   testValue,
// //     Changed: true,
// //   }

// //   BindPFlag("eYEs", flag)
// //   assert.Equal(t, "green", Get("eyes"))
// // }
