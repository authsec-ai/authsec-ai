package sdkmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/sirupsen/logrus"
)

// ── AuthSec internal SDK for CIBA / TOTP ─────────────────────────────────

// authSecSDK wraps HTTP calls to the authsec CIBA and TOTP endpoints.
// When client_id is non-empty the tenant flow is used; otherwise admin.
type authSecSDK struct {
	baseURL     string
	clientID    string
	mu          sync.Mutex
	activePolls map[string]bool // email → cancelled
	retryCounts map[string]int  // email → remaining TOTP attempts
}

func newAuthSecSDK(clientID string) *authSecSDK {
	base := config.GetConfig().HydraPublicURL
	if base == "" {
		base = "https://dev.api.authsec.dev"
	}
	return &authSecSDK{
		baseURL:     base,
		clientID:    clientID,
		activePolls: make(map[string]bool),
		retryCounts: make(map[string]int),
	}
}

func (a *authSecSDK) retryCount(email string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.retryCounts[email]
}

func (a *authSecSDK) initiateAppApproval(email string) map[string]interface{} {
	a.mu.Lock()
	a.retryCounts[email] = 0
	if _, ok := a.activePolls[email]; ok {
		a.activePolls[email] = true // cancel any running poll
	}
	a.mu.Unlock()

	var endpoint string
	payload := map[string]string{}

	if a.clientID != "" {
		endpoint = a.baseURL + "/uflow/auth/tenant/ciba/initiate"
		payload["client_id"] = a.clientID
		payload["email"] = email
		payload["binding_message"] = "Authentication requested via Voice SDK"
	} else {
		endpoint = a.baseURL + "/uflow/auth/ciba/initiate"
		payload["login_hint"] = email
		payload["binding_message"] = "Authentication requested via Voice SDK"
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		logrus.Errorf("CIBA Initiate error: %v", err)
		return map[string]interface{}{"error": err.Error(), "auth_req_id": nil}
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return map[string]interface{}{"error": "decode error", "auth_req_id": nil}
	}
	return result
}

func (a *authSecSDK) verifyTOTP(email, code string) map[string]interface{} {
	a.mu.Lock()
	if _, ok := a.retryCounts[email]; !ok {
		a.retryCounts[email] = 0
	}
	if a.retryCounts[email] >= 3 {
		a.mu.Unlock()
		return map[string]interface{}{"success": false, "error": "too_many_retries", "remaining": 0}
	}
	a.mu.Unlock()

	var endpoint string
	payload := map[string]string{}

	if a.clientID != "" {
		endpoint = a.baseURL + "/uflow/auth/tenant/totp/login"
		payload["client_id"] = a.clientID
		payload["email"] = email
		payload["totp_code"] = code
	} else {
		endpoint = a.baseURL + "/uflow/auth/totp/login"
		payload["email"] = email
		payload["totp_code"] = code
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		a.mu.Lock()
		a.retryCounts[email]++
		remaining := 3 - a.retryCounts[email]
		a.mu.Unlock()
		return map[string]interface{}{"success": false, "error": err.Error(), "remaining": remaining}
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&resData)

	token, _ := resData["token"].(string)
	if token == "" {
		token, _ = resData["access_token"].(string)
	}

	if token != "" || resData["success"] == true {
		a.mu.Lock()
		a.retryCounts[email] = 0
		a.mu.Unlock()
		resData["success"] = true
		resData["token"] = token
		resData["remaining"] = 3
		return resData
	}

	a.mu.Lock()
	a.retryCounts[email]++
	remaining := 3 - a.retryCounts[email]
	a.mu.Unlock()
	return map[string]interface{}{"success": false, "error": "invalid_code", "remaining": remaining}
}

