package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != "3000" {
		t.Errorf("Expected Port '3000', got '%s'", cfg.Port)
	}

	if cfg.DatabasePath != "./data/bot.db" {
		t.Errorf("Expected DatabasePath './data/bot.db', got '%s'", cfg.DatabasePath)
	}

	if cfg.Concurrency != 5 {
		t.Errorf("Expected Concurrency 5, got %d", cfg.Concurrency)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", cfg.MaxRetries)
	}

	if cfg.CallTimeout != 30 {
		t.Errorf("Expected CallTimeout 30, got %d", cfg.CallTimeout)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected int
	}{
		{
			name:     "Empty config should have 7 errors (all required fields + valid ranges)",
			config:   Config{},
			expected: 7, // bot_token, plivo_auth_id, plivo_auth_token, plivo_number, ngrok_url, concurrency, call_timeout
		},
		{
			name: "Config with all required fields",
			config: Config{
				BotToken:       "token",
				PlivoAuthID:    "id",
				PlivoAuthToken: "token",
				PlivoNumber:    "+1234567890",
				NgrokURL:       "https://example.ngrok.io",
				Concurrency:    5,  // Valid range 1-50
				CallTimeout:    30, // Valid range 10-300
			},
			expected: 0,
		},
		{
			name: "Partial config",
			config: Config{
				BotToken: "token",
			},
			expected: 6, // missing plivo fields, ngrok_url, and invalid concurrency/call_timeout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.config.Validate()
			if len(errors) != tt.expected {
				t.Errorf("Expected %d errors, got %d: %v", tt.expected, len(errors), errors)
			}
		})
	}
}
