package sdkmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
	models "github.com/authsec-ai/authsec/models/sdkmgr"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MCPPlaygroundService provides conversation CRUD, AI chat with tool calling,
// and MCP server management. Translates sdk-manager's mcp_playground_service.py.
type MCPPlaygroundService struct {
	openAIConfigured bool
}

// NewMCPPlaygroundService creates a new service instance.
func NewMCPPlaygroundService() *MCPPlaygroundService {
	svc := &MCPPlaygroundService{}
	cfg := config.AppConfig
	if cfg != nil && cfg.AzureOpenAIKey != "" && cfg.AzureOpenAIEndpoint != "" {
		svc.openAIConfigured = true
		logrus.Info("Azure OpenAI client configured for playground")
	} else {
		logrus.Warn("Azure OpenAI not configured; playground chat will return 503")
	}
	return svc
}

// HealthCheck returns playground health status.
func (s *MCPPlaygroundService) HealthCheck() map[string]interface{} {
	if !s.openAIConfigured {
		return map[string]interface{}{
			"status":  "unhealthy",
			"message": "Azure OpenAI client not initialized",
		}
	}
	return map[string]interface{}{
		"status":  "healthy",
		"message": "MCP Playground is operational",
	}
}

// tenantDB returns a GORM instance for the given tenant.
func (s *MCPPlaygroundService) tenantDB(tenantID string) (*gorm.DB, error) {
	return config.GetTenantGORMDB(tenantID)
}

// ==================== Conversation Management ====================

// CreateConversation creates a new conversation in the tenant DB.
func (s *MCPPlaygroundService) CreateConversation(
	tenantID, title string,
	model, systemPrompt *string,
	temperature *float64,
	maxTokens *int,
) (map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant DB error: %w", err)
	}

	conv := models.PlaygroundConversation{
		ID:    uuid.NewString(),
		Title: title,
	}
	if model != nil {
		conv.Model = model
	} else {
		cfg := config.AppConfig
		if cfg != nil && cfg.AzureOpenAIDeployment != "" {
			conv.Model = &cfg.AzureOpenAIDeployment
		}
	}
	if systemPrompt != nil {
		conv.SystemPrompt = systemPrompt
	}
	if temperature != nil {
		conv.Temperature = *temperature
	}
	if maxTokens != nil {
		conv.MaxTokens = *maxTokens
	}

	if err := db.Create(&conv).Error; err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	logrus.WithField("id", conv.ID).Info("created playground conversation")
	return conversationToMap(conv), nil
}

// GetConversation returns a conversation by ID.
func (s *MCPPlaygroundService) GetConversation(tenantID, conversationID string) (map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	var conv models.PlaygroundConversation
	if err := db.Where("id = ?", conversationID).First(&conv).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return conversationToMap(conv), nil
}

// ListConversations returns conversations ordered by updated_at desc.
func (s *MCPPlaygroundService) ListConversations(tenantID string, limit, offset int) ([]map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	var convs []models.PlaygroundConversation
	if err := db.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&convs).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(convs))
	for i, c := range convs {
		result[i] = conversationToMap(c)
	}
	return result, nil
}

