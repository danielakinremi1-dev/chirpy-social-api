// Package config loads the server configuration from environment variables.
package config

import (
	"fmt"
	"strings"
)

type Config struct {
	DBURL     string
	JWTSecret string
	PolkaKey  string
	Platform  string
}

// Load reads the configuration using getenv, which is normally os.Getenv.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DBURL:     getenv("DB_URL"),
		JWTSecret: getenv("SECRET"),
		PolkaKey:  getenv("POLKA_KEY"),
		Platform:  getenv("PLATFORM"),
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

	return cfg, nil
}
