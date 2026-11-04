// Package config loads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	DBURL     string
	JWTSecret string
	PolkaKey  string
	Platform  string
	Port      string
	// MigrateOnStart applies pending database migrations before serving.
	MigrateOnStart bool
	// TrustedProxies is how many reverse proxies sit in front of the server.
	// When it's above 0, the client IP comes from the X-Forwarded-For entry
	// the outermost proxy appended, instead of from the connection.
	TrustedProxies int
}

// Load reads the configuration using getenv, which is normally os.Getenv.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DBURL:     getenv("DB_URL"),
		JWTSecret: getenv("SECRET"),
		PolkaKey:  getenv("POLKA_KEY"),
		Platform:  getenv("PLATFORM"),
		Port:      getenv("PORT"),

		MigrateOnStart: getenv("MIGRATE_ON_START") == "true",
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	required := []struct{ key, value string }{
		{"DB_URL", cfg.DBURL},
		{"SECRET", cfg.JWTSecret},
		{"POLKA_KEY", cfg.PolkaKey},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.key)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if v := getenv("TRUST_PROXY"); v != "" && v != "false" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("TRUST_PROXY must be the number of reverse proxies in front of the server, got %q", v)
		}
		cfg.TrustedProxies = n
	}

	// PLATFORM=dev enables POST /admin/reset, which deletes every user. Render
	// always sets RENDER=true, so refuse to start there rather than expose it.
	if cfg.Platform == "dev" && getenv("RENDER") == "true" {
		return Config{}, errors.New("PLATFORM=dev is not allowed on Render: it enables POST /admin/reset")
	}

	return cfg, nil
}
