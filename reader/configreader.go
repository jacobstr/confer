package reader

import(
	"path/filepath"
	"io"
	// "github.com/spf13/cast"
	"io/ioutil"
	"bytes"
	"encoding/json"
	"gopkg.in/yaml.v1"
	"github.com/BurntSushi/toml"
	"github.com/jacobstr/viper/errors"
	jww "github.com/spf13/jwalterweatherman"
)

type ConfigFormat string

const (
	FormatYAML ConfigFormat = "yaml"
	FormatJSON ConfigFormat = "json"
	FormatTOML ConfigFormat = "toml"
)

type ConfigReader struct {
	Format string
	reader io.Reader
}

// Retuns the configuration data into a generic object for for us.
func (cr *ConfigReader) Export() (interface{}, error) {
	return cr.ExportAs(struct{}{})
}

// Provide a struct to marshall this config reader's data into.  This allows some
// usage by those who might want to take advantage of the json decoders custom
// tags e.g.
//
//	struct Database {
//		Host string `json:"host"`
//		Port int		`json:"port"`
//	}
//
// Though we tend to convert to stringmaps anyway.
func (cr *ConfigReader) ExportAs(template struct{}) (interface{}, error) {
	var config interface{}
	buf := new(bytes.Buffer)
	buf.ReadFrom(cr.reader)

	switch cr.Format {
	case "yaml":
		if err := yaml.Unmarshal(buf.Bytes(), &config); err != nil {
			jww.ERROR.Fatalf("Error parsing config: %s", err)
		}

	case "json":
		if err := json.Unmarshal(buf.Bytes(), &config); err != nil {
			jww.ERROR.Fatalf("Error parsing config: %s", err)
		}

	case "toml":
		if _, err := toml.Decode(buf.String(), &config); err != nil {
			jww.ERROR.Fatalf("Error parsing config: %s", err)
		}
	default:
		return nil, err.UnsupportedConfigError(cr.Format)
	}

	return config, nil
}

func Readfile(path string) (interface{}, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		jww.DEBUG.Println("Error reading config file:", err)
		return nil, err
	}

	reader := bytes.NewReader(file)

	cr := &ConfigReader{ Format: getConfigType(path), reader: reader }
	return cr.Export()
}

func Readbytes(data []byte, format string) (interface{}, error) {
	cr := ConfigReader{
		Format: format,
		reader: bytes.NewReader(data),
	}

	return cr.Export()
}

func getConfigType(path string) string {
	ext := filepath.Ext(path)
	if len(ext) > 1 {
		return ext[1:]
	} else {
		return ""
	}
}

