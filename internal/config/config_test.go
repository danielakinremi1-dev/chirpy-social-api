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

		"MIGRATE_ON_START": "true",
		"TRUST_PROXY":      "true",
	}

	tests := []struct {
		name    string
		unset   []string
		extra   map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "All set",
			want: Config{DBURL: "postgres://localhost/chirpy", JWTSecret: "secret", PolkaKey: "polka", Platform: "dev", Port: "9000", MigrateOnStart: true, TrustProxy: true},
		},
		{
			name:  "Default port",
			unset: []string{"PORT", "PLATFORM", "MIGRATE_ON_START", "TRUST_PROXY"},
			want:  Config{DBURL: "postgres://localhost/chirpy", JWTSecret: "secret", PolkaKey: "polka", Port: "8080"},
		},
		{
			name:    "Missing required",
			unset:   []string{"DB_URL", "POLKA_KEY"},
			wantErr: "DB_URL, POLKA_KEY",
		},
		{
			name:    "Dev platform on Render",
			extra:   map[string]string{"RENDER": "true"},
			wantErr: "PLATFORM=dev is not allowed on Render",
		},
		{
			name:  "Production platform on Render",
			extra: map[string]string{"RENDER": "true", "PLATFORM": "production"},
			want:  Config{DBURL: "postgres://localhost/chirpy", JWTSecret: "secret", PolkaKey: "polka", Platform: "production", Port: "9000", MigrateOnStart: true, TrustProxy: true},
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
			for k, v := range tt.extra {
				env[k] = v
			}

			got, err := Load(func(key string) string { return env[key] })
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load() error = %v, want it to contain %q", err, tt.wantErr)
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
