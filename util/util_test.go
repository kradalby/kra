package util

import (
	"os"
	"testing"
)

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		setEnv   bool
		fallback string
		want     string
	}{
		{
			name:     "returns fallback when env not set",
			key:      "TEST_GETENVSTRING_UNSET",
			setEnv:   false,
			fallback: "default",
			want:     "default",
		},
		{
			name:     "returns env value when set",
			key:      "TEST_GETENVSTRING_SET",
			envVal:   "custom",
			setEnv:   true,
			fallback: "default",
			want:     "custom",
		},
		{
			name:     "returns empty string when env set to empty",
			key:      "TEST_GETENVSTRING_EMPTY",
			envVal:   "",
			setEnv:   true,
			fallback: "default",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				t.Setenv(tt.key, tt.envVal)
			}

			got := GetEnvString(tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("GetEnvString(%q, %q) = %q, want %q", tt.key, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		setEnv   bool
		fallback bool
		want     bool
	}{
		{
			name:     "returns fallback when env not set",
			key:      "TEST_GETENVBOOL_UNSET",
			setEnv:   false,
			fallback: false,
			want:     false,
		},
		{
			name:     "returns true for 'true'",
			key:      "TEST_GETENVBOOL_TRUE",
			envVal:   "true",
			setEnv:   true,
			fallback: false,
			want:     true,
		},
		{
			name:     "returns true for '1'",
			key:      "TEST_GETENVBOOL_ONE",
			envVal:   "1",
			setEnv:   true,
			fallback: false,
			want:     true,
		},
		{
			name:     "returns false for 'false'",
			key:      "TEST_GETENVBOOL_FALSE",
			envVal:   "false",
			setEnv:   true,
			fallback: true,
			want:     false,
		},
		{
			name:     "returns false for '0'",
			key:      "TEST_GETENVBOOL_ZERO",
			envVal:   "0",
			setEnv:   true,
			fallback: true,
			want:     false,
		},
		{
			name:     "returns fallback for invalid value",
			key:      "TEST_GETENVBOOL_INVALID",
			envVal:   "not-a-bool",
			setEnv:   true,
			fallback: true,
			want:     true,
		},
		{
			name:     "returns fallback for empty string",
			key:      "TEST_GETENVBOOL_EMPTY",
			envVal:   "",
			setEnv:   true,
			fallback: false,
			want:     false,
		},
		{
			name:     "returns true for 'TRUE'",
			key:      "TEST_GETENVBOOL_UPPER",
			envVal:   "TRUE",
			setEnv:   true,
			fallback: false,
			want:     true,
		},
		{
			name:     "returns true for 'True'",
			key:      "TEST_GETENVBOOL_MIXED",
			envVal:   "True",
			setEnv:   true,
			fallback: false,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				t.Setenv(tt.key, tt.envVal)
			}

			got := GetEnvBool(tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("GetEnvBool(%q, %v) = %v, want %v", tt.key, tt.fallback, got, tt.want)
			}
		})
	}
}