// pollForApproval polls with a short timeout (used by check_status tool, timeout=2s).
func (a *authSecSDK) pollForApproval(email, authReqID string, timeout int) map[string]interface{} {
	a.mu.Lock()
	a.activePolls[email] = false
	a.mu.Unlock()

	var endpoint string
	payload := map[string]string{}

	if a.clientID != "" {
		endpoint = a.baseURL + "/uflow/auth/tenant/ciba/token"
		payload["client_id"] = a.clientID
		payload["auth_req_id"] = authReqID
	} else {
		endpoint = a.baseURL + "/uflow/auth/ciba/token"
		payload["auth_req_id"] = authReqID
	}

	body, _ := json.Marshal(payload)
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	interval := 5 * time.Second

	client := &http.Client{Timeout: 10 * time.Second}

	for time.Now().Before(deadline) {
		a.mu.Lock()
		cancelled := a.activePolls[email]
		a.mu.Unlock()
		if cancelled {
			return map[string]interface{}{"status": "cancelled"}
		}

		resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body)) //nolint:gosec
		if err != nil {
			return map[string]interface{}{"status": "error", "error": err.Error()}
		}
		var data map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()

		token, _ := data["access_token"].(string)
		if token == "" {
			token, _ = data["token"].(string)
		}
		if token != "" {
			return map[string]interface{}{"status": "approved", "token": token}
		}

		errVal, _ := data["error"].(string)
		if errVal == "access_denied" || errVal == "expired_token" {
			return map[string]interface{}{"status": errVal}
		}

		if timeout <= 2 {
			break
		}
		time.Sleep(interval)
	}
	return map[string]interface{}{"status": "timeout"}
}

func (a *authSecSDK) cancelApproval(email string) {
	a.mu.Lock()
	a.activePolls[email] = true
	a.retryCounts[email] = 0
	a.mu.Unlock()
}

// ── LLM tool definitions ─────────────────────────────────────────────────

type llmTool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

