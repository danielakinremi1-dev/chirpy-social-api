// Package config loads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
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
	// TrustProxy takes the client IP from the X-Forwarded-For entry appended
	// by the reverse proxy in front of the server, instead of the connection.
	TrustProxy bool
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
		TrustProxy:     getenv("TRUST_PROXY") == "true",
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

	// PLATFORM=dev enables POST /admin/reset, which deletes every user. Render
	// always sets RENDER=true, so refuse to start there rather than expose it.
	if cfg.Platform == "dev" && getenv("RENDER") == "true" {
		return Config{}, errors.New("PLATFORM=dev is not allowed on Render: it enables POST /admin/reset")
	}

	return cfg, nil
}
