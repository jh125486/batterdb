package text_test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/formats/text"
)

type TestStruct struct {
	V1 string
	V2 int
}

func (t TestStruct) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s/%d", t.V1, t.V2)), nil
}
func (t *TestStruct) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), "/")
	if len(parts) != 2 {
		return fmt.Errorf("expected input to be in format 'string/int', got %s", string(text))
	}
	t.V1 = parts[0]

	v2, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("error parsing integer: %w", err)
	}
	t.V2 = v2

	return nil
}

func TestDefaultTextFormat(t *testing.T) {
	t.Parallel()
	format := text.DefaultTextFormat
	tests := []struct {
		name     string
		v        TestStruct
		expected string
	}{
		{
			name: "struct",
			v: TestStruct{
				V1: "key",
				V2: 666,
			},
			expected: "key/666",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var bb bytes.Buffer
			require.NoError(t, format.Marshal(&bb, tt.v))
			assert.Equal(t, tt.expected, bb.String())
			var v2 TestStruct
			require.NoError(t, format.Unmarshal(bb.Bytes(), &v2))
			require.Equal(t, tt.v, v2)
		})
	}
}
