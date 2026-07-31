package voice

import (
	"fmt"
	"net/url"
)

// Provider type constants
const (
	ProviderPlivo  = "plivo"
	ProviderTelnyx = "telnyx"
)

// VoiceProvider interface - unified interface for all voice providers
type VoiceProvider interface {
	// MakeCall initiates a call with the given parameters
	MakeCall(req CallRequest) (*CallResponse, error)
	
	// HangupCall terminates an active call
	HangupCall(uuid string) (*PlivoResponse, error)
	
	// GetCallStatus retrieves the current status of a call
	GetCallStatus(uuid string) (*CallStatusResponse, error)
	
	// SendSMS sends an SMS message
	SendSMS(from, to, text string) (*PlivoResponse, error)
	
	// GetNumbers returns the list of phone numbers associated with the account
	GetNumbers() ([]map[string]interface{}, error)
	
	// ProviderName returns the name of the provider
	ProviderName() string
	
	// BuildCallResponseXML generates the XML response for call setup
	BuildCallResponseXML(greeting, actionURL, confirmation, holdMusic, voice, language string, loops int) string
	
	// BuildDTMFXML generates XML for DTMF collection
	BuildDTMFXML(message, holdMusic, voice, language string) string
	
	// BuildHangupXML generates XML for call termination
	BuildHangupXML(message string) string
	
	// BuildRingXML generates XML for ring state
	BuildRingXML() string
	
	// BuildMachineXML generates XML for answering machine detection
	BuildMachineXML() string
}

// ProviderFactory creates the appropriate voice provider based on config
type ProviderFactory struct{}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a voice provider based on the provider type
func (f *ProviderFactory) CreateProvider(providerType, authID, authToken, number string) (VoiceProvider, error) {
	switch providerType {
	case ProviderTelnyx:
		return &TelnyxAdapter{
			client: NewTelnyxClient(authID, number),
			number: number,
		}, nil
	case ProviderPlivo:
		return &PlivoAdapter{
			client: NewPlivoClient(authID, authToken, number),
			number: number,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported voice provider: %s (supported: telnyx, plivo)", providerType)
	}
}

// PlivoAdapter wraps PlivoClient to implement VoiceProvider interface
type PlivoAdapter struct {
	client *PlivoClient
	number string
}

func (p *PlivoAdapter) MakeCall(req CallRequest) (*CallResponse, error) {
	return p.client.MakeCall(req)
}

func (p *PlivoAdapter) HangupCall(uuid string) (*PlivoResponse, error) {
	return p.client.HangupCall(uuid)
}

func (p *PlivoAdapter) GetCallStatus(uuid string) (*CallStatusResponse, error) {
	return p.client.GetCallStatus(uuid)
}

func (p *PlivoAdapter) SendSMS(from, to, text string) (*PlivoResponse, error) {
	return p.client.SendSMS(from, to, text)
}

func (p *PlivoAdapter) GetNumbers() ([]map[string]interface{}, error) {
	return p.client.GetNumbers()
}

func (p *PlivoAdapter) ProviderName() string {
	return "plivo"
}

func (p *PlivoAdapter) BuildCallResponseXML(greeting, actionURL, confirmation, holdMusic, voice, language string, loops int) string {
	return BuildXMLResponse(greeting, actionURL, confirmation, holdMusic, voice, language, loops)
}

func (p *PlivoAdapter) BuildDTMFXML(message, holdMusic, voice, language string) string {
	return BuildDTMFXML(message, holdMusic, voice, language)
}

func (p *PlivoAdapter) BuildHangupXML(message string) string {
	return BuildHangupXML(message)
}

func (p *PlivoAdapter) BuildRingXML() string {
	return BuildRingXML()
}

func (p *PlivoAdapter) BuildMachineXML() string {
	return BuildMachineXML()
}

// TelnyxAdapter wraps TelnyxClient to implement VoiceProvider interface
type TelnyxAdapter struct {
	client *TelnyxClient
	number string
}

func (t *TelnyxAdapter) MakeCall(req CallRequest) (*CallResponse, error) {
	// Convert unified CallRequest to Telnyx-specific format
	telnyxReq := TelnyxCallRequest{
		From:      req.From,
		To:        req.To,
		AnswerURL: req.AnswerURL,
		Timeout:   req.RingTimeout,
	}

	resp, err := t.client.MakeCall(telnyxReq)
	if err != nil {
		return nil, err
	}

	// Convert Telnyx response to unified format
	return &CallResponse{
		RequestUUID: resp.Data.ID,
		UUID:        resp.Data.ID,
		Message:     resp.Data.State,
	}, nil
}

func (t *TelnyxAdapter) HangupCall(uuid string) (*PlivoResponse, error) {
	err := t.client.HangupCall(uuid)
	if err != nil {
		return nil, err
	}
	return &PlivoResponse{Message: "hangup"}, nil
}

func (t *TelnyxAdapter) GetCallStatus(uuid string) (*CallStatusResponse, error) {
	resp, err := t.client.GetCallStatus(uuid)
	if err != nil {
		return nil, err
	}

	// Convert Telnyx status to unified format
	return &CallStatusResponse{
		CallUUID:  resp.Data.ID,
		CallState: resp.Data.State,
		From:      resp.Data.From,
		To:         resp.Data.To,
		Duration:   resp.Data.Duration,
	}, nil
}

func (t *TelnyxAdapter) SendSMS(from, to, text string) (*PlivoResponse, error) {
	resp, err := t.client.SendSMS(from, to, text)
	if err != nil {
		return nil, err
	}

	return &PlivoResponse{
		APIID:    resp.Data.ID,
		Message:  resp.Data.Type,
	}, nil
}

func (t *TelnyxAdapter) GetNumbers() ([]map[string]interface{}, error) {
	return t.client.GetNumbers()
}

func (t *TelnyxAdapter) ProviderName() string {
	return "telnyx"
}

func (t *TelnyxAdapter) BuildCallResponseXML(greeting, actionURL, confirmation, holdMusic, voice, language string, loops int) string {
	return BuildTelnyxResponse(greeting, actionURL, confirmation, holdMusic, voice, language, loops)
}

func (t *TelnyxAdapter) BuildDTMFXML(message, holdMusic, voice, language string) string {
	return BuildTelnyxDTMFResponse(message, holdMusic, voice, language)
}

func (t *TelnyxAdapter) BuildHangupXML(message string) string {
	return BuildTelnyxHangupResponse(message)
}

func (t *TelnyxAdapter) BuildRingXML() string {
	return BuildTelnyxRingResponse()
}

func (t *TelnyxAdapter) BuildMachineXML() string {
	return BuildTelnyxMachineResponse()
}

// URLEncode helper for Twilio form encoding
func URLEncode(s string) string {
	return url.QueryEscape(s)
}
