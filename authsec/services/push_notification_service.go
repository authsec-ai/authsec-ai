package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const expoPushURL = "https://exp.host/--/api/v2/push/send"

// PushNotificationService handles sending push notifications via Expo Push Service
type PushNotificationService struct {
	httpClient *http.Client
}

// ExpoPushMessage represents a single push notification message for Expo
type ExpoPushMessage struct {
	To             string            `json:"to"`
	Title          string            `json:"title,omitempty"`
	Body           string            `json:"body,omitempty"`
	Data           map[string]string `json:"data,omitempty"`
	Sound          string            `json:"sound,omitempty"`
	TTL            int               `json:"ttl,omitempty"`
	Expiration     int               `json:"expiration,omitempty"`
	Priority       string            `json:"priority,omitempty"`
	CategoryId     string            `json:"categoryId,omitempty"`
	MutableContent bool              `json:"mutableContent,omitempty"`
}

// ExpoPushResponse represents the response from Expo Push API
type ExpoPushResponse struct {
	Data []ExpoPushTicket `json:"data"`
}

// ExpoPushTicket represents a single ticket in the response
type ExpoPushTicket struct {
	Status  string                 `json:"status"`
	ID      string                 `json:"id,omitempty"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewPushNotificationService creates a new push notification service using Expo Push API
func NewPushNotificationService() (*PushNotificationService, error) {
	return &PushNotificationService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendAuthRequest sends a push notification for authentication request via Expo Push Service
func (s *PushNotificationService) SendAuthRequest(
	deviceToken string,
	authReqID string,
	bindingMessage string,
	userEmail string,
) error {
	if s.httpClient == nil {
		return fmt.Errorf("push notification service not initialized")
	}

	title := "Authentication Request"
	body := bindingMessage
	if body == "" {
		body = "Approve login request"
	}

	// Expo Push message
	message := ExpoPushMessage{
		To:             deviceToken, // ExponentPushToken[xxx]
		Title:          title,
		Body:           body,
		Sound:          "default",
		Priority:       "high",
		CategoryId:     "auth_request", // For iOS notification actions
		MutableContent: true,
		Data: map[string]string{
			"auth_req_id":     authReqID,
			"type":            "auth_request",
			"binding_message": bindingMessage,
			"user_email":      userEmail,
		},
	}

	// Send single message
	return s.sendExpoPush([]ExpoPushMessage{message})
}

// SendMultipleAuthRequests sends push to multiple devices via Expo Push Service
func (s *PushNotificationService) SendMultipleAuthRequests(
	deviceTokens []string,
	authReqID string,
	bindingMessage string,
	userEmail string,
) error {
	if len(deviceTokens) == 0 {
		return fmt.Errorf("no device tokens provided")
	}

	title := "Authentication Request"
	body := bindingMessage
	if body == "" {
		body = "Approve login request"
	}

	// Build messages for all devices
	messages := make([]ExpoPushMessage, len(deviceTokens))
	for i, token := range deviceTokens {
		messages[i] = ExpoPushMessage{
			To:             token,
			Title:          title,
			Body:           body,
			Sound:          "default",
			Priority:       "high",
			CategoryId:     "auth_request",
			MutableContent: true,
			Data: map[string]string{
				"auth_req_id":     authReqID,
				"type":            "auth_request",
				"binding_message": bindingMessage,
				"user_email":      userEmail,
			},
		}
	}

	return s.sendExpoPush(messages)
}

// TestPushNotification sends a test push notification
func (s *PushNotificationService) TestPushNotification(deviceToken string) error {
	return s.SendAuthRequest(deviceToken, "test-auth-req", "Test push notification", "test@example.com")
}

// sendExpoPush sends messages to Expo Push API
func (s *PushNotificationService) sendExpoPush(messages []ExpoPushMessage) error {
	jsonData, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal push messages: %w", err)
	}

	req, err := http.NewRequest("POST", expoPushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expo push API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response to check for errors
	var pushResponse ExpoPushResponse
	if err := json.Unmarshal(respBody, &pushResponse); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check each ticket for errors
	for i, ticket := range pushResponse.Data {
		if ticket.Status == "error" {
			fmt.Printf("Push notification %d failed: %s\n", i, ticket.Message)
			if i == 0 {
				// Return error for the first message failure
				return fmt.Errorf("push notification failed: %s", ticket.Message)
			}
		} else {
			fmt.Printf("Successfully sent push notification %d: ticket=%s\n", i, ticket.ID)
		}
	}

	return nil
}
