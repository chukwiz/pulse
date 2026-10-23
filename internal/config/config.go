package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	//Addr is the listen address passed to http.Server
	Addr string
}

func Load() (Config, error) {
	err := godotenv.Load()

	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("load .env: %w", err)
		}
	}
	cfg := Config{
		Server: ServerConfig{
			Addr: getString("HTTP_ADDR", ":8080"),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c Config) validate() error {
	var errs []error

	if err := validateListenAddr(c.Server.Addr); err != nil {
		errs = append(errs, fmt.Errorf("HTTP_ADDR: %w", err))
	}

	return errors.Join(errs...)
}

func validateListenAddr(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%q is not a valid listen address: %w", addr, err)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("%q has an invalid port %q", addr, port)
	}
	return nil
}

func getString(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}

	return fallback
}
