package db

import (
	"errors"
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

func TestErrDatabaseNotInitialized(t *testing.T) {
	// Test that the error variable is properly defined
	if ErrDatabaseNotInitialized == nil {
		t.Error("ErrDatabaseNotInitialized should not be nil")
	}

	expectedMsg := "database has not been initialized"
	if ErrDatabaseNotInitialized.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, ErrDatabaseNotInitialized.Error())
	}

	// Test that it's detectable via errors.Is
	if !errors.Is(ErrDatabaseNotInitialized, ErrDatabaseNotInitialized) {
		t.Error("errors.Is should return true for the same error")
	}
}

func TestErrRecordNotFound(t *testing.T) {
	// Test that the error variable is properly defined
	if ErrRecordNotFound == nil {
		t.Error("ErrRecordNotFound should not be nil")
	}

	expectedMsg := "record not found"
	if ErrRecordNotFound.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, ErrRecordNotFound.Error())
	}

	// Test that it's detectable via errors.Is
	if !errors.Is(ErrRecordNotFound, ErrRecordNotFound) {
		t.Error("errors.Is should return true for the same error")
	}
}

func TestDatabaseStruct(t *testing.T) {
	db := &Database{}

	// Verify initial state
	if db.db != nil {
		t.Error("Expected db.db to be nil initially")
	}
}

func TestMaskPhoneExportEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Edge cases - maskPhoneExport shows first 4 and last 2 digits
		{"+1", "**"},           // Short (no plus handling)
		{"+12", "***"},         // Short (no plus handling)
		{"123", "***"},         // Exactly 3 chars (all masked)
		{"12345", "*****"},     // Exactly 5 chars (all masked since <=6)
		{"1234567", "1234*67"}, // 7 chars (first 4 + masked + last 2)
		
		// Long numbers
		{"+12345678901234567890", "+123***************90"}, // Long with plus prefix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskPhoneExport(tt.input)
			if result != tt.expected {
				t.Errorf("maskPhoneExport(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
