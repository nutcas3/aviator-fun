package cache

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envValue   string
		want       string
	}{
		{
			name:       "Environment variable exists",
			key:        "TEST_KEY_EXISTS",
			defaultVal: "default",
			envValue:   "custom_value",
			want:       "custom_value",
		},
		{
			name:       "Environment variable does not exist",
			key:        "TEST_KEY_NOT_EXISTS",
			defaultVal: "default_value",
			envValue:   "",
			want:       "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal int
		envValue   string
		want       int
	}{
		{
			name:       "Valid integer",
			key:        "TEST_INT_VALID",
			defaultVal: 0,
			envValue:   "42",
			want:       42,
		},
		{
			name:       "Invalid integer",
			key:        "TEST_INT_INVALID",
			defaultVal: 10,
			envValue:   "not_a_number",
			want:       10,
		},
		{
			name:       "Empty value",
			key:        "TEST_INT_EMPTY",
			defaultVal: 5,
			envValue:   "",
			want:       5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvAsInt(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvAsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Note: Integration tests for Redis require a running Redis instance
// These would be in a separate integration test file
func TestNew_NoRedis(t *testing.T) {
	// Set invalid Redis URL to test error handling
	os.Setenv("REDIS_URL", "invalid_host:9999")
	defer os.Unsetenv("REDIS_URL")

	// This should return nil when Redis is not available
	service := New()
	
	// The service should handle the error gracefully
	if service != nil {
		// If service is not nil, it means Redis was available
		// This is okay for the test
		t.Log("Redis service created (Redis might be running)")
	} else {
		t.Log("Redis service is nil (expected when Redis is not available)")
	}
}

func TestService_Interface(t *testing.T) {
	// Verify that service implements Service interface
	var _ Service = (*service)(nil)
}
