# 🔐 OTP Bot Master

Telegram-powered Voice OTP Spoofing Bot with Plivo Integration

## ⚡ Features

- **📱 Telegram Control Center** - Full bot control via Telegram commands
- **📞 Voice OTP Capture** - Call victims and capture DTMF tones
- **📋 Campaign Management** - Batch calling with real-time progress
- **📝 Service Templates** - Pre-built templates for Chase, PayPal, Amazon, etc.
- **⭐ Instant Capture Notifications** - Get OTPs delivered instantly
- **🗄️ SQLite Database** - Local storage with campaign history
- **🐳 Docker Ready** - One-command deployment
- **⚡ High Performance** - Concurrent calling with rate limiting

## 🚀 Quick Start

### Method 1: Docker (Recommended)

```bash
# Clone and setup
git clone <repo>
cd otp-bot

# Create config
cp config.yml.example config.yml

# Edit config with your credentials
nano config.yml

# Run with Docker Compose
docker-compose up -d
```

### Method 2: Manual Build

```bash
# Clone
git clone <repo>
cd otp-bot

# Install Go 1.18+
go mod download

# Build
go build -o otp-bot .

# Run
./otp-bot -config config.yml
```

## ⚙️ Configuration

Edit `config.yml`:

```yaml
# Telegram Bot Token (get from @BotFather)
bot_token: "123456:ABC-DEF..."

# Plivo Credentials
plivo_auth_id: "PAXXXXXX"
plivo_auth_token: "your_token"
plivo_number: "+15551234567"

# Admin Telegram IDs (leave empty to allow all)
admin_ids:
  - 123456789

# Server
port: "3000"
ngrok_url: "https://your-ngrok.ngrok.io"

# Campaign Settings
concurrency: 5
call_timeout: 60
```

## 📱 Telegram Commands

### Calling
| Command | Description |
|---------|-------------|
| `/call +15551234567 chase` | Make single call |
| `/sms +15551234567 message` | Send SMS fallback |

### Campaigns
| Command | Description |
|---------|-------------|
| `/batch` (reply to CSV) | Start batch campaign |
| `/campaigns` | List all campaigns |
| `/campaign 1` | Campaign details |
| `/stop 1` | Stop campaign |
| `/pause 1` | Pause campaign |
| `/resume 1` | Resume campaign |
| `/delete 1` | Delete campaign |

### Templates
| Command | Description |
|---------|-------------|
| `/templates` | List all templates |
| `/template chase` | Template details |
| `/addtemplate name\|voice\|greeting\|action\|otp\|confirm` | Add template |
| `/edittemplate ...` | Edit template |
| `/deltemplate name` | Delete template |

### Monitoring
| Command | Description |
|---------|-------------|
| `/stats` | Global statistics |
| `/logs 20` | Recent logs |
| `/export` | Export captures CSV |
| `/backup` | Database backup |

### Configuration
| Command | Description |
|---------|-------------|
| `/config` | Show config |
| `/reload` | Hot-reload config |
| `/cleanup 30` | Clean old data |

## 📋 Service Templates

Pre-built templates included:

- 🏦 **chase** - Chase Bank fraud alert
- 🏦 **bank_of_america** - Bank of America
- 💳 **paypal** - PayPal payment alert
- 📦 **amazon** - Amazon order verification
- 🎬 **netflix** - Netflix billing
- 🍎 **apple** - Apple ID verification
- 🔍 **google** - Google security alert
- 🎮 **steam** - Steam trade verification
- 🏦 **wells_fargo** - Wells Fargo fraud
- 💳 **citi** - Citibank fraud alert

## 📁 Batch CSV Format

```csv
+15551234567
+15559876543
+15551112222
+15553334455
```

Reply to CSV file with `/batch chase`

## 🐳 Docker Commands

```bash
# Start
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down

# Rebuild
docker-compose up -d --build
```

## 🔧 Ngrok Setup (Local Dev)

```bash
# Install ngrok
wget https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip
unzip ngrok-stable-linux-amd64.zip
./ngrok authtoken YOUR_TOKEN

# Start tunnel
./ngrok http 3000

# Copy the https URL to config.yml
ngrok_url: "https://abc123.ngrok.io"
```

## 📊 Campaign Flow

```
1. Create Campaign → /batch (reply to CSV)
2. Bot queues calls based on concurrency
3. Each call:
   ├─ Ring → Notify
   ├─ Answered → Play greeting + prompt
   ├─ DTMF "1" → Ask for OTP
   ├─ OTP Entered → Capture & notify
   └─ Confirmation → End call
4. Campaign complete → Stats summary
```

## 🔐 Security Notes

- Keep `config.yml` private
- Use environment variables in production
- Restrict `admin_ids` to trusted users
- Enable firewall on server port

## 📦 Project Structure

```
otp-bot/
├── main.go           # Entry point
├── bot/
│   ├── handlers.go   # Telegram commands
│   ├── handlers2.go  # Extended handlers
│   ├── campaign.go   # Campaign logic
│   └── webhooks.go   # Plivo webhooks
├── db/
│   └── database.go   # SQLite operations
├── voice/
│   └── plivo.go      # Plivo API wrapper
├── config/
│   └── loader.go     # Config management
├── models/
│   └── models.go     # Data structures
├── config.yml        # Configuration
├── Dockerfile        # Container build
├── docker-compose.yml
└── README.md
```

## ⚠️ Disclaimer

This tool is for **educational purposes only**. Unauthorized OTP interception is illegal. Use responsibly.

## 📜 License

MIT License
