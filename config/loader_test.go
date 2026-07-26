package config

import (
	"errors"
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				PlivoNumber:  "+15551234567",
				CallerID:     "+15551234567",
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				PlivoNumber:  "+15551234567",
				CallerID:     "+15559876543",
				CallerName:   "Security Alert",
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
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{123, 456, 789},
				CallerName:   "Security Alert",
			},
		},
		{
			name: "CallerName defaults to Security Alert",
			input: Config{},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
			},
		},
		{
			name: "CallerName preserved if set",
			input: Config{
				CallerName: "My Custom Bank",
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "My Custom Bank",
			},
		},
		{
			name: "RingTimeout defaults to 30",
			input: Config{},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
			},
		},
		{
			name: "RingTimeout preserved if set",
			input: Config{
				RingTimeout: 60,
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				RingTimeout:  60,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
			},
		},
		{
			name: "Zero RingTimeout gets default",
			input: Config{
				RingTimeout: 0,
			},
			expected: Config{
				Port:         "3000",
				DatabasePath: "./data/bot.db",
				LogPath:      "./logs/bot.log",
				Concurrency:  5,
				MaxRetries:   3,
				CallTimeout:  30,
				RingTimeout:  30,
				NgrokURL:     "http://localhost:4040",
				AdminIDs:     []int64{},
				CallerName:   "Security Alert",
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
			if tt.input.RingTimeout != tt.expected.RingTimeout {
				t.Errorf("RingTimeout = %d, want %d", tt.input.RingTimeout, tt.expected.RingTimeout)
			}
			if tt.input.NgrokURL != tt.expected.NgrokURL {
				t.Errorf("NgrokURL = %s, want %s", tt.input.NgrokURL, tt.expected.NgrokURL)
			}
			if tt.input.CallerID != tt.expected.CallerID {
				t.Errorf("CallerID = %s, want %s", tt.input.CallerID, tt.expected.CallerID)
			}
			if tt.input.CallerName != tt.expected.CallerName {
				t.Errorf("CallerName = %s, want %s", tt.input.CallerName, tt.expected.CallerName)
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
		{
			name: "Concurrency too low",
			config: Config{
				BotToken:       "token",
				PlivoAuthID:    "id",
				PlivoAuthToken: "token",
				PlivoNumber:    "+1234567890",
				NgrokURL:       "https://example.ngrok.io",
				Concurrency:    0, // Invalid: must be 1-50
				CallTimeout:    30,
			},
			expected: 1, // Just concurrency error
		},
		{
			name: "Concurrency too high",
			config: Config{
				BotToken:       "token",
				PlivoAuthID:    "id",
				PlivoAuthToken: "token",
				PlivoNumber:    "+1234567890",
				NgrokURL:       "https://example.ngrok.io",
				Concurrency:    100, // Invalid: must be 1-50
				CallTimeout:    30,
			},
			expected: 1, // Just concurrency error
		},
		{
			name: "CallTimeout too low",
			config: Config{
				BotToken:       "token",
				PlivoAuthID:    "id",
				PlivoAuthToken: "token",
				PlivoNumber:    "+1234567890",
				NgrokURL:       "https://example.ngrok.io",
				Concurrency:    5,
				CallTimeout:    5, // Invalid: must be 10-300
			},
			expected: 1, // Just call_timeout error
		},
		{
			name: "CallTimeout too high",
			config: Config{
				BotToken:       "token",
				PlivoAuthID:    "id",
				PlivoAuthToken: "token",
				PlivoNumber:    "+1234567890",
				NgrokURL:       "https://example.ngrok.io",
				Concurrency:    5,
				CallTimeout:    500, // Invalid: must be 10-300
			},
			expected: 1, // Just call_timeout error
		},
		{
			name: "Multiple validation errors",
			config: Config{
				BotToken:       "", // Missing
				PlivoAuthID:    "", // Missing
				PlivoAuthToken: "", // Missing
				PlivoNumber:    "", // Missing
				NgrokURL:       "", // Missing
				Concurrency:    0,  // Invalid
				CallTimeout:    0,  // Invalid
			},
			expected: 7, // All required fields missing + valid ranges
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

func TestErrConfigNotLoaded(t *testing.T) {
	// Test that the error variable is properly defined
	if ErrConfigNotLoaded == nil {
		t.Error("ErrConfigNotLoaded should not be nil")
	}

	expectedMsg := "configuration has not been loaded"
	if ErrConfigNotLoaded.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, ErrConfigNotLoaded.Error())
	}

	// Test that it's detectable via errors.Is
	if !errors.Is(ErrConfigNotLoaded, ErrConfigNotLoaded) {
		t.Error("errors.Is should return true for the same error")
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		BotToken:       "test-token",
		PlivoAuthID:    "auth-id",
		PlivoAuthToken: "auth-token",
		PlivoNumber:    "+15551234567",
		CallerID:       "+15559876543",
		CallerName:     "Test Bank",
		EnableCNAM:     true,
		RandomizeCID:   false,
		CallerIDPool:   []string{"+1111", "+2222"},
		AdminIDs:       []int64{123, 456},
		NgrokURL:       "https://example.ngrok.io",
		Port:           "8080",
		DatabasePath:    "./test.db",
		LogPath:        "./test.log",
		Concurrency:    10,
		MaxRetries:     5,
		CallTimeout:    60,
		RingTimeout:    30,
		MachineDetection: true,
	}

	if cfg.BotToken != "test-token" {
		t.Errorf("Expected BotToken 'test-token', got '%s'", cfg.BotToken)
	}

	if cfg.PlivoAuthID != "auth-id" {
		t.Errorf("Expected PlivoAuthID 'auth-id', got '%s'", cfg.PlivoAuthID)
	}

	if cfg.PlivoAuthToken != "auth-token" {
		t.Errorf("Expected PlivoAuthToken 'auth-token', got '%s'", cfg.PlivoAuthToken)
	}

	if cfg.PlivoNumber != "+15551234567" {
		t.Errorf("Expected PlivoNumber '+15551234567', got '%s'", cfg.PlivoNumber)
	}

	if cfg.CallerID != "+15559876543" {
		t.Errorf("Expected CallerID '+15559876543', got '%s'", cfg.CallerID)
	}

	if cfg.CallerName != "Test Bank" {
		t.Errorf("Expected CallerName 'Test Bank', got '%s'", cfg.CallerName)
	}

	if !cfg.EnableCNAM {
		t.Error("Expected EnableCNAM to be true")
	}

	if cfg.RandomizeCID {
		t.Error("Expected RandomizeCID to be false")
	}

	if len(cfg.CallerIDPool) != 2 {
		t.Errorf("Expected CallerIDPool length 2, got %d", len(cfg.CallerIDPool))
	}

	if len(cfg.AdminIDs) != 2 {
		t.Errorf("Expected AdminIDs length 2, got %d", len(cfg.AdminIDs))
	}

	if cfg.Concurrency != 10 {
		t.Errorf("Expected Concurrency 10, got %d", cfg.Concurrency)
	}

	if cfg.CallTimeout != 60 {
		t.Errorf("Expected CallTimeout 60, got %d", cfg.CallTimeout)
	}

	if cfg.MachineDetection != true {
		t.Error("Expected MachineDetection to be true")
	}
}

func TestLoaderStruct(t *testing.T) {
	loader := &Loader{
		filePath: "/path/to/config.yml",
		loaded:   true,
	}

	if loader.filePath != "/path/to/config.yml" {
		t.Errorf("Expected filePath '/path/to/config.yml', got '%s'", loader.filePath)
	}

	if !loader.loaded {
		t.Error("Expected loaded to be true")
	}
}
