package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateListenAddr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		addr    string
		wantsErr bool
	}{
		{name: "port only", addr: ":8080"},
		{name: "host and port", addr: "localhost:8080"},
		{name: "missing colon", addr: "8080", wantsErr: true},
		{name: "empty port", addr: ":", wantsErr: true},
		{name: "port upper limit", addr: ":65535"},
		{name: "port out of range", addr: ":65536", wantsErr: true},
		{name: "port not a number", addr: ":abc", wantsErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateListenAddr(tc.addr)

			if tc.wantsErr {
				if err == nil {
					t.Fatalf("validateListenAddr(%q) = nil, wants an error", tc.addr)
				}
				return
			}

			if err != nil {
				t.Fatalf("validateListenAddr(%q) unexpected error, got %v, want nil", tc.addr, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name     string
		env      string
		want     string
		wantsErr bool
	}{
		{name: "empty uses fallback", env: "", want: ":8080"},
		{name: "whitespace uses fallback", env: "   ", want: ":8080"},
		{name: "surrounding spaces trimmed", env: "  :9000  ", want: ":9000"},
		{name: "valid value", env: ":9000", want: ":9000"},
		{name: "invalid value names key", env: ":abc", wantsErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("HTTP_ADDR", tc.env)

			cfg, err := Load()

			if tc.wantsErr {
				if err == nil {
					t.Fatalf("Load() with HTTP_ADDR=%q = %+v, want an error", tc.env, cfg)
				}

				if !strings.Contains(err.Error(), "HTTP_ADDR") {
					t.Errorf("Load() error = %q, want it to name HTTP_ADDR", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() with HTTP_ADDR=%q unexpected error: %v", tc.env, err)
			}
			if cfg.Server.Addr != tc.want {
				t.Errorf("Load() with HTTP_ADDR=%q: Server.Addr = %q, want %q", tc.env, cfg.Server.Addr, tc.want)
			}
		})
	}
}

func TestLoadDotEnv(t *testing.T) {
	cases := []struct {
		name    string
		dotenv  string // file contents; "" means no .env file
		env     string // "" means HTTP_ADDR is unset
		want    string
		wantsErr bool
	}{
		{name: "no file is fine", want: ":8080"},
		{name: "file value used", dotenv: "HTTP_ADDR=:7000\n", want: ":7000"},
		{name: "real env wins over file", dotenv: "HTTP_ADDR=:7000\n", env: ":9000", want: ":9000"},
		{name: "malformed file is an error", dotenv: "HTTP_ADDR=\":7000\n", wantsErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.dotenv != "" {
				if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(tc.dotenv), 0o600); err != nil {
					t.Fatalf("write .env: %v", err)
				}
			}
			t.Chdir(dir)

			// t.Setenv registers a cleanup that restores the original value.
			// godotenv never overrides a variable that is set (even to ""),
			// so unset it afterwards to let the file's value through. The
			// cleanup still removes whatever godotenv writes.
			t.Setenv("HTTP_ADDR", tc.env)
			if tc.env == "" {
				os.Unsetenv("HTTP_ADDR")
			}

			cfg, err := Load()

			if tc.wantsErr {
				if err == nil {
					t.Fatalf("Load() with .env %q = %+v, want an error", tc.dotenv, cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() with .env %q unexpected error: %v", tc.dotenv, err)
			}
			if cfg.Server.Addr != tc.want {
				t.Errorf("Load() with .env %q: Server.Addr = %q, want %q", tc.dotenv, cfg.Server.Addr, tc.want)
			}
		})
	}
}
