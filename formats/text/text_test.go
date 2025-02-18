package text_test

import (
	"bytes"
	"fmt"
	"io"
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

type BadMarshaler struct{}

func (b BadMarshaler) MarshalText() ([]byte, error) { return nil, io.EOF }

type BadWriter struct {
	io.Reader
}

func (b BadWriter) Write(_ []byte) (int, error) { return 0, io.EOF }

func TestDefaultTextFormat_Marshal(t *testing.T) {
	t.Parallel()
	format := text.DefaultTextFormat()
	type args struct {
		w io.ReadWriter
		v any
	}
	tests := []struct {
		name     string
		args     args
		wantErr  require.ErrorAssertionFunc
		expected string
	}{
		{
			name: "valid text marshaler",
			args: args{
				w: new(bytes.Buffer),
				v: &TestStruct{
					V1: "key",
					V2: 666,
				},
			},
			wantErr:  require.NoError,
			expected: "key/666",
		},
		{
			name: "not a text marshaler",
			args: args{
				w: new(bytes.Buffer),
				v: 666,
			},
			wantErr:  require.NoError,
			expected: "666",
		},
		{
			name: "bad marshaler",
			args: args{
				w: new(bytes.Buffer),
				v: &BadMarshaler{},
			},
			wantErr: require.Error,
		},
		{
			name: "writer errors",
			args: args{
				w: BadWriter{new(bytes.Buffer)},
				v: 5,
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := format.Marshal(tt.args.w, tt.args.v)
			if tt.wantErr(t, err); err != nil {
				return
			}
			b, err := io.ReadAll(tt.args.w)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(b))
		})
	}
}

func TestDefaultTextFormat_Unmarshal(t *testing.T) {
	t.Parallel()
	format := text.DefaultTextFormat()
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name     string
		obj      any
		args     args
		wantErr  require.ErrorAssertionFunc
		expected any
	}{
		{
			name: "valid",
			obj:  &TestStruct{},
			args: args{
				bytes: []byte("key/666"),
			},
			wantErr: require.NoError,
			expected: &TestStruct{
				V1: "key",
				V2: 666,
			},
		},
		{
			name: "invalid bytes",
			obj:  &TestStruct{},
			args: args{
				bytes: []byte("key"),
			},
			wantErr: require.Error,
		},
		{
			name: "not an unmarshaler",
			obj:  new(int),
			args: args{
				bytes: []byte("key"),
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := format.Unmarshal(tt.args.bytes, tt.obj)
			if tt.wantErr(t, err); err != nil {
				return
			}
			require.Equal(t, tt.expected, tt.obj)
		})
	}
}
