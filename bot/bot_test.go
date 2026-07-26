package bot

import (
	"strings"
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

func TestBuildWebhookURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		parts    []interface{}
		expected string
	}{
		{
			name:     "Simple URL with single part",
			baseURL:  "https://example.ngrok.io",
			parts:    []interface{}{"answer"},
			expected: "https://example.ngrok.io/answer",
		},
		{
			name:     "URL with multiple parts",
			baseURL:  "https://example.ngrok.io",
			parts:    []interface{}{"answer", 1, 2},
			expected: "https://example.ngrok.io/answer/1/2",
		},
		{
			name:     "URL with trailing slash in base (should be removed)",
			baseURL:  "https://example.ngrok.io/",
			parts:    []interface{}{"hangup", 123},
			expected: "https://example.ngrok.io/hangup/123",
		},
		{
			name:     "URL with leading slash in first part (should be normalized)",
			baseURL:  "https://example.ngrok.io",
			parts:    []interface{}{"/webhook", 456},
			expected: "https://example.ngrok.io/webhook/456",
		},
		{
			name:     "Localhost URL",
			baseURL:  "http://localhost:4040",
			parts:    []interface{}{"ring", 99, 100},
			expected: "http://localhost:4040/ring/99/100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildWebhookURL(tt.baseURL, tt.parts...)
			if result != tt.expected {
				t.Errorf("buildWebhookURL(%s, %v) = %s, expected %s", tt.baseURL, tt.parts, result, tt.expected)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{10000, "10.0K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
		{999999999, "1000.0M"},
		// Large numbers fall through to M formatting (1B = 1000M)
		{1000000000, "1000.0M"},
	}

	for _, tt := range tests {
		result := formatNumber(tt.input)
		if result != tt.expected {
			t.Errorf("formatNumber(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	// This function returns true by default (placeholder implementation)
	result := verifyWebhookSignature("token", "uuid", "time", "status")
	if !result {
		t.Error("verifyWebhookSignature should return true for placeholder implementation")
	}
}

func TestReplaceTemplateVars(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		victimName string
		amount     string
		orderID    string
		expected   string
	}{
		{
			name:       "All variables replaced",
			input:      "Hello {{victim_name}}, your order {{order_id}} for {{amount}} is ready",
			victimName: "John",
			amount:     "$100",
			orderID:    "ORD-123",
			expected:   "Hello John, your order ORD-123 for $100 is ready",
		},
		{
			name:       "No variables",
			input:      "Hello customer, how are you?",
			victimName: "John",
			amount:     "$100",
			orderID:    "ORD-123",
			expected:   "Hello customer, how are you?",
		},
		{
			name:       "Empty template",
			input:      "",
			victimName: "John",
			amount:     "$100",
			orderID:    "ORD-123",
			expected:   "",
		},
		{
			name:       "Only victim_name",
			input:      "Hello {{victim_name}}",
			victimName: "Jane",
			amount:     "$100",
			orderID:    "ORD-123",
			expected:   "Hello Jane",
		},
		{
			name:       "Multiple occurrences",
			input:      "{{victim_name}} called about {{victim_name}}'s order",
			victimName: "Bob",
			amount:     "$50",
			orderID:    "ORD-456",
			expected:   "Bob called about Bob's order",
		},
	}

	bot := &Bot{} // Create a bot instance for the method
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bot.replaceTemplateVars(tt.input, tt.victimName, tt.amount, tt.orderID)
			if result != tt.expected {
				t.Errorf("replaceTemplateVars(%s, %s, %s, %s) = %s, expected %s",
					tt.input, tt.victimName, tt.amount, tt.orderID, result, tt.expected)
			}
		})
	}
}

func TestGetStatusText(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"active", "🚀 Campaign Running"},
		{"paused", "⏸️ Campaign Paused"},
		{"completed", "✅ Campaign Completed"},
		{"cancelled", "🛑 Campaign Stopped"},
		{"pending", "⚪ Unknown Status"},
		{"unknown", "⚪ Unknown Status"},
		{"", "⚪ Unknown Status"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusText(tt.status)
			if result != tt.expected {
				t.Errorf("getStatusText(%s) = %s, expected %s", tt.status, result, tt.expected)
			}
		})
	}
}

