package bot

import (
	"testing"
	"time"
)

func TestParseCallbackData(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAction  string
		wantDataLen int
		wantErr     bool
	}{
		{
			name:        "Valid JSON with action and data",
			input:       `{"Action":"stats","Data":{"limit":"10"}}`,
			wantAction:  "stats",
			wantDataLen: 1,
			wantErr:     false,
		},
		{
			name:        "Valid JSON with empty data",
			input:       `{"Action":"menu","Data":{}}`,
			wantAction:  "menu",
			wantDataLen: 0,
			wantErr:     false,
		},
		{
			name:        "Valid JSON without data field",
			input:       `{"Action":"campaigns"}`,
			wantAction:  "campaigns",
			wantDataLen: 0,
			wantErr:     false,
		},
		{
			name:        "Invalid JSON",
			input:       `{invalid json}`,
			wantErr:     true,
		},
		{
			name:        "Empty string",
			input:       ``,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCallbackData(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCallbackData(%s) expected error, got nil", tt.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseCallbackData(%s) unexpected error: %v", tt.input, err)
				return
			}
			
			if result.Action != tt.wantAction {
				t.Errorf("ParseCallbackData(%s).Action = %s, want %s", tt.input, result.Action, tt.wantAction)
			}
			
			if len(result.Data) != tt.wantDataLen {
				t.Errorf("ParseCallbackData(%s).Data len = %d, want %d", tt.input, len(result.Data), tt.wantDataLen)
			}
		})
	}
}

func TestMarshalCallbackData(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		data      map[string]string
		wantValid bool
	}{
		{
			name:   "Marshal with data",
			action: "stats",
			data:   map[string]string{"limit": "10"},
		},
		{
			name:   "Marshal with multiple data fields",
			action: "campaign_detail",
			data: map[string]string{
				"id":    "123",
				"name":  "test",
				"type":  "detail",
			},
		},
		{
			name:      "Marshal with nil data",
			action:    "menu",
			data:      nil,
		},
		{
			name:      "Marshal with empty data",
			action:    "campaigns",
			data:      map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarshalCallbackData(tt.action, tt.data)
			
			// Verify it can be parsed back
			parsed, err := ParseCallbackData(result)
			if err != nil {
				t.Errorf("MarshalCallbackData result is not valid JSON: %v", err)
				return
			}
			
			if parsed.Action != tt.action {
				t.Errorf("MarshalCallbackData(%s, %v).Action = %s, want %s", tt.action, tt.data, parsed.Action, tt.action)
			}
		})
	}
}

func TestBotStateStruct(t *testing.T) {
	// Create a fresh state struct for each test to avoid global state pollution
	state := &BotStateStruct{
		states: make(map[int64]*UserState),
	}

	userID := int64(12345)
	
	// Test GetUserState when no state exists
	t.Run("GetUserState returns false when no state", func(t *testing.T) {
		_, exists := state.GetUserState(userID)
		if exists {
			t.Error("GetUserState should return false for non-existent user")
		}
	})

	// Test SetUserState
	t.Run("SetUserState stores state", func(t *testing.T) {
		testState := &UserState{
			Action: "test_action",
			Data:   map[string]interface{}{"key": "value"},
			Step:   1,
		}
		
		state.SetUserState(userID, testState)
		
		retrieved, exists := state.GetUserState(userID)
		if !exists {
			t.Error("GetUserState should return true after SetUserState")
		}
		if retrieved.Action != testState.Action {
			t.Errorf("Retrieved state Action = %s, want %s", retrieved.Action, testState.Action)
		}
		if retrieved.Step != testState.Step {
			t.Errorf("Retrieved state Step = %d, want %d", retrieved.Step, testState.Step)
		}
		
		// Verify timestamp was set
		if retrieved.Timestamp.IsZero() {
			t.Error("Timestamp should be set by SetUserState")
		}
	})

	// Test ClearUserState
	t.Run("ClearUserState removes state", func(t *testing.T) {
		state.ClearUserState(userID)
		
		_, exists := state.GetUserState(userID)
		if exists {
			t.Error("GetUserState should return false after ClearUserState")
		}
	})
}

func TestCallbackData(t *testing.T) {
	cb := CallbackData{
		Action: "test_action",
		Data:   map[string]string{"key": "value"},
	}

	if cb.Action != "test_action" {
		t.Errorf("CallbackData.Action = %s, want test_action", cb.Action)
	}

	if cb.Data["key"] != "value" {
		t.Errorf("CallbackData.Data[key] = %s, want value", cb.Data["key"])
	}
}

func TestUserState(t *testing.T) {
	now := time.Now()
	state := &UserState{
		Action:     "campaign_detail",
		Data:       map[string]interface{}{"id": float64(123)},
		Step:       2,
		MessageID:  100,
		CallbackID: "callback_123",
		Timestamp:  now,
	}

	if state.Action != "campaign_detail" {
		t.Errorf("UserState.Action = %s, want campaign_detail", state.Action)
	}

	if state.Step != 2 {
		t.Errorf("UserState.Step = %d, want 2", state.Step)
	}

	if state.MessageID != 100 {
		t.Errorf("UserState.MessageID = %d, want 100", state.MessageID)
	}

	if state.CallbackID != "callback_123" {
		t.Errorf("UserState.CallbackID = %s, want callback_123", state.CallbackID)
	}

	if !state.Timestamp.Equal(now) {
		t.Errorf("UserState.Timestamp = %v, want %v", state.Timestamp, now)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 50, 50},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}
