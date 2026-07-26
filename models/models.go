package models

import (
	"time"
)


type CampaignStatus string

const (
	CampaignStatusPending   CampaignStatus = "pending"
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusCancelled CampaignStatus = "cancelled"
)


type CallStatus string

const (
	CallStatusPending    CallStatus = "pending"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusAnswered   CallStatus = "answered"
	CallStatusVoicemail  CallStatus = "voicemail"
	CallStatusNoAnswer   CallStatus = "no_answer"
	CallStatusBusy       CallStatus = "busy"
	CallStatusFailed     CallStatus = "failed"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusCancelled  CallStatus = "cancelled"
)


type Campaign struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Service     string          `json:"service" db:"service"`
	Status      CampaignStatus  `json:"status" db:"status"`
	TotalCalls  int             `json:"total_calls" db:"total_calls"`
	Completed   int             `json:"completed" db:"completed"`
	Captures    int             `json:"captures" db:"captures"`
	Concurrency int             `json:"concurrency" db:"concurrency"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
	ScheduleAt  *time.Time     `json:"schedule_at,omitempty" db:"schedule_at"`
}


type Call struct {
	ID          int64      `json:"id" db:"id"`
	CampaignID  int64      `json:"campaign_id" db:"campaign_id"`
	Phone       string     `json:"phone" db:"phone"`
	Status      CallStatus `json:"status" db:"status"`
	Duration    int        `json:"duration" db:"duration"` 
	PlivoCallID string     `json:"plivo_call_id" db:"plivo_call_id"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	AnsweredAt  *time.Time `json:"answered_at,omitempty" db:"answered_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty" db:"ended_at"`
}


type Capture struct {
	ID        int64     `json:"id" db:"id"`
	CallID    int64     `json:"call_id" db:"call_id"`
	CampaignID int64    `json:"campaign_id" db:"campaign_id"`
	Phone     string    `json:"phone" db:"phone"`
	OTP       string    `json:"otp" db:"otp"`
	Service   string    `json:"service" db:"service"`
	CapturedAt time.Time `json:"captured_at" db:"captured_at"`
}


type Template struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Voice     string    `json:"voice" db:"voice"`
	Greeting  string    `json:"greeting" db:"greeting"`
	ActionPrompt string `json:"action_prompt" db:"action_prompt"`
	OTPPrompt   string `json:"otp_prompt" db:"otp_prompt"`
	Confirmation string `json:"confirmation" db:"confirmation"`
	FallbackMessage string `json:"fallback_message" db:"fallback_message"`
	HoldMusic  string    `json:"hold_music" db:"hold_music"`
	Category   string    `json:"category" db:"category"`   // Category for organization
	Icon       string    `json:"icon" db:"icon"`           // Emoji icon for display
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Template categories for organization
type TemplateCategory string

const (
	CategoryBanking    TemplateCategory = "banking"
	CategoryTech      TemplateCategory = "tech"
	CategoryEcommerce TemplateCategory = "ecommerce"
	CategorySocial    TemplateCategory = "social"
	CategoryGovernment TemplateCategory = "government"
	CategoryOther     TemplateCategory = "other"
)


type LogEntry struct {
	ID        int64     `json:"id" db:"id"`
	Level     string    `json:"level" db:"level"`
	Message   string    `json:"message" db:"message"`
	Details   string    `json:"details" db:"details"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}


type CampaignStats struct {
	CampaignID    int64   `json:"campaign_id"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	TotalCalls    int     `json:"total_calls"`
	Answered      int     `json:"answered"`
	Voicemails    int     `json:"voicemails"`
	NoAnswers     int     `json:"no_answers"`
	Failed        int     `json:"failed"`
	Captures      int     `json:"captures"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDuration   float64 `json:"avg_duration"`
	Duration      string  `json:"duration"`
}


type GlobalStats struct {
	TotalCampaigns    int     `json:"total_campaigns"`    
	ActiveCampaigns  int     `json:"active_campaigns"`   
	PausedCampaigns  int     `json:"paused_campaigns"`  
	CompletedCampaigns int   `json:"completed_campaigns"`  
	TotalCalls       int64   `json:"total_calls"`         
	AnsweredCalls    int64   `json:"answered_calls"`      
	Voicemails       int64   `json:"voicemails"`      
	FailedCalls      int64   `json:"failed_calls"`      
	TotalCaptures    int64   `json:"total_captures"`     
	SuccessRate      float64 `json:"success_rate"`       
	TodayCalls       int64   `json:"today_calls"`        
	TodayCaptures    int64   `json:"today_captures"`     
}
