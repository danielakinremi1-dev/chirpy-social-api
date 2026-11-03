package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	full := map[string]string{
		"DB_URL":    "postgres://localhost/chirpy",
		"SECRET":    "secret",
		"POLKA_KEY": "polka",
		"PLATFORM":  "dev",
		"PORT":      "9000",
	}

	tests := []struct {
		name        string
		unset       []string
		want        Config
		wantMissing string
	}{
		{
			name: "All set",
			want: Config{DBURL: "postgres://localhost/chirpy", JWTSecret: "secret", PolkaKey: "polka", Platform: "dev", Port: "9000"},
		},
		{
			name:  "Default port",
			unset: []string{"PORT", "PLATFORM"},
			want:  Config{DBURL: "postgres://localhost/chirpy", JWTSecret: "secret", PolkaKey: "polka", Port: "8080"},
		},
		{
			name:        "Missing required",
			unset:       []string{"DB_URL", "POLKA_KEY"},
			wantMissing: "DB_URL, POLKA_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]string{}
			for k, v := range full {
				env[k] = v
			}
			for _, k := range tt.unset {
				delete(env, k)
			}

			got, err := Load(func(key string) string { return env[key] })
			if tt.wantMissing != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantMissing) {
					t.Fatalf("Load() error = %v, want it to list %s", err, tt.wantMissing)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
