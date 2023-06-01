package env

import (
	"os"
	"reflect"
	"testing"
	"time"
)

// Config is a struct for testing ParseVars function.
type Config struct {
	Database struct {
		Port     int           `env:"DB_PORT"`
		Timeout  time.Duration `env:"DB_TIMEOUT"`
		Password string        `env:"DB_PASSWORD"`
	}
}

// TestLoadVars tests LoadVars function.
func TestLoadVars(t *testing.T) {
	err := os.WriteFile("example.env", []byte("KEY=VALUE"), 0600)
	if err != nil {
		t.Fatalf("writing environment file: %v", err)
	}
	defer os.Remove("example.env")

	err = LoadVars("example.env")
	if err != nil {
		t.Errorf("LoadVars() error = %v", err)
		return
	}

	got := os.Getenv("KEY")
	want := "VALUE"
	if got != want {
		t.Errorf("os.Getenv() = %v, want %v", got, want)
	}
}

// TestParseVars tests ParseVars function.
func TestParseVars(t *testing.T) {
	err := os.WriteFile("example.env", []byte("DB_PORT=3306\nDB_TIMEOUT=5s\nDB_PASSWORD=secret"), 0600)
	if err != nil {
		t.Fatalf("writing environment file: %v", err)
	}
	defer os.Remove("example.env")

	got, err := ParseVars[Config]("example.env", "env")
	if err != nil {
		t.Errorf("ParseVars[Config]() error = %v", err)
		return
	}

	want := &Config{
		Database: struct {
			Port     int           `env:"DB_PORT"`
			Timeout  time.Duration `env:"DB_TIMEOUT"`
			Password string        `env:"DB_PASSWORD"`
		}(struct {
			Port     int
			Timeout  time.Duration
			Password string
		}{
			Port:     3306,
			Timeout:  5 * time.Second,
			Password: "secret",
		}),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseVars[Config]() = %v, want %v", got, want)
	}
}
