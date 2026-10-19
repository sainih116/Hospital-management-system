package config

import "testing"

func TestValidateRejectsEmptyJWTSecretsRegardlessOfEnv(t *testing.T) {
	cases := []struct {
		name          string
		env           string
		accessSecret  string
		refreshSecret string
		wantErr       bool
	}{
		{"missing both in development", "development", "", "", true},
		{"missing access secret only", "development", "", "set", true},
		{"missing refresh secret only", "development", "set", "", true},
		{"missing both with no APP_ENV set", "", "", "", true},
		{"missing both in production", "production", "", "", true},
		{"both set in development", "development", "set", "set", false},
		{"both set in production", "production", "set", "set", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Env: tc.env,
				JWT: JWTConfig{AccessSecret: tc.accessSecret, RefreshSecret: tc.refreshSecret},
			}

			err := cfg.validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected validate() to reject an empty JWT secret, got nil error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected validate() to accept non-empty JWT secrets, got: %v", err)
			}
		})
	}
}
