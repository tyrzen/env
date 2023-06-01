package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func LoadVars(pth string) error {
	if pth == "" {
		pth = ".env"
	}
	env, err := os.Open(pth)
	if err != nil {
		return fmt.Errorf("opening environment file: %w", err)
	}

	defer func() {
		if err := env.Close(); err != nil {
			log.Printf("closing environment file: %v", err)
		}
	}()

	buf := bufio.NewScanner(env)
	buf.Split(bufio.ScanLines)

	for buf.Scan() {
		if keyVal := strings.Split(buf.Text(), "="); len(keyVal) > 1 {
			if err := os.Setenv(keyVal[0], keyVal[1]); err != nil {
				return fmt.Errorf("setting environment variable: %w", err)
			}
		}
	}

	return nil
}

func ParseVars[T any](pth, tag string) (*T, error) {
	if tag == "" {
		tag = "env"
	}
	if err := LoadVars(pth); err != nil {
		return nil, err
	}

	dst := new(T)
	val := reflect.ValueOf(dst).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)

		if field.Type.Kind() == reflect.Struct {
			for j := 0; j < field.Type.NumField(); j++ {
				envTag := field.Type.Field(j).Tag.Get(tag)
				if envTag != "" {
					envVal := os.Getenv(envTag)
					if envVal == "" {
						continue
					}

					switch field.Type.Field(j).Type.Kind() {
					case reflect.Int:
						integer, err := strconv.Atoi(envVal)
						if err != nil {
							return nil, err
						}
						val.Field(i).Field(j).SetInt(int64(integer))

					case reflect.Float64:
						float, err := strconv.ParseFloat(envVal, 64)
						if err != nil {
							return nil, err
						}
						val.Field(i).Field(j).SetFloat(float)

					case reflect.TypeOf(time.Duration(0)).Kind():
						duration, err := time.ParseDuration(envVal)
						if err != nil {
							return nil, err
						}
						val.Field(i).Field(j).Set(reflect.ValueOf(duration))

					default:
						val.Field(i).Field(j).SetString(envVal)
					}
				}
			}
		}
	}

	return dst, nil
}
