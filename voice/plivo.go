package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SSML语音合成模块 - 支持丰富的语音标签
// 支持: <break>, <emphasis>, <phoneme>, <lang>, <say-as>, <audio>, <prosody>

type SSMLBuilder struct {
	xml strings.Builder
}

func NewSSMLBuilder() *SSMLBuilder {
	return &SSMLBuilder{}
}

func (s *SSMLBuilder) Start() *SSMLBuilder {
	s.xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	s.xml.WriteString("\n<Response>")
	return s
}

func (s *SSMLBuilder) End() string {
	s.xml.WriteString("\n</Response>")
	return s.xml.String()
}

// Speak 添加语音文本
func (s *SSMLBuilder) Speak(text string, opts ...SpeakOption) *SSMLBuilder {
	s.xml.WriteString("\n  <Speak")
	
	voice := "WOMAN"
	lang := "en-US"
	loop := 1
	
	for _, opt := range opts {
		switch opt.key {
		case "voice":
			voice = opt.value
		case "language":
			lang = opt.value
		case "loop":
			loop, _ = strconv.Atoi(opt.value)
		}
	}
	
	s.xml.WriteString(fmt.Sprintf(` voice="%s" language="%s" loop="%d">`, voice, lang, loop))
	s.xml.WriteString("\n    ")
	s.xml.WriteString(text)
	s.xml.WriteString("\n  </Speak>")
	return s
}

// Break 添加停顿
func (s *SSMLBuilder) Break(strength string, seconds float64) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Break strength="%s" time="%.1fs"/>`, strength, seconds))
	return s
}

// Emphasis 添加重音
func (s *SSMLBuilder) Emphasis(text, level string) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Emphasis level="%s">`, level))
	s.xml.WriteString(text)
	s.xml.WriteString("</Emphasis>")
	return s
}

// Lang 切换语言
func (s *SSMLBuilder) Lang(text, lang string) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Lang xml:lang="%s">`, lang))
	s.xml.WriteString(text)
	s.xml.WriteString("</Lang>")
	return s
}

// Prosody 语速/音调/音量控制
func (s *SSMLBuilder) Prosody(text, rate, pitch, volume string) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Prosody rate="%s" pitch="%s" volume="%s">`, rate, pitch, volume))
	s.xml.WriteString(text)
	s.xml.WriteString("</Prosody>")
	return s
}

// Phoneme IPA音标
func (s *SSMLBuilder) Phoneme(text, alphabet, ph string) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Phoneme alphabet="%s" ph="%s">`, alphabet, ph))
	s.xml.WriteString(text)
	s.xml.WriteString("</Phoneme>")
	return s
}

// SayAs 特殊文本朗读
func (s *SSMLBuilder) SayAs(text, interpretAs, format string) *SSMLBuilder {
	attrs := fmt.Sprintf(` interpret-as="%s"`, interpretAs)
	if format != "" {
		attrs += fmt.Sprintf(` format="%s"`, format)
	}
	s.xml.WriteString(fmt.Sprintf(`\n  <Say-As%s>`, attrs))
	s.xml.WriteString(text)
	s.xml.WriteString("</Say-As>")
	return s
}

// Audio 播放音频
func (s *SSMLBuilder) Audio(url string, loop int) *SSMLBuilder {
	if loop > 1 {
		s.xml.WriteString(fmt.Sprintf(`\n  <Audio src="%s" loop="%d"/>`, url, loop))
	} else {
		s.xml.WriteString(fmt.Sprintf(`\n  <Audio src="%s"/>`, url))
	}
	return s
}

// Play 播放音频文件
func (s *SSMLBuilder) Play(url string) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Play>%s</Play>`, url))
	return s
}

// Wait 等待
func (s *SSMLBuilder) Wait(length int) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <Wait length="%d"/>`, length))
	return s
}

// GetDigits DTMF输入
func (s *SSMLBuilder) GetDigits(actionURL string, numDigits int, timeout int, retry int) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <GetInput action="%s" method="POST" inputType="dtmf" numDigits="%d" digitEndTimeout="%d" retries="%d">`,
		actionURL, numDigits, timeout, retry))
	return s
}

// EndGetDigits 结束GetDigits
func (s *SSMLBuilder) EndGetDigits() *SSMLBuilder {
	s.xml.WriteString("\n  </GetInput>")
	return s
}

// Hangup 挂断
func (s *SSMLBuilder) Hangup() *SSMLBuilder {
	s.xml.WriteString("\n  <Hangup/>")
	return s
}