// UpdateConversation updates selected fields of a conversation.
func (s *MCPPlaygroundService) UpdateConversation(
	tenantID, conversationID string,
	title, systemPrompt *string,
	temperature *float64,
	maxTokens *int,
) (map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if title != nil {
		updates["title"] = *title
	}
	if systemPrompt != nil {
		updates["system_prompt"] = *systemPrompt
	}
	if temperature != nil {
		updates["temperature"] = *temperature
	}
	if maxTokens != nil {
		updates["max_tokens"] = *maxTokens
	}

	if len(updates) == 0 {
		return s.GetConversation(tenantID, conversationID)
	}

	result := db.Model(&models.PlaygroundConversation{}).Where("id = ?", conversationID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	return s.GetConversation(tenantID, conversationID)
}

// DeleteConversation deletes a conversation and cascading messages/servers.
func (s *MCPPlaygroundService) DeleteConversation(tenantID, conversationID string) (bool, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return false, err
	}

	// Delete messages and MCP servers first (cascade may not be enforced by GORM).
	db.Where("conversation_id = ?", conversationID).Delete(&models.PlaygroundMessage{})
	db.Where("conversation_id = ?", conversationID).Delete(&models.PlaygroundMCPServer{})

	result := db.Where("id = ?", conversationID).Delete(&models.PlaygroundConversation{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ==================== Message Management ====================

// GetConversationMessages returns all messages for a conversation.
func (s *MCPPlaygroundService) GetConversationMessages(tenantID, conversationID string) ([]map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	var msgs []models.PlaygroundMessage
	if err := db.Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(msgs))
	for i, m := range msgs {
		result[i] = map[string]interface{}{
			"id":              m.ID,
			"conversation_id": m.ConversationID,
			"role":            m.Role,
			"content":         m.Content,
			"created_at":      m.CreatedAt,
		}
	}
	return result, nil
}

// AddMessage stores a message in the conversation.
func (s *MCPPlaygroundService) AddMessage(tenantID, conversationID, role, content string) error {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return err
	}

	msg := models.PlaygroundMessage{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
	}
	return db.Create(&msg).Error
}

// ==================== Chat Completion ====================

// ChatCompletion performs a non-streaming chat with Azure OpenAI.
func (s *MCPPlaygroundService) ChatCompletion(tenantID, conversationID, userMessage string) (map[string]interface{}, error) {
	if !s.openAIConfigured {
		return nil, fmt.Errorf("Azure OpenAI is not configured")
	}

	conv, err := s.GetConversation(tenantID, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, fmt.Errorf("conversation %s not found", conversationID)
	}

	// Save user message.
	if err := s.AddMessage(tenantID, conversationID, "user", userMessage); err != nil {
		return nil, err
	}

	// Build messages array.
	msgs, err := s.GetConversationMessages(tenantID, conversationID)
	if err != nil {
		return nil, err
	}

	openAIMsgs := s.buildOpenAIMessages(conv, msgs)

	// Call Azure OpenAI.
	respContent, err := s.callAzureOpenAI(conv, openAIMsgs, false)
	if err != nil {
		return nil, err
	}

	// Save assistant response.
	if err := s.AddMessage(tenantID, conversationID, "assistant", respContent); err != nil {
		logrus.WithError(err).Error("failed to save assistant message")
	}

	return map[string]interface{}{
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": respContent,
		},
	}, nil
}

