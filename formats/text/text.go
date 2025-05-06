package text

import (
	"encoding"
	"fmt"
	"io"

	"github.com/danielgtaylor/huma/v2"
)

// DefaultTextFormat is the default text formatter that can be set in the API's
// `Config.Formats` map. This is usually not needed as importing this package
// automatically adds the text format to the default formats.
//
//	config := huma.Config{}
//	config.Formats = map[string]huma.Format{
//		"plain/text": huma.DefaultTextFormat,
//		"text":       huma.DefaultTextFormat,
//	}
func DefaultTextFormat() huma.Format {
	return huma.Format{
		Marshal: func(w io.Writer, v any) error {
			if m, ok := v.(encoding.TextMarshaler); ok {
				b, err := m.MarshalText()
				if err != nil {
					return err
				}
				_, err = w.Write(b)

				return err
			}
			_, err := fmt.Fprint(w, v)

			return err
		},
		Unmarshal: func(data []byte, v any) error {
			if m, ok := v.(encoding.TextUnmarshaler); ok {
				return m.UnmarshalText(data)
			}
			return huma.Error501NotImplemented("text format not supported")
		},
	}
}
