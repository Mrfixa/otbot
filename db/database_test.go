package db

import (
	"testing"
)

func TestMaskPhoneExport(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Normal phone numbers - shows first 4 chars, last 2 chars
		{"+15551234567", "+155******67"},  // 12 chars: 4 shown, 6 masked (12-6=6), 2 shown
		{"15551234567", "1555*****67"},    // 11 chars: 4 shown, 5 masked (11-6=5), 2 shown
		
		// Short phone numbers (<=6 chars) - all masked
		{"123456", "******"},
		{"12345", "*****"},
		{"1234", "****"},
		{"123", "***"},
		{"12", "**"},
		{"1", "*"},
		{"", ""},
		
		// Phone with spaces and dashes
		{"+1 555 123 4567", "+1 5*********67"},  // first 4: "+1 5", last 2: "67", masked: 14-6=8
		
		// Long phone numbers
		{"+12345678901234", "+123*********34"},  // 15 chars: 4 shown, 9 masked (15-6=9), 2 shown
	}

	for _, tt := range tests {
		result := maskPhoneExport(tt.input)
		if result != tt.expected {
			t.Errorf("maskPhoneExport(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestIsInitialized(t *testing.T) {
	tests := []struct {
		name        string
		db          *Database
		expectError bool
	}{
		{
			name:        "Nil database",
			db:          nil,
			expectError: true,
		},
		{
			name:        "Database with nil db pointer",
			db:          &Database{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.db.isInitialized()
			if tt.expectError && err == nil {
				t.Error("Expected error for nil database, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}
