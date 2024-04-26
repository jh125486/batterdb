package yaml_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/formats/yaml"
)

func TestDefaultYAMLFormat(t *testing.T) {
	t.Parallel()
	format := yaml.DefaultYAMLFormat
	tests := []struct {
		name     string
		v        any
		expected string
	}{
		{
			name:     "simple map",
			v:        map[string]any{"key": "value"},
			expected: "key: value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var bb bytes.Buffer
			require.NoError(t, format.Marshal(&bb, tt.v))
			assert.Equal(t, tt.expected, bb.String())
			var v2 any
			require.NoError(t, format.Unmarshal(bb.Bytes(), &v2))
			require.Equal(t, tt.v, v2)
		})
	}
}