// PreRecorded 添加预录音
func (s *SSMLBuilder) PreRecorded(url string) *SSMLBuilder {
	s.xml.WriteString(fmt.Sprintf(`\n  <PreRecorded url="%s"/>`, url))
	return s
}

// Concat 连接文本（用于模板变量替换）
func (s *SSMLBuilder) Concat(parts ...string) *SSMLBuilder {
	for _, part := range parts {
		s.xml.WriteString(part)
	}
	return s
}

// ToString 返回XML字符串
func (s *SSMLBuilder) ToString() string {
	return s.xml.String()
}

// SpeakOption SSML Speak标签选项
type SpeakOption struct {
	key   string
	value string
}

func WithVoice(voice string) SpeakOption {
	return SpeakOption{key: "voice", value: voice}
}

func WithLanguage(lang string) SpeakOption {
	return SpeakOption{key: "language", value: lang}
}

func WithLoop(loop int) SpeakOption {
	return SpeakOption{key: "loop", value: strconv.Itoa(loop)}
}

// SSMLBreakStrength 停顿强度常量
const (
	BreakStrengthNone   = "none"
	BreakStrengthXWeak  = "x-weak"
	BreakStrengthWeak   = "weak"
	BreakStrengthMedium = "medium"
	BreakStrengthStrong = "strong"
	BreakStrengthXStrong = "x-strong"
)

// SSMLEmphasisLevel 重音级别常量
const (
	EmphasisLevelReduced = "reduced"
	EmphasisLevelLow     = "low"
	EmphasisLevelMedium  = "moderate"
	EmphasisLevelStrong  = "strong"
)

// SSMLProsodyRate 语速常量
const (
	ProsodyRateXSlow  = "x-slow"
	ProsodyRateSlow   = "slow"
	ProsodyRateMedium = "medium"
	ProsodyRateFast   = "fast"
	ProsodyRateXFast  = "x-fast"
)

// VoiceConstants 语音常量
var VoiceConstants = map[string]string{
	"WOMAN":     "WOMAN",
	"MAN":       "MAN",
	"WOMAN_GB":  "en-GB-WOMAN",
	"MAN_GB":    "en-GB-MAN",
	"WOMAN_AU":  "en-AU-WOMAN",
	"WOMAN_CA":  "en-CA-WOMAN",
	"WOMAN_IN":  "en-IN-WOMAN",
	"WOMAN_ES":  "es-ES-WOMAN",
	"WOMAN_FR":  "fr-FR-WOMAN",
	"WOMAN_DE":  "de-DE-WOMAN",
	"WOMAN_IT":  "it-IT-WOMAN",
	"WOMAN_BR":  "pt-BR-WOMAN",
}

// 快速构建SSML的辅助函数
func BuildSimpleSSML(text string, voice string) string {
	b := NewSSMLBuilder()
	return b.Start().Speak(text, WithVoice(voice)).End()
}

// 构建带停顿的SSML
func BuildSSMLWithPause(parts []struct{Text string; Pause float64}, voice string) string {
	b := NewSSMLBuilder().Start()
	for _, part := range parts {
		b.Speak(part.Text, WithVoice(voice))
		if part.Pause > 0 {
			b.Break(BreakStrengthMedium, part.Pause)
		}
	}
	return b.End()
}

// 构建带强调的SSML
func BuildSSMLWithEmphasis(text string, emphasisLevel string, voice string) string {
	b := NewSSMLBuilder()
	b.Start()
	b.xml.WriteString("\n  <Speak voice=\"" + voice + "\">")
	b.xml.WriteString("\n    <Emphasis level=\"" + emphasisLevel + "\">")
	b.xml.WriteString(text)
	b.xml.WriteString("</Emphasis>")
	b.xml.WriteString("\n  </Speak>")
	return b.End()
}

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
	From                string
	To                  string
	AnswerURL           string
	HangupURL           string
	RingURL             string
	MachineDetectionURL string
	ErrorCallbackURL    string
	TimeLimit           int
	RingTimeout         int
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
		"from":                req.From,
		"to":                  req.To,
		"answer_url":          req.AnswerURL,
		"answer_method":       "POST",
		"hangup_url":          req.HangupURL,
		"hangup_method":       "POST",
		"ring_url":            req.RingURL,
		"ring_method":         "POST",
		"error_call_back_url": req.ErrorCallbackURL,
		"error_method":        "POST",
		"time_limit":          req.TimeLimit,
		"ring_timeout":        req.RingTimeout,
	}

	if req.MachineDetectionURL != "" {
		payload["machine_detection_url"] = req.MachineDetectionURL
		payload["machine_detection_method"] = "POST"
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
