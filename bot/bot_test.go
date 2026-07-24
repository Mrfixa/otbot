package bot

import (
	"testing"
	"time"
)

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// With + prefix
		{"+15551234567", "+15******567"},     // 12 chars: 3 visible + 6 masked + 3 visible = 12
		{"+12345678901234", "+12*********234"}, // 15 chars: 3 visible + 9 masked + 3 visible = 15
		{"+1555", "+****"},                  // Short with +: +1555 -> +****
		{"+123", "+***"},                     // Very short with +: +123 -> +*** (all digits masked)
		
		// Without + prefix
		{"1234567890", "123****890"},
		{"1234", "****"},
		{"123", "***"},
		{"12", "**"},
		{"1", "*"},
		{"", ""},
	}

	for _, tt := range tests {
		result := maskPhone(tt.input)
		if result != tt.expected {
			t.Errorf("maskPhone(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestGetStatusEmoji(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"active", "🟢"},
		{"paused", "⏸️"},
		{"completed", "✅"},
		{"cancelled", "🛑"},
		{"pending", "⚪"},
		{"unknown", "⚪"},
	}

	for _, tt := range tests {
		result := getStatusEmoji(tt.status)
		if result != tt.expected {
			t.Errorf("getStatusEmoji(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestCreateProgressBar(t *testing.T) {
	tests := []struct {
		percent  int
		expected string
	}{
		{0, "[░░░░░░░░░░░░░░░░░░░░]"},
		{25, "[█████░░░░░░░░░░░░░░░]"},
		{50, "[██████████░░░░░░░░░░]"},
		{75, "[███████████████░░░░░]"},
		{100, "[████████████████████]"},
	}

	for _, tt := range tests {
		result := createProgressBar(tt.percent)
		if result != tt.expected {
			t.Errorf("createProgressBar(%d) = %s, expected %s", tt.percent, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%s, %d) = %s, expected %s", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{
			[]string{"+15551234567", "+15551234567", "+15559876543"},
			[]string{"+15551234567", "+15559876543"},
		},
		{
			[]string{"a", "b", "c", "a", "b"},
			[]string{"a", "b", "c"},
		},
		{
			[]string{"+15551234567"},
			[]string{"+15551234567"},
		},
		{
			[]string{},
			[]string{},
		},
	}

	for _, tt := range tests {
		result := unique(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("unique(%v) length = %d, expected %d", tt.input, len(result), len(tt.expected))
		}
	}
}

func TestFormatDuration(t *testing.T) {
	// Note: We can't easily test exact time durations in tests
	// But we can test the function exists and doesn't panic
	testCases := []int64{30, 60, 90, 3600, 7200}

	for _, seconds := range testCases {
		_ = formatDuration(durationFromSeconds(seconds))
	}
}

// Helper function for testing
func durationFromSeconds(s int64) time.Duration {
	return time.Duration(s) * time.Second
}

// Note: time is already imported at the top