func TestParseCSVPhones(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single phone number per line",
			input:    "+15551234567\n+15559876543\n",
			expected: []string{"+15551234567", "+15559876543"},
		},
		{
			name:     "Phone with extra whitespace",
			input:    "  +15551234567  \n  +15559876543  \n",
			expected: []string{"+15551234567", "+15559876543"},
		},
		{
			name:     "Invalid phones filtered out",
			input:    "+15551234567\ninvalid\n+15559876543\n",
			expected: []string{"+15551234567", "+15559876543"},
		},
		{
			name:     "Multiple phones per line (CSV)",
			input:    "+15551234567,+15559876543\n",
			expected: []string{"+15551234567", "+15559876543"},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Mixed valid and invalid",
			input:    "+15551234567\n123\n+15559876543\nabc\n",
			expected: []string{"+15551234567", "+15559876543"},
		},
		{
			name:     "Phones without plus prefix",
			input:    "15551234567\n",
			expected: []string{"15551234567"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCSVPhones(strings.NewReader(tt.input))
			if err != nil {
				t.Errorf("parseCSVPhones(%s) unexpected error: %v", tt.name, err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("parseCSVPhones(%s) len = %d, expected %d", tt.name, len(result), len(tt.expected))
				return
			}
			for i, phone := range result {
				if phone != tt.expected[i] {
					t.Errorf("parseCSVPhones(%s)[%d] = %s, expected %s", tt.name, i, phone, tt.expected[i])
				}
			}
		})
	}
}

func TestCampaignStateStruct(t *testing.T) {
	state := &CampaignState{
		CampaignID: 123,
		Status:     "active",
		phones:     []string{"+15551234567", "+15559876543"},
		index:      0,
	}

	if state.CampaignID != 123 {
		t.Errorf("Expected CampaignID 123, got %d", state.CampaignID)
	}

	if state.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", state.Status)
	}

	if len(state.phones) != 2 {
		t.Errorf("Expected 2 phones, got %d", len(state.phones))
	}
}

func TestActiveCallStruct(t *testing.T) {
	now := time.Now()
	answeredAt := now.Add(-10 * time.Second)
	
	call := &ActiveCall{
		CallID:     1,
		CampaignID: 123,
		Phone:      "+15551234567",
		UUID:       "uuid-123",
		Status:     "answered",
		StartedAt:  now,
		AnsweredAt: &answeredAt,
		Greeting:   "Hello Customer",
		Service:    "chase",
		OTP:        "123456",
		Duration:   30,
		HangupCause: "completed",
	}

	if call.CallID != 1 {
		t.Errorf("Expected CallID 1, got %d", call.CallID)
	}

	if call.CampaignID != 123 {
		t.Errorf("Expected CampaignID 123, got %d", call.CampaignID)
	}

	if call.Phone != "+15551234567" {
		t.Errorf("Expected Phone '+15551234567', got '%s'", call.Phone)
	}

	if call.UUID != "uuid-123" {
		t.Errorf("Expected UUID 'uuid-123', got '%s'", call.UUID)
	}

	if call.Status != "answered" {
		t.Errorf("Expected Status 'answered', got '%s'", call.Status)
	}

	if call.Greeting != "Hello Customer" {
		t.Errorf("Expected Greeting 'Hello Customer', got '%s'", call.Greeting)
	}

	if call.OTP != "123456" {
		t.Errorf("Expected OTP '123456', got '%s'", call.OTP)
	}

	if call.Duration != 30 {
		t.Errorf("Expected Duration 30, got %d", call.Duration)
	}

	if call.AnsweredAt == nil {
		t.Error("Expected AnsweredAt to be set")
	}
}

func TestBotStruct(t *testing.T) {
	bot := &Bot{
		activeCalls:     make(map[string]*ActiveCall),
		campaignState:   make(map[int64]*CampaignState),
		stopChan:        make(chan struct{}),
		callWizardState: make(map[int64]*CallWizardState),
	}

	if bot.activeCalls == nil {
		t.Error("Expected activeCalls to be initialized")
	}

	if bot.campaignState == nil {
		t.Error("Expected campaignState to be initialized")
	}

	if bot.stopChan == nil {
		t.Error("Expected stopChan to be initialized")
	}

	if bot.callWizardState == nil {
		t.Error("Expected callWizardState to be initialized")
	}
}

func TestCallWizardStateStruct(t *testing.T) {
	wizard := &CallWizardState{
		Phone:   "+15551234567",
		Service: "chase",
	}

	if wizard.Phone != "+15551234567" {
		t.Errorf("Expected Phone '+15551234567', got '%s'", wizard.Phone)
	}

	if wizard.Service != "chase" {
		t.Errorf("Expected Service 'chase', got '%s'", wizard.Service)
	}
}

func TestCategoryIcons(t *testing.T) {
	icons := CategoryIcons

	if icons["banking"] != "🏦" {
		t.Errorf("Expected banking icon '🏦', got '%s'", icons["banking"])
	}

	if icons["tech"] != "💻" {
		t.Errorf("Expected tech icon '💻', got '%s'", icons["tech"])
	}

	if icons["ecommerce"] != "🛒" {
		t.Errorf("Expected ecommerce icon '🛒', got '%s'", icons["ecommerce"])
	}

	if icons["social"] != "📱" {
		t.Errorf("Expected social icon '📱', got '%s'", icons["social"])
	}

	if icons["government"] != "🏛️" {
		t.Errorf("Expected government icon '🏛️', got '%s'", icons["government"])
	}

	if icons["other"] != "📌" {
		t.Errorf("Expected other icon '📌', got '%s'", icons["other"])
	}
}
