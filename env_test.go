package env

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTo(t *testing.T) {
	type config struct {
		Home    string `env:"HOME"`
		Default string `default:"default"`
		Empty   int
		Nested  struct{ Value string }
	}

	tests := map[string]struct {
		vars    map[string]string
		want    *config
		wantErr error
	}{
		"All environment variables present": {
			vars:    map[string]string{"HOME": "/home/test", "DEFAULT": "default", "EMPTY": "10", "NESTED_VALUE": "nested"},
			want:    &config{Home: "/home/test", Default: "default", Empty: 10, Nested: struct{ Value string }{"nested"}},
			wantErr: nil,
		},
		"No environment variables, except default present": {
			vars:    nil,
			want:    new(config),
			wantErr: fmt.Errorf("no environment variables"),
		},
		"Some environment variables present": {
			vars:    map[string]string{"HOME": "/home/test", "EMPTY": "10"},
			want:    &config{Home: "/home/test", Empty: 10, Default: "default"},
			wantErr: errors.New("no value for field: Value"),
		},
		"Environment variable set to empty string": {
			vars:    map[string]string{"HOME": "", "DEFAULT": "", "EMPTY": "0", "NESTED_VALUE": ""},
			want:    &config{Home: "", Default: "", Empty: 0},
			wantErr: errors.New("no environment variables"),
		},
		"Environment variable set to invalid value": {
			vars:    map[string]string{"HOME": "/home/test", "DEFAULT": "default", "EMPTY": "invalid"},
			want:    nil,
			wantErr: fmt.Errorf("parsing integer: strconv.ParseInt: parsing \"invalid\": invalid syntax"),
		},
		"Nested struct environment variable set": {
			vars:    map[string]string{"HOME": "/home/test", "DEFAULT": "default", "EMPTY": "10", "NESTED_VALUE": "nested"},
			want:    &config{Home: "/home/test", Default: "default", Empty: 10, Nested: struct{ Value string }{"nested"}},
			wantErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			teardown, err := setupEnv(t, tc.vars)
			require.NoError(t, err)
			defer teardown()

			var cfg config
			err = parseTo(&cfg, "")

			if tc.wantErr != nil {
				require.Error(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, &cfg)
		})
	}
}

func TestLoadEnv(t *testing.T) {
	tests := map[string]struct {
		vars    map[string]string
		wantErr error
	}{
		"Valid environment file": {
			vars:    map[string]string{"HOME": "/home/test", "DEFAULT": "default", "EMPTY": "10", "NESTED_VALUE": "nested"},
			wantErr: nil,
		},
		"Empty variables": {
			vars:    map[string]string{"": ""},
			wantErr: errors.New("setting []: setenv: The parameter is incorrect"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file, teardown, err := setupEnvFile(t, tc.vars)
			require.NoError(t, err)
			defer teardown()

			teardown, err = setupEnv(t, tc.vars)
			if tc.wantErr != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			defer teardown()

			err = Load(file.Name())
			if tc.wantErr != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			for k, v := range tc.vars {
				require.Equal(t, v, os.Getenv(k))
			}
		})
	}
}

func TestSetFieldValue(t *testing.T) {
	tests := map[string]struct {
		field   interface{}
		have    string
		want    interface{}
		wantErr error
	}{
		"Duration":         {time.Duration(0), "1h", time.Hour, nil},
		"Invalid Duration": {time.Duration(0), "invalid", nil, errors.New("parsing duration: time: invalid duration \"invalid\"")},
		"Int":              {0, "123", 123, nil},
		"Invalid Int":      {0, "invalid", nil, errors.New("parsing integer: strconv.ParseInt: parsing \"invalid\": invalid syntax")},
		"Float":            {0.0, "1.23", 1.23, nil},
		"Invalid Float":    {0.0, "invalid", nil, errors.New("parsing float: strconv.ParseFloat: parsing \"invalid\": invalid syntax")},
		"Bool":             {false, "true", true, nil},
		"Invalid Bool":     {false, "invalid", nil, errors.New("parsing bool: strconv.ParseBool: parsing \"invalid\": invalid syntax")},
		"String":           {"", "test", "test", nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fieldVal := reflect.New(reflect.TypeOf(tc.field)).Elem()
			err := setFieldValue(reflect.TypeOf(tc.field), fieldVal, tc.have)

			if tc.wantErr != nil {
				require.Error(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, fieldVal.Interface())
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name string
		have string
		want string
	}{
		{name: "Pascal case", have: "PascalCase", want: "PASCAL_CASE"},
		{name: "Camel case", have: "camelCase", want: "CAMEL_CASE"},
		{name: "Snake case", have: "snake_case", want: "SNAKE_CASE"},
		{name: "Lowercase", have: "lowercase", want: "LOWERCASE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := camelToSnake(tc.have)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestParseLine(t *testing.T) {
	tests := map[string]struct {
		have      string
		wantKey   string
		wantValue string
	}{
		"Line with key-value pair":         {have: "KEY=value", wantKey: "KEY", wantValue: "value"},
		"Line with empty key":              {have: "EMPTY_KEY=", wantKey: "EMPTY_KEY", wantValue: ""},
		"Line with empty value":            {have: "=EMPTY_VALUE", wantKey: "", wantValue: ""},
		"Line without equal sign":          {have: "NO_EQUAL_SIGN", wantKey: "", wantValue: ""},
		"Another have with key-value pair": {have: "ANOTHER_CASE=another_value", wantKey: "ANOTHER_CASE", wantValue: "another_value"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			key, value := parseLine(tc.have)
			require.Equal(t, tc.wantKey, key)
			require.Equal(t, tc.wantValue, value)
		})
	}
}

func setupEnv(t *testing.T, vars map[string]string) (func(), error) {
	t.Helper()

	if vars == nil {
		return func() {}, nil
	}

	for k, v := range vars {
		if err := os.Setenv(k, v); err != nil {
			return func() {}, fmt.Errorf("setupEnv: setting %s[%s]: %v", k, v, err)
		}
	}

	teardown := func() {
		for k := range vars {
			if err := os.Unsetenv(k); err != nil {
				t.Errorf("setEnv: unsetting %s: %v", k, err)
			}
		}
	}

	return teardown, nil
}

func setupEnvFile(t *testing.T, vars map[string]string) (*os.File, func(), error) {
	t.Helper()

	file, err := os.CreateTemp(os.TempDir(), "test.env")
	if err != nil {
		return nil, func() {}, err
	}

	if vars == nil {
		return file, func() {}, nil
	}

	for key, value := range vars {
		_, err := file.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			return nil, nil, err
		}
	}

	teardown := func() {
		err := file.Close()
		if err != nil {
			t.Error("createEnvFile: close:", err)
		}

		err = os.RemoveAll(file.Name())
		if err != nil {
			t.Error("createEnvFile: teardown:", err)
		}
	}

	return file, teardown, nil
}
