package voice

import (
	"strings"
	"testing"
)

func TestNewPlivoClient(t *testing.T) {
	client := NewPlivoClient("auth-id", "auth-token", "+15551234567")

	if client.AuthID != "auth-id" {
		t.Errorf("Expected AuthID 'auth-id', got '%s'", client.AuthID)
	}

	if client.AuthToken != "auth-token" {
		t.Errorf("Expected AuthToken 'auth-token', got '%s'", client.AuthToken)
	}

	if client.Number != "+15551234567" {
		t.Errorf("Expected Number '+15551234567', got '%s'", client.Number)
	}

	if client.BaseURL != "https://api.plivo.com/v1/Account/" {
		t.Errorf("Expected BaseURL, got '%s'", client.BaseURL)
	}

	if client.Client == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestBuildXMLResponse(t *testing.T) {
	greeting := "Hello customer"
	otpPrompt := "/capture_otp/1/1"
	confirmation := "Thank you"
	holdMusic := ""
	voice := "WOMAN"
	language := "en-US"
	loops := 2

	xml := BuildXMLResponse(greeting, otpPrompt, confirmation, holdMusic, voice, language, loops)

	// Check XML structure
	if !strings.Contains(xml, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML should contain XML declaration")
	}

	if !strings.Contains(xml, "<Response>") {
		t.Error("XML should contain <Response>")
	}

	if !strings.Contains(xml, "<GetInput") {
		t.Error("XML should contain <GetInput>")
	}

	if !strings.Contains(xml, greeting) {
		t.Error("XML should contain greeting text")
	}

	if !strings.Contains(xml, "<Speak") {
		t.Error("XML should contain <Speak> elements")
	}

	if !strings.Contains(xml, "</Response>") {
		t.Error("XML should close with </Response>")
	}

	// Check voice and language attributes
	if !strings.Contains(xml, "voice=\""+voice+"\"") {
		t.Error("XML should contain voice attribute")
	}

	if !strings.Contains(xml, "language=\""+language+"\"") {
		t.Error("XML should contain language attribute")
	}
}

func TestBuildDTMFXML(t *testing.T) {
	message := "Please enter your OTP"
	holdMusic := "https://example.com/music.mp3"
	voice := "WOMAN"
	language := "en-US"

	xml := BuildDTMFXML(message, holdMusic, voice, language)

	// Check structure
	if !strings.Contains(xml, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML should contain XML declaration")
	}

	if !strings.Contains(xml, "<Response>") {
		t.Error("XML should contain <Response>")
	}

	if !strings.Contains(xml, message) {
		t.Error("XML should contain message")
	}

	if !strings.Contains(xml, "<Play>") {
		t.Error("XML should contain <Play> for hold music")
	}

	if !strings.Contains(xml, holdMusic) {
		t.Error("XML should contain hold music URL")
	}
}

func TestBuildHangupXML(t *testing.T) {
	message := "Goodbye"

	xml := BuildHangupXML(message)

	if !strings.Contains(xml, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML should contain XML declaration")
	}

	if !strings.Contains(xml, "<Response>") {
		t.Error("XML should contain <Response>")
	}

	if !strings.Contains(xml, message) {
		t.Error("XML should contain message")
	}

	if !strings.Contains(xml, "<Hangup/>") {
		t.Error("XML should contain <Hangup/>")
	}
}

func TestBuildHangupXMLEmptyMessage(t *testing.T) {
	xml := BuildHangupXML("")

	if !strings.Contains(xml, "<Hangup/>") {
		t.Error("XML should contain <Hangup/> even without message")
	}

	// Should not contain <Speak> for empty message
	if strings.Contains(xml, "<Speak>") {
		t.Error("XML should not contain <Speak> for empty message")
	}
}

func TestBuildRingXML(t *testing.T) {
	xml := BuildRingXML()

	if !strings.Contains(xml, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML should contain XML declaration")
	}

	if !strings.Contains(xml, "<Response></Response>") {
		t.Error("Ring XML should be empty Response")
	}
}

func TestBuildMachineXML(t *testing.T) {
	xml := BuildMachineXML()

	if !strings.Contains(xml, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("XML should contain XML declaration")
	}

	if !strings.Contains(xml, "<Hangup") {
		t.Error("Machine XML should contain <Hangup>")
	}

	if !strings.Contains(xml, "-machine") {
		t.Error("Machine XML should contain -machine reason")
	}
}

func TestURLEncode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello world", "hello+world"},
		{"test+value", "test%2Bvalue"},
		{"special=chars&here", "special%3Dchars%26here"},
	}

	for _, tt := range tests {
		result := URLEncode(tt.input)
		if result != tt.expected {
			t.Errorf("URLEncode(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestCallRequest(t *testing.T) {
	req := CallRequest{
		From:        "+15551234567",
		To:          "+15559876543",
		AnswerURL:   "https://example.com/answer",
		HangupURL:   "https://example.com/hangup",
		RingURL:     "https://example.com/ring",
		TimeLimit:   60,
		RingTimeout: 30,
	}

	if req.From != "+15551234567" {
		t.Errorf("Expected From '+15551234567', got '%s'", req.From)
	}

	if req.To != "+15559876543" {
		t.Errorf("Expected To '+15559876543', got '%s'", req.To)
	}

	if req.TimeLimit != 60 {
		t.Errorf("Expected TimeLimit 60, got %d", req.TimeLimit)
	}
}

func TestCallResponse(t *testing.T) {
	resp := CallResponse{
		APIID:       "api-123",
		RequestUUID: "uuid-456",
		UUID:        "call-789",
		Message:     "call initiated",
	}

	if resp.APIID != "api-123" {
		t.Errorf("Expected APIID 'api-123', got '%s'", resp.APIID)
	}

	if resp.RequestUUID != "uuid-456" {
		t.Errorf("Expected RequestUUID 'uuid-456', got '%s'", resp.RequestUUID)
	}
}

func TestXMLResponseWithHoldMusic(t *testing.T) {
	xml := BuildXMLResponse(
		"Hello",
		"/callback",
		"Confirmed",
		"https://example.com/hold.mp3",
		"WOMAN",
		"en-US",
		1,
	)

	if !strings.Contains(xml, "<Play>") {
		t.Error("XML with hold music should contain <Play>")
	}

	if !strings.Contains(xml, "https://example.com/hold.mp3") {
		t.Error("XML should contain hold music URL")
	}
}
