package config

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const minAPIKeyLen = 32

type Config struct {
	Server    ServerConfig
	APIKeys   []APIKey
	RateLimit RateLimitConfig
}

type APIKey struct {
	ID  string
	Key string
}

type ServerConfig struct {
	// Addr is the listen address passed to http.Server.
	Addr string
}

type RateLimitConfig struct {
	RPS   float64
	Burst int
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	apiKeys, keysErr := parseAPIKeys(getString("API_KEYS", ""))
	burst, burstErr := positiveInt("RATE_LIMIT_BURST", "100")
	rps, rpsErr := positiveFloat("RATE_LIMIT_RPS", "10")

	cfg := Config{
		Server: ServerConfig{
			Addr: getString("HTTP_ADDR", ":8080"),
		},
		APIKeys: apiKeys,
		RateLimit: RateLimitConfig{
			RPS:   rps,
			Burst: burst,
		},
	}

	if err := errors.Join(cfg.validate(), keysErr, burstErr, rpsErr); err != nil {
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

func positiveInt(key, def string) (int, error) {
	raw := getString(key, def)

	intVal, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %s is not a whole number", key, raw)
	}
	if intVal <= 0 {
		return intVal, fmt.Errorf("%s: must be greater than 0, got %d", key, intVal)
	}
	return intVal, nil
}

func positiveFloat(key, fallback string) (float64, error) {
	raw := getString(key, fallback)

	floatVal, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %s is not a number", key, raw)
	}
	if math.IsNaN(floatVal) || math.IsInf(floatVal, 0) {
		return floatVal, fmt.Errorf("%s: %g must be a finite number", key, floatVal)
	}
	if floatVal <= 0 {
		return floatVal, fmt.Errorf("%s: must be greater than 0, got %g", key, floatVal)
	}
	return floatVal, nil
}

func parseAPIKeys(raw string) ([]APIKey, error) {
	if raw == "" {
		return nil, errors.New("API_KEYS: at least one key is required")
	}

	var (
		keys     []APIKey
		errs     []error
		seenIDs  = make(map[string]bool)
		seenKeys = make(map[string]bool)
	)

	for i, entry := range strings.Split(raw, ",") {
		id, key, ok := strings.Cut(entry, ":")
		id, key = strings.TrimSpace(id), strings.TrimSpace(key)

		switch {
		case !ok || id == "" || key == "":
			errs = append(errs, fmt.Errorf("API_KEYS: entry %d must be id:key", i+1))
			continue
		case len(key) < minAPIKeyLen:
			errs = append(errs, fmt.Errorf("API_KEYS: key for %q must be at least %d characters", id, minAPIKeyLen))
		case seenIDs[id]:
			errs = append(errs, fmt.Errorf("API_KEYS: duplicate id %q", id))
		case seenKeys[key]:
			errs = append(errs, fmt.Errorf("API_KEYS: key for %q is already used by another id", id))
		}

		seenIDs[id] = true
		seenKeys[key] = true
		keys = append(keys, APIKey{ID: id, Key: key})
	}

	return keys, errors.Join(errs...)
}

func getString(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}

	return fallback
}