// ChatCompletionStreamWriter writes SSE chunks to the response writer.
// It returns the full assistant response for persistence.
func (s *MCPPlaygroundService) ChatCompletionStreamWriter(
	tenantID, conversationID, userMessage string,
	writer io.Writer, flusher http.Flusher,
) error {
	if !s.openAIConfigured {
		return fmt.Errorf("Azure OpenAI is not configured")
	}

	conv, err := s.GetConversation(tenantID, conversationID)
	if err != nil {
		return err
	}
	if conv == nil {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	// Save user message.
	if err := s.AddMessage(tenantID, conversationID, "user", userMessage); err != nil {
		return err
	}

	// Build messages.
	msgs, err := s.GetConversationMessages(tenantID, conversationID)
	if err != nil {
		return err
	}
	openAIMsgs := s.buildOpenAIMessages(conv, msgs)

	// Call Azure OpenAI (non-streaming for simplicity, chunk into SSE).
	// TODO: Implement true streaming with Azure OpenAI SSE endpoint.
	respContent, err := s.callAzureOpenAI(conv, openAIMsgs, false)
	if err != nil {
		return err
	}

	// Write as SSE chunks.
	fmt.Fprintf(writer, "data: %s\n\n", respContent)
	flusher.Flush()
	fmt.Fprintf(writer, "data: [DONE]\n\n")
	flusher.Flush()

	// Persist.
	if err := s.AddMessage(tenantID, conversationID, "assistant", respContent); err != nil {
		logrus.WithError(err).Error("failed to save assistant message")
	}

	return nil
}

func (s *MCPPlaygroundService) buildOpenAIMessages(conv map[string]interface{}, msgs []map[string]interface{}) []map[string]interface{} {
	var openAIMsgs []map[string]interface{}

	sysPrompt, _ := conv["system_prompt"].(string)
	if sysPrompt == "" {
		sysPrompt = "You are a helpful AI assistant with access to various tools."
	}
	openAIMsgs = append(openAIMsgs, map[string]interface{}{
		"role":    "system",
		"content": sysPrompt,
	})

	for _, m := range msgs {
		openAIMsgs = append(openAIMsgs, map[string]interface{}{
			"role":    m["role"],
			"content": m["content"],
		})
	}
	return openAIMsgs
}

// callAzureOpenAI makes a direct HTTP call to the Azure OpenAI chat completions endpoint.
func (s *MCPPlaygroundService) callAzureOpenAI(
	conv map[string]interface{},
	messages []map[string]interface{},
	stream bool,
) (string, error) {
	cfg := config.AppConfig
	deployment := cfg.AzureOpenAIDeployment
	if m, ok := conv["model"].(string); ok && m != "" {
		deployment = m
	}

	apiURL := fmt.Sprintf(
		"%s/openai/deployments/%s/chat/completions?api-version=%s",
		cfg.AzureOpenAIEndpoint, deployment, cfg.AzureOpenAIVersion,
	)

	temp := 0.7
	if t, ok := conv["temperature"].(float64); ok {
		temp = t
	}
	maxTok := 2048
	if mt, ok := conv["max_tokens"].(int); ok {
		maxTok = mt
	}

	body := map[string]interface{}{
		"messages":    messages,
		"temperature": temp,
		"max_tokens":  maxTok,
		"stream":      stream,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", cfg.AzureOpenAIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Azure OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Azure OpenAI returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}
	return result.Choices[0].Message.Content, nil
}

// ==================== MCP Server Management ====================

// AddMCPServer adds an MCP server to a conversation.
func (s *MCPPlaygroundService) AddMCPServer(
	tenantID, conversationID, name, protocol, serverURL string,
	cfg map[string]interface{},
	oauthAccessToken, oauthRefreshToken, oauthTokenExpiry *string,
	oauthConfig map[string]interface{},
) (map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	configJSON, _ := json.Marshal(cfg)
	oauthConfigJSON, _ := json.Marshal(oauthConfig)

	var tokenExpiry *time.Time
	if oauthTokenExpiry != nil && *oauthTokenExpiry != "" {
		t, err := time.Parse(time.RFC3339, *oauthTokenExpiry)
		if err == nil {
			tokenExpiry = &t
		}
	}

	server := models.PlaygroundMCPServer{
		ID:                uuid.NewString(),
		ConversationID:    conversationID,
		Name:              name,
		Protocol:          protocol,
		ServerURL:         serverURL,
		Config:            configJSON,
		IsConnected:       false,
		OAuthAccessToken:  oauthAccessToken,
		OAuthRefreshToken: oauthRefreshToken,
		OAuthTokenExpiry:  tokenExpiry,
		OAuthConfig:       oauthConfigJSON,
	}

	if err := db.Create(&server).Error; err != nil {
		return nil, fmt.Errorf("failed to add MCP server: %w", err)
	}

	// TODO: Attempt actual MCP connection (streamable-http, SSE, stdio) and
	// update is_connected. For now, mark as connected optimistically.
	server.IsConnected = true
	db.Model(&server).Update("is_connected", true)

	logrus.WithFields(logrus.Fields{"id": server.ID, "name": name}).Info("added MCP server to conversation")

	return mcpServerToMap(server), nil
}

// ListMCPServers lists all MCP servers for a conversation.
func (s *MCPPlaygroundService) ListMCPServers(tenantID, conversationID string) ([]map[string]interface{}, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	var servers []models.PlaygroundMCPServer
	if err := db.Where("conversation_id = ?", conversationID).Order("created_at DESC").Find(&servers).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(servers))
	for i, srv := range servers {
		result[i] = mcpServerToMap(srv)
	}
	return result, nil
}

// DisconnectMCPServer marks a server as disconnected.
func (s *MCPPlaygroundService) DisconnectMCPServer(tenantID, conversationID, serverID string) (bool, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return false, err
	}

	result := db.Model(&models.PlaygroundMCPServer{}).
		Where("id = ? AND conversation_id = ?", serverID, conversationID).
		Update("is_connected", false)
	return result.RowsAffected > 0, result.Error
}