func voiceTools() []llmTool {
	return []llmTool{
		{Type: "function", Function: map[string]interface{}{
			"name": "initiate_app_approval", "description": "Trigger a push notification to the user's mobile app.",
			"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: map[string]interface{}{
			"name": "verify_totp", "description": "Verify a 6-digit TOTP code.",
			"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"code": map[string]string{"type": "string"}}},
		}},
		{Type: "function", Function: map[string]interface{}{
			"name": "check_status", "description": "Check if app approval is done.",
			"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"auth_req_id": map[string]string{"type": "string"}}},
		}},
		{Type: "function", Function: map[string]interface{}{
			"name": "cancel_auth", "description": "Cancel the current authentication flow.",
			"parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
	}
}

// ── VoiceClientService ───────────────────────────────────────────────────

// VoiceClientService handles LLM-based voice/chat authentication interactions
// using Azure OpenAI for conversation and CIBA/TOTP for actual auth.
type VoiceClientService struct {
	endpoint      string
	deployment    string
	apiVersion    string
	apiKey        string
	ttsDeployment string
	authSDK       *authSecSDK

	mu             sync.Mutex
	messageHistory map[string][]map[string]interface{} // email → messages
}

// NewVoiceClientService creates a new voice client service.
func NewVoiceClientService() *VoiceClientService {
	cfg := config.GetConfig()
	return &VoiceClientService{
		endpoint:       cfg.AzureOpenAIEndpoint,
		deployment:     cfg.AzureOpenAIDeployment,
		apiVersion:     cfg.AzureOpenAIVersion,
		apiKey:         cfg.AzureOpenAIKey,
		ttsDeployment:  cfg.AzureOpenAITTSDeployment,
		authSDK:        newAuthSecSDK(""), // default admin flow
		messageHistory: make(map[string][]map[string]interface{}),
	}
}

func (s *VoiceClientService) configured() bool {
	return s.apiKey != "" && s.endpoint != ""
}

func (s *VoiceClientService) systemPrompt(email string) string {
	retryCount := s.authSDK.retryCount(email)
	return fmt.Sprintf(`You are the AuthSec Smart Voice Assistant. Your goal is to guide the user through authentication.
User Email: %s

### AUTHENTICATION INTENT RECOGNITION:
Recognize these phrases as authentication requests:
- "authenticate me", "log me in", "sign me in", "login", "sign in"
- "log me in please", "can you log me in", "i want to log in"
- "authenticate", "let me in", "access my account"

When ANY of these are detected, immediately proceed to method selection.

### MANDATORY AUTHENTICATION FLOW:
1. **STAGE 1: Greeting & Discovery** - If the conversation is just starting, use the exact greeting: "Welcome back! I'm your AuthSec assistant. To authenticate say authenticate me to authsec or help for available skills or commands".
2. **STAGE 2: Intent Validation** - If user says ANY authentication phrase, acknowledge and proceed immediately to Method Selection.
3. **STAGE 3: Method Selection** - Ask: "Great! Would you like to approve via the AuthSec app or use a TOTP code?"
4. **STAGE 4: Verification** - Call tools based on choice.

**CONVERSATIONAL RULES:**
- DO NOT repeat the Stage 1 greeting if the user has already stated their intent.
- Be smart: If the user says any authentication phrase in their first message, skip the greeting and go straight to method selection.
- Be concise and natural.

### HELP COMMAND:
If user asks for 'help' or 'commands' or 'skills', explain:
- "I can help you authenticate to the AuthSec platform using either a push notification to your mobile app or a 6-digit TOTP code."
- Mention that no other functionalities are currently configured.

### RETRY LIMITS:
- Users have exactly 3 attempts for TOTP.
- If they fail 3 times, you MUST tell them to restart the process and stop verifying codes until they start a new 'authenticate' intent.
- Current TOTP retry count: %d

Intents you can handle via tools:
1. 'initiate_app_approval': For mobile app/push.
2. 'verify_totp': For 6-digit TOTP code.
3. 'check_status': To check if app approval is done.
4. 'cancel_auth': If the user explicitly asks to 'cancel', 'stop', or 'abort' an ongoing authentication request.
`, email, retryCount)
}

// getHistory returns message history for an email, initializing if needed.
func (s *VoiceClientService) getHistory(email string) []map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.messageHistory[email]; !ok {
		s.messageHistory[email] = []map[string]interface{}{
			{"role": "system", "content": s.systemPrompt(email)},
		}
	}
	return s.messageHistory[email]
}

func (s *VoiceClientService) setHistory(email string, h []map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageHistory[email] = h
}

func (s *VoiceClientService) resetHistory(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageHistory[email] = []map[string]interface{}{
		{"role": "system", "content": s.systemPrompt(email)},
	}
}

// trimHistory trims excess messages while keeping the system prompt and
// not splitting tool_calls from their tool responses.
func trimHistory(msgs []map[string]interface{}) []map[string]interface{} {
	if len(msgs) <= 11 {
		return msgs
	}
	sys := msgs[0]
	trimmed := make([]map[string]interface{}, len(msgs[len(msgs)-10:]))
	copy(trimmed, msgs[len(msgs)-10:])

	for len(trimmed) > 0 && trimmed[0]["role"] == "tool" {
		trimmed = trimmed[1:]
	}
	if len(trimmed) > 0 && trimmed[0]["role"] == "assistant" {
		if _, ok := trimmed[0]["tool_calls"]; ok {
			if len(trimmed) < 2 || trimmed[1]["role"] != "tool" {
				trimmed = trimmed[1:]
			}
		}
	}
	return append([]map[string]interface{}{sys}, trimmed...)
}

// Interact processes a user message through the voice agent.
func (s *VoiceClientService) Interact(userInput, email, authReqID, clientID string) (map[string]interface{}, error) {
	if !s.configured() {
		return map[string]interface{}{
			"reply":  "Voice agent is not configured. Azure OpenAI credentials are missing.",
			"action": nil,
		}, nil
	}

	// Select SDK based on client_id
	sdk := s.authSDK
	if clientID != "" && clientID != s.authSDK.clientID {
		sdk = newAuthSecSDK(clientID)
	}

	history := s.getHistory(email)
	history = append(history, map[string]interface{}{"role": "user", "content": userInput})
	history = trimHistory(history)
	// Update system prompt with latest retry count
	history[0]["content"] = s.systemPrompt(email)

	// Call Azure OpenAI chat completion
	chatResp, err := s.callChatCompletion(history)
	if err != nil {
		logrus.Errorf("OpenAI API error: %v", err)
		s.resetHistory(email)
		return map[string]interface{}{
			"reply":  "I encountered an issue. Let's start fresh. Say 'authenticate me' when you're ready.",
			"action": nil,
		}, nil
	}

	aiMsg := chatResp["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})
	history = append(history, aiMsg)

	replyText, _ := aiMsg["content"].(string)
	if replyText == "" {
		replyText = "Processing..."
	}
	var action map[string]interface{}

	// Handle tool calls
	if toolCalls, ok := aiMsg["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		for _, tcRaw := range toolCalls {
			tc := tcRaw.(map[string]interface{})
			tcID, _ := tc["id"].(string)
			fn := tc["function"].(map[string]interface{})
			funcName, _ := fn["name"].(string)
			argsStr, _ := fn["arguments"].(string)
			var args map[string]string
			json.Unmarshal([]byte(argsStr), &args)

			var toolRes map[string]interface{}

			switch funcName {
			case "initiate_app_approval":
				res := sdk.initiateAppApproval(email)
				aid, _ := res["auth_req_id"].(string)
				errStr, _ := res["error"].(string)

				if errStr == "no_device_registered" {
					replyText = "It looks like you don't have the AuthSec app set up yet. Would you like to authenticate using a TOTP code instead? Just provide your 6-digit code."
					action = nil
					toolRes = map[string]interface{}{"status": "no_device", "error": errStr}
				} else if aid != "" {
					authReqID = aid
					action = map[string]interface{}{"type": "CIBA_STARTED", "auth_req_id": aid}
					replyText = "I've sent a notification to your AuthSec app. Please approve it."
					toolRes = map[string]interface{}{"status": "initiated", "id": aid}
				} else {
					desc, _ := res["error_description"].(string)
					if desc == "" {
						desc = "Unknown error"
					}
					replyText = fmt.Sprintf("Sorry, I couldn't send the push notification. Error: %s. Would you like to try TOTP instead?", desc)
					action = nil
					toolRes = map[string]interface{}{"status": "failed", "error": errStr}
				}

			case "verify_totp":
				res := sdk.verifyTOTP(email, args["code"])
				if res["success"] == true || res["token"] != nil {
					token, _ := res["token"].(string)
					replyText = fmt.Sprintf("Success! You are authorized. Your token is: %s", token)
					action = map[string]interface{}{"type": "AUTH_SUCCESS", "token": token}
					toolRes = map[string]interface{}{"status": "success", "token": token}
				} else {
					remaining, _ := res["remaining"].(int)
					if remaining <= 0 {
						replyText = "Too many attempts. Please restart the process."
					} else {
						replyText = fmt.Sprintf("Invalid code. You have %d attempts left.", remaining)
					}
					toolRes = map[string]interface{}{"status": "failed", "remaining": remaining}
				}

			case "check_status":
				reqID := args["auth_req_id"]
				if reqID == "" {
					reqID = authReqID
				}
				if reqID != "" {
					res := sdk.pollForApproval(email, reqID, 2)
					if res["status"] == "approved" {
						token, _ := res["token"].(string)
						replyText = fmt.Sprintf("Yes, it's approved! You are signed in. Your token is: %s", token)
						action = map[string]interface{}{"type": "AUTH_SUCCESS", "token": token}
					} else {
						replyText = "It's still pending on your device."
					}
					toolRes = res
				} else {
					toolRes = map[string]interface{}{"status": "none"}
				}

			case "cancel_auth":
				sdk.cancelApproval(email)
				replyText = "Authentication cancelled. Say 'authenticate me' to start over."
				action = map[string]interface{}{"type": "AUTH_CANCELLED"}
				toolRes = map[string]interface{}{"status": "cancelled"}
			}

			resJSON, _ := json.Marshal(toolRes)
			history = append(history, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": tcID,
				"content":      string(resJSON),
			})
		}
	}

	// Clear history on successful auth
	if action != nil && action["type"] == "AUTH_SUCCESS" {
		s.resetHistory(email)
	} else {
		s.setHistory(email, history)
	}

	return map[string]interface{}{
		"reply":  replyText,
		"action": action,
	}, nil
}

