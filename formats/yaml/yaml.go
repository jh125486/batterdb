package yaml

import (
	"io"

	"github.com/danielgtaylor/huma/v2"
	"gopkg.in/yaml.v3"
)

// DefaultYAMLFormat is the default YAML formatter that can be set in the API's
// `Config.Formats` map. This is usually not needed as importing this package
// automatically adds the text format to the default formats.
//
//	config := huma.Config{}
//	config.Formats = map[string]huma.Format{
//		"application/yaml": huma.DefaultYAMLFormat,
//		"yaml":             huma.DefaultYAMLFormat,
//	}
var DefaultYAMLFormat = huma.Format{
	Marshal: func(w io.Writer, v any) error {
		return yaml.NewEncoder(w).Encode(v)
	},
	Unmarshal: yaml.Unmarshal,
}

func init() {
	huma.DefaultFormats["application/yaml"] = DefaultYAMLFormat
	huma.DefaultFormats["yaml"] = DefaultYAMLFormat
}
