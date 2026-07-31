package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TelnyxClient struct {
	APIKey  string
	Number  string
	BaseURL string
	Client  *http.Client
}

type TelnyxCallRequest struct {
	From      string // Caller ID - must be a Telnyx number you own
	To        string
	AnswerURL string // Webhook URL for call setup instructions
	Timeout   int    // Ring timeout in seconds
	WebhookURL string // For call status callbacks
}

type TelnyxCallResponse struct {
	Data struct {
		ID            string `json:"id"`
		RecordType    string `json:"record_type"`
		State         string `json:"state"`
		From          string `json:"from"`
		To            string `json:"to"`
		Direction     string `json:"direction"`
		AudioURL      string `json:"audio_url"`
		ClientState   string `json:"client_state"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
		EndedAt       string `json:"ended_at"`
		Duration      int    `json:"duration"`
		BillDuration  int    `json:"bill_duration"`
		CallLegID     string `json:"call_leg_id"`
		CallSessionID string `json:"call_session_id"`
		DTMFDigits    string `json:"dtmf_digits"`
	} `json:"data"`
}

type TelnyxMessageResponse struct {
	Data struct {
		ID             string `json:"id"`
		RecordType     string `json:"record_type"`
		To             string `json:"to"`
		From           string `json:"from"`
		Text           string `json:"text"`
		Type           string `json:"type"`
		WebhookURL     string `json:"webhook_url"`
		WebhookFailURL string `json:"webhook_fail_url"`
		Encoding       string `json:"encoding"`
		Priority       string `json:"priority"`
		Cost           struct {
			Amount   string `json:"amount"`
			Currency string `json:"currency"`
		} `json:"cost"`
		Carrier struct {
			Name   string `json:"name"`
			RCS    bool   `json:"rcs"`
			MCC     int    `json:"mcc"`
			MNC     int    `json:"mnc"`
		} `json:"carrier"`
	} `json:"data"`
}

func NewTelnyxClient(apiKey, number string) *TelnyxClient {
	return &TelnyxClient{
		APIKey:  apiKey,
		Number:  number,
		BaseURL: "https://api.telnyx.com/v2",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *TelnyxClient) MakeCall(req TelnyxCallRequest) (*TelnyxCallResponse, error) {
	url := fmt.Sprintf("%s/calls", t.BaseURL)

	payload := map[string]interface{}{
		"connection_id": t.Number, // Your Telnyx connection/messaging profile ID
		"to":            req.To,
		"from":          req.From,
		"timeout_secs":  req.Timeout,
	}

	// Add custom caller ID (any number you own on your Telnyx account)
	if req.From != "" {
		payload["from"] = req.From
	}

	// Add answer URL for call instructions
	if req.AnswerURL != "" {
		payload["answer"] = map[string]interface{}{
			"url": req.AnswerURL,
			"method": "POST",
		}
	}

	// Add webhook for status updates
	if req.WebhookURL != "" {
		payload["events"] = []string{"call.initiated", "call.answered", "call.hangup", "call.completed"}
		payload["webhook_url"] = req.WebhookURL
		payload["webhook_timeout_secs"] = 30
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	t.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("telnyx API error: %s - %s", resp.Status, string(body))
	}

	var callResp TelnyxCallResponse
	if err := json.Unmarshal(body, &callResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &callResp, nil
}

func (t *TelnyxClient) HangupCall(callID string) error {
	url := fmt.Sprintf("%s/calls/%s", t.BaseURL, callID)

	payload := map[string]interface{}{
		"state": "hangup",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	t.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to hangup call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telnyx API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (t *TelnyxClient) GetCallStatus(callID string) (*TelnyxCallResponse, error) {
	url := fmt.Sprintf("%s/calls/%s", t.BaseURL, callID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	t.setAuthHeader(httpReq)

	resp, err := t.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get call status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telnyx API error: %s - %s", resp.Status, string(body))
	}

	var callResp TelnyxCallResponse
	if err := json.Unmarshal(body, &callResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &callResp, nil
}

func (t *TelnyxClient) SendSMS(from, to, text string) (*TelnyxMessageResponse, error) {
	url := fmt.Sprintf("%s/messages", t.BaseURL)

	payload := map[string]interface{}{
		"from": from,
		"to":   to,
		"text": text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	t.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telnyx API error: %s - %s", resp.Status, string(body))
	}

	var msgResp TelnyxMessageResponse
	if err := json.Unmarshal(body, &msgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msgResp, nil
}

func (t *TelnyxClient) GetNumbers() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/available_numbers", t.BaseURL)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	t.setAuthHeader(httpReq)

	resp, err := t.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get numbers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telnyx API error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

func (t *TelnyxClient) setAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+t.APIKey)
}

// Telnyx XML Response builders
func BuildTelnyxResponse(greeting, actionURL, confirmation, holdMusic, voice, language string, loops int) string {
	var xml strings.Builder

	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	xml.WriteString("\n<Response>")

	xml.WriteString("\n  <Gather action=\"")
	xml.WriteString(actionURL)
	xml.WriteString(`" method="POST" numDigits="6" timeout="15">`)
	xml.WriteString("\n    <Speak language=\"")
	xml.WriteString(language)
	xml.WriteString(`" voice="`)
	xml.WriteString(voice)
	xml.WriteString(`" loop="`)
	xml.WriteString(fmt.Sprintf("%d", loops))
	xml.WriteString(`">`)
	xml.WriteString("\n      ")
	xml.WriteString(greeting)
	xml.WriteString("\n    </Speak>")

	if holdMusic != "" {
		xml.WriteString("\n    <Play>")
		xml.WriteString(holdMusic)
		xml.WriteString("</Play>")
	}

	xml.WriteString("\n  </Gather>")

	xml.WriteString("\n  <Speak language=\"")
	xml.WriteString(language)
	xml.WriteString(`" voice="`)
	xml.WriteString(voice)
	xml.WriteString(`">`)
	xml.WriteString("\n    ")
	xml.WriteString(confirmation)
	xml.WriteString("\n  </Speak>")

	xml.WriteString("\n  <Redirect method=\"POST\">")
	xml.WriteString(actionURL)
	xml.WriteString("</Redirect>")

	xml.WriteString("\n</Response>")

	return xml.String()
}

func BuildTelnyxDTMFResponse(message, holdMusic, voice, language string) string {
	var xml strings.Builder

	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	xml.WriteString("\n<Response>")
	xml.WriteString("\n  <Speak language=\"")
	xml.WriteString(language)
	xml.WriteString(`" voice="`)
	xml.WriteString(voice)
	xml.WriteString(`" loop="2">`)
	xml.WriteString("\n    ")
	xml.WriteString(message)
	xml.WriteString("\n  </Speak>")

	if holdMusic != "" {
		xml.WriteString("\n  <Play>")
		xml.WriteString(holdMusic)
		xml.WriteString("</Play>")
		xml.WriteString("\n  <Wait length=\"10\"/>")
	}

	xml.WriteString("\n</Response>")

	return xml.String()
}

func BuildTelnyxHangupResponse(message string) string {
	var xml strings.Builder

	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	xml.WriteString("\n<Response>")

	if message != "" {
		xml.WriteString("\n  <Speak language=\"en-US\" voice=\"female\">")
		xml.WriteString("\n    ")
		xml.WriteString(message)
		xml.WriteString("\n  </Speak>")
	}

	xml.WriteString("\n  <Hangup/>")
	xml.WriteString("\n</Response>")

	return xml.String()
}

func BuildTelnyxRingResponse() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Response><Dial answerOnBridge="true"></Dial></Response>`
}

func BuildTelnyxMachineResponse() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Response><Hangup reason="MACHINE_DETECTED"/></Response>`
}
