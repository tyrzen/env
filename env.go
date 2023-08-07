/*
Package env provides functions to load environment variables from a .env file into the system's
environment variables, and to parse them into a given struct.

It supports several tags:
`default`- provides the default variable value.
`env` - provides variable name that allows overriding the default variable.

If tags are not provided, field names in the struct are automatically transformed
to the conventional SNAKE_CASE with parent struct prefix to match environment variable.
In case, a variable or the given struct field is not found,
and default value is not provided; an error is returned.
*/
package env

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	tagEnv     = "env"
	tagDefault = "default"
)

// ParseTo loads the environment variables from .env file paths
// and fills the configuration struct with the values.
func ParseTo(dst any, envPaths ...string) error {
	if envPaths == nil {
		envPaths = []string{".env"}
	}

	for i := range envPaths {
		if err := Load(envPaths[i]); err != nil {
			return err
		}
	}

	if err := parseTo(dst, ""); err != nil {
		return err
	}

	return nil
}

// Load loads the environment variables from a .env file into the system's.
func Load(pth string) error {
	env, err := os.Open(pth)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("opening dotenv file: %w", err)
	}

	defer func() {
		if err := env.Close(); err != nil {
			log.Printf("closing dotenv file: %v", err)
		}
	}()

	buf := bufio.NewScanner(env)
	for buf.Scan() {
		line := buf.Text()
		if line == "" {
			continue
		}

		key, val := parseLine(line)

		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("setting %s[%s]: %w", key, val, err)
		}
	}

	if err := buf.Err(); err != nil {
		return fmt.Errorf("reading env file: %w", err)
	}

	return nil
}

// parseTo fills the struct fields
// assigning the values from sources.
func parseTo(dst any, prefix string) error {
	v := reflect.ValueOf(dst).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		fieldName := prefix + fieldType.Name
		if fieldType.Type.Kind() == reflect.Struct {
			// recursive call is needed for nested structs.
			if err := parseTo(fieldVal.Addr().Interface(), fieldName); err != nil {
				return err
			}

			continue
		}

		val := getFieldValue(fieldType, fieldName)
		if val == "" && prefix != "" {
			return fmt.Errorf("no value for field: %s", fieldType.Name)
		}

		if err := setFieldValue(fieldType.Type, fieldVal, val); err != nil {
			return err
		}
	}

	return nil
}

// getFieldValue gets the value for a field from different sources:
// the environment variables or the `default` struct tag values.
func getFieldValue(fieldType reflect.StructField, fieldName string) string {
	envTag := fieldType.Tag.Get(tagEnv)
	if val, ok := os.LookupEnv(envTag); ok {
		return val
	}

	if val, ok := os.LookupEnv(camelToSnake(fieldName)); ok {
		return val
	}

	val := fieldType.Tag.Get(tagDefault)

	return val
}

// parseLine parses a have from the .env file or value from os.Environ().
func parseLine(line string) (string, string) {
	i := strings.Index(line, "=")
	if i <= 0 {
		return "", ""
	}

	return line[:i], line[i+1:]
}

// camelToSnake converts a CamelCase string to SNAKE_CASE.
func camelToSnake(s string) string {
	var (
		parts []string
		start int
	)

	for i, r := range s {
		if unicode.IsUpper(r) && i != 0 {
			parts = append(parts, s[start:i])
			start = i
		}
	}

	parts = append(parts, s[start:])

	for i, p := range parts {
		parts[i] = strings.ToUpper(p)
	}

	return strings.Join(parts, "_")
}

// setFieldValue sets the value for a struct field according to a field type.
//
//nolint:cyclop
func setFieldValue(fieldType reflect.Type, fieldVal reflect.Value, val string) error {
	switch fieldType.Kind() {
	case reflect.TypeOf(time.Duration(0)).Kind():
		val, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("parsing duration: %w", err)
		}

		fieldVal.Set(reflect.ValueOf(val))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(val, 0, fieldType.Bits())
		if err != nil {
			return fmt.Errorf("parsing integer: %w", err)
		}

		fieldVal.SetInt(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(val, 0, fieldType.Bits())
		if err != nil {
			return fmt.Errorf("parsing unsigned integer: %w", err)
		}

		fieldVal.SetUint(val)

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("parsing float: %w", err)
		}

		fieldVal.SetFloat(val)

	case reflect.Bool:
		val, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("parsing bool: %w", err)
		}

		fieldVal.SetBool(val)

	case reflect.String:
		fieldVal.SetString(val)

	case reflect.Slice:
		if fieldType.Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice kind: %s", fieldType.Elem().Kind())
		}

		fieldVal.Set(reflect.ValueOf(strings.Split(val, ",")))

	default:
		return fmt.Errorf("unsupported field type: %s", fieldType.Name())
	}

	return nil
}
