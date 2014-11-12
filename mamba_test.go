// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mamba

import (
	"bytes"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
beard: true
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

// Setup Test Suite
type TestSuite struct {
	suite.Suite
	Config *Config
}

func (test *TestSuite) SetupTest() {
	test.Config = NewConfig()
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// Unit Tests
func (test *TestSuite) TestConstructor() {
	config := NewConfig()
	assert.NotNil(test.T(), config)
	assert.Equal(test.T(), []string{"json", "toml", "yaml", "yml"}, config.SupportedExts)
}

func (test *TestSuite) TestBasics() {
	test.Config.SetConfigFile("/tmp/config.yaml")
	assert.Equal(test.T(), "/tmp/config.yaml", test.Config.getConfigFile())
}

func (test *TestSuite) TestDefault() {
	test.Config.SetDefault("age", 45)
	assert.Equal(test.T(), 45, test.Config.Get("age"))
}

func (test *TestSuite) TestMarshalling() {
	test.Config.SetConfigType("yaml")
	r := bytes.NewReader(yamlExample)

	test.Config.MarshallReader(r)
	assert.True(test.T(), test.Config.InConfig("name"))
	assert.False(test.T(), test.Config.InConfig("state"))
	assert.Equal(test.T(), "steve", test.Config.Get("name"))
	assert.Equal(test.T(), []interface{}{"skateboarding", "snowboarding", "go"}, test.Config.Get("hobbies"))
	assert.Equal(test.T(), map[interface{}]interface{}{"jacket": "leather", "trousers": "denim"}, test.Config.Get("clothing"))
	assert.Equal(test.T(), 35, test.Config.Get("age"))
}

func (test *TestSuite) TestOverrides() {
	test.Config.Set("age", 40)
	assert.Equal(test.T(), 40, test.Config.Get("age"))
}

func (test *TestSuite) TestDefaultPost() {
	assert.NotEqual(test.T(), "NYC", test.Config.Get("state"))
	test.Config.SetDefault("state", "NYC")
	assert.Equal(test.T(), "NYC", test.Config.Get("state"))
}

func (test *TestSuite) TestYML() {
	test.Config.Reset()
	test.Config.SetConfigType("yml")
	r := bytes.NewReader(yamlExample)

	test.Config.MarshallReader(r)
	assert.Equal(test.T(), "steve", test.Config.Get("name"))
}

func (test *TestSuite) TestJSON() {
	test.Config.SetConfigType("json")
	r := bytes.NewReader(jsonExample)

	test.Config.MarshallReader(r)
	assert.Equal(test.T(), "0001", test.Config.Get("id"))
}

func (test *TestSuite) TestTOML() {
	test.Config.SetConfigType("toml")
	r := bytes.NewReader(tomlExample)

	test.Config.MarshallReader(r)
	assert.Equal(test.T(), "TOML Example", test.Config.Get("title"))
}

func (test *TestSuite) TestEnv() {
	test.Config.SetConfigType("json")
	r := bytes.NewReader(jsonExample)
	test.Config.MarshallReader(r)
	test.Config.BindEnv("id")
	test.Config.BindEnv("f", "FOOD")

	os.Setenv("ID", "13")
	os.Setenv("FOOD", "apple")
	os.Setenv("NAME", "crunk")

	assert.Equal(test.T(), "13", test.Config.Get("id"))
	assert.Equal(test.T(), "apple", test.Config.Get("f"))
	assert.Equal(test.T(), "Cake", test.Config.Get("name"))

	test.Config.AutomaticEnv()

	assert.Equal(test.T(), "crunk", test.Config.Get("name"))
}

func (test *TestSuite) TestAllKeys() {
  ks := sort.StringSlice{"title", "owner", "name", "beard", "ppu", "batters", "hobbies", "clothing", "age", "hacker", "id", "type"}
  dob, _ := time.Parse(time.RFC3339, "1979-05-27T07:32:00Z")
  all := map[string]interface{}{"hacker": true, "beard": true, "batters": map[string]interface{}{"batter": []interface{}{map[string]interface{}{"type": "Regular"}, map[string]interface{}{"type": "Chocolate"}, map[string]interface{}{"type": "Blueberry"}, map[string]interface{}{"type": "Devil's Food"}}}, "hobbies": []interface{}{"skateboarding", "snowboarding", "go"}, "ppu": 0.55, "clothing": map[interface{}]interface{}{"jacket": "leather", "trousers": "denim"}, "name": "crunk", "owner": map[string]interface{}{"organization": "MongoDB", "Bio": "MongoDB Chief Developer Advocate & Hacker at Large", "dob": dob}, "id": "13", "title": "TOML Example", "age": 35, "type": "donut"}
  test.Config.config = all

  var allkeys sort.StringSlice
  allkeys = test.Config.AllKeys()
  allkeys.Sort()
  ks.Sort()

  assert.Equal(test.T(), ks, allkeys)
  assert.Equal(test.T(), all, test.Config.AllSettings())
}

func (test *TestSuite) TestMarshal() {
	test.Config.SetDefault("port", 1313)
	test.Config.Set("name", "Steve")

	type configStruct struct {
		Port int
		Name string
	}

	var C configStruct

	err := test.Config.Marshal(&C)
	if err != nil {
		test.T().Fatalf("unable to decode into struct, %v", err)
	}

	assert.Equal(test.T(), &C, &configStruct{Name: "Steve", Port: 1313})

	test.Config.Set("port", 1234)
	err = test.Config.Marshal(&C)
	if err != nil {
		test.T().Fatalf("unable to decode into struct, %v", err)
	}
	assert.Equal(test.T(), &C, &configStruct{Name: "Steve", Port: 1234})
}