// PollStatus polls for CIBA approval status (short timeout).
func (s *VoiceClientService) PollStatus(authReqID, email, clientID string) map[string]interface{} {
	sdk := s.authSDK
	if clientID != "" && clientID != s.authSDK.clientID {
		sdk = newAuthSecSDK(clientID)
	}
	return sdk.pollForApproval(email, authReqID, 2)
}

// GenerateSpeech produces audio bytes via Azure OpenAI TTS.
func (s *VoiceClientService) GenerateSpeech(text, voice string) ([]byte, error) {
	if !s.configured() {
		return nil, fmt.Errorf("Azure OpenAI not configured")
	}
	if voice == "" {
		voice = "nova"
	}
	deploy := s.ttsDeployment
	if deploy == "" {
		deploy = "tts"
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/audio/speech?api-version=%s",
		s.endpoint, deploy, s.apiVersion)

	payload := map[string]string{
		"model": deploy,
		"voice": voice,
		"input": text,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS error %d: %s", resp.StatusCode, string(respBody))
	}
	return io.ReadAll(resp.Body)
}

// callChatCompletion calls Azure OpenAI chat completions via REST.
func (s *VoiceClientService) callChatCompletion(messages []map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		s.endpoint, s.deployment, s.apiVersion)

	payload := map[string]interface{}{
		"messages":    messages,
		"tools":       voiceTools(),
		"tool_choice": "auto",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat completion error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
