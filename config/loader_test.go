package config

import (
	"testing"
)

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected Config
	}{
		{
			name:  "Empty config gets all defaults",
			input: Config{},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
			},
		},
		{
			name: "Partial config preserves existing values",
			input: Config{
				Port: "8080",
			},
			expected: Config{
				Port:         "8080",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
			},
		},
		{
			name: "Zero concurrency gets default",
			input: Config{
				Concurrency: 0,
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
			},
		},
		{
			name: "Negative concurrency gets default",
			input: Config{
				Concurrency: -5,
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
			},
		},
		{
			name: "Zero max retries gets default",
			input: Config{
				MaxRetries: 0,
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
			},
		},
		{
			name: "Zero call timeout gets default",
			input: Config{
				CallTimeout: 0,
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
			},
		},
		{
			name: "CallerID defaults to PlivoNumber",
			input: Config{
				PlivoNumber: "+15551234567",
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				PlivoNumber:  "+15551234567",
				CallerID:     "+15551234567",
			},
		},
		{
			name: "CallerID preserved if set",
			input: Config{
				PlivoNumber: "+15551234567",
				CallerID:    "+15559876543",
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				PlivoNumber:  "+15551234567",
				CallerID:     "+15559876543",
			},
		},
		{
			name: "AdminIDs preserved if set",
			input: Config{
				AdminIDs: []int64{123, 456, 789},
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{123, 456, 789},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a loader to use applyDefaults method
			loader := &Loader{config: &tt.input}
			loader.applyDefaults(&tt.input)
			
			if tt.input.Port != tt.expected.Port {
				t.Errorf("Port = %s, want %s", tt.input.Port, tt.expected.Port)
			}
			if tt.input.DatabasePath != tt.expected.DatabasePath {
				t.Errorf("DatabasePath = %s, want %s", tt.input.DatabasePath, tt.expected.DatabasePath)
			}
			if tt.input.LogPath != tt.expected.LogPath {
				t.Errorf("LogPath = %s, want %s", tt.input.LogPath, tt.expected.LogPath)
			}
			if tt.input.Concurrency != tt.expected.Concurrency {
				t.Errorf("Concurrency = %d, want %d", tt.input.Concurrency, tt.expected.Concurrency)
			}
			if tt.input.MaxRetries != tt.expected.MaxRetries {
				t.Errorf("MaxRetries = %d, want %d", tt.input.MaxRetries, tt.expected.MaxRetries)
			}
			if tt.input.CallTimeout != tt.expected.CallTimeout {
				t.Errorf("CallTimeout = %d, want %d", tt.input.CallTimeout, tt.expected.CallTimeout)
			}
			if tt.input.NgrokURL != tt.expected.NgrokURL {
				t.Errorf("NgrokURL = %s, want %s", tt.input.NgrokURL, tt.expected.NgrokURL)
			}
			if tt.input.CallerID != tt.expected.CallerID {
				t.Errorf("CallerID = %s, want %s", tt.input.CallerID, tt.expected.CallerID)
			}
			if len(tt.input.AdminIDs) != len(tt.expected.AdminIDs) {
				t.Errorf("AdminIDs len = %d, want %d", len(tt.input.AdminIDs), len(tt.expected.AdminIDs))
			}
		})
	}
}

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