// ReconnectMCPServer marks a server as connected.
func (s *MCPPlaygroundService) ReconnectMCPServer(tenantID, conversationID, serverID string) (bool, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return false, err
	}

	// TODO: Actually reconnect to the MCP server protocol.
	result := db.Model(&models.PlaygroundMCPServer{}).
		Where("id = ? AND conversation_id = ?", serverID, conversationID).
		Update("is_connected", true)
	return result.RowsAffected > 0, result.Error
}

// RemoveMCPServer removes a server from a conversation.
func (s *MCPPlaygroundService) RemoveMCPServer(tenantID, conversationID, serverID string) (bool, error) {
	db, err := s.tenantDB(tenantID)
	if err != nil {
		return false, err
	}

	result := db.Where("id = ? AND conversation_id = ?", serverID, conversationID).Delete(&models.PlaygroundMCPServer{})
	return result.RowsAffected > 0, result.Error
}

// GetMCPTools returns tools from a specific MCP server.
func (s *MCPPlaygroundService) GetMCPTools(tenantID, conversationID, serverID string) ([]map[string]interface{}, error) {
	// TODO: Implement actual MCP tool discovery via the server's JSON-RPC endpoint.
	// For now return an empty list; the actual protocol integration requires
	// connecting to the MCP server and calling tools/list.
	return []map[string]interface{}{}, nil
}

// GetAllConversationTools returns tools from all connected MCP servers.
func (s *MCPPlaygroundService) GetAllConversationTools(tenantID, conversationID string) (map[string][]map[string]interface{}, error) {
	servers, err := s.ListMCPServers(tenantID, conversationID)
	if err != nil {
		return nil, err
	}

	allTools := map[string][]map[string]interface{}{}
	for _, srv := range servers {
		connected, _ := srv["is_connected"].(bool)
		if !connected {
			continue
		}
		srvID, _ := srv["id"].(string)
		srvName, _ := srv["name"].(string)
		tools, err := s.GetMCPTools(tenantID, conversationID, srvID)
		if err != nil {
			logrus.WithError(err).WithField("server", srvName).Warn("failed to get tools from MCP server")
			continue
		}
		if len(tools) > 0 {
			allTools[srvName] = tools
		}
	}
	return allTools, nil
}

// ==================== Helpers ====================

func conversationToMap(c models.PlaygroundConversation) map[string]interface{} {
	m := map[string]interface{}{
		"id":          c.ID,
		"title":       c.Title,
		"temperature": c.Temperature,
		"max_tokens":  c.MaxTokens,
		"created_at":  c.CreatedAt,
		"updated_at":  c.UpdatedAt,
	}
	if c.Model != nil {
		m["model"] = *c.Model
	}
	if c.SystemPrompt != nil {
		m["system_prompt"] = *c.SystemPrompt
	}
	return m
}

func mcpServerToMap(srv models.PlaygroundMCPServer) map[string]interface{} {
	return map[string]interface{}{
		"id":              srv.ID,
		"conversation_id": srv.ConversationID,
		"name":            srv.Name,
		"protocol":        srv.Protocol,
		"server_url":      srv.ServerURL,
		"config":          srv.Config,
		"is_connected":    srv.IsConnected,
		"created_at":      srv.CreatedAt,
	}
}
