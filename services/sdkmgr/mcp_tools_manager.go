package sdkmgr

import (
	"github.com/authsec-ai/authsec/config"
)

// ToolSchema represents an MCP tool definition returned to SDK clients.
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPToolsManager generates and manages MCP tool schemas.
type MCPToolsManager struct{}

// NewMCPToolsManager creates a new tools manager.
func NewMCPToolsManager() *MCPToolsManager {
	return &MCPToolsManager{}
}

// requireExplicitSessionID checks whether session_id should be a required field.
func requireExplicitSessionID() bool {
	if config.AppConfig != nil {
		return config.AppConfig.SDKRequireSessionID
	}
	return false
}

// GetOAuthTools returns the static OAuth management tool schemas.
func (m *MCPToolsManager) GetOAuthTools() []ToolSchema {
	return []ToolSchema{
		{
			Name:        "oauth_start",
			Description: "Start OAuth authentication flow",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"open_browser": map[string]interface{}{
						"type":        "boolean",
						"description": "Open authorization URL in local browser automatically",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "oauth_authenticate",
			Description: "Authenticate with JWT token",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"jwt_token":  map[string]interface{}{"type": "string"},
					"session_id": map[string]interface{}{"type": "string"},
					"expires_in": map[string]interface{}{"type": "number"},
				},
				"required": []string{"jwt_token", "session_id"},
			},
		},
		{
			Name:        "oauth_status",
			Description: "Check authentication status",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			Name:        "oauth_logout",
			Description: "Logout and invalidate session",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			Name:        "oauth_user_info",
			Description: "Get complete user information from JWT token (roles, groups, permissions, resources, scopes, and token metadata)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
		},
	}
}

// GenerateUserToolSchema creates a default tool schema for user's protected tools.
// Used as a fallback for tools that don't provide their own inputSchema.
func (m *MCPToolsManager) GenerateUserToolSchema(toolName string) ToolSchema {
	required := []string{}
	if requireExplicitSessionID() {
		required = []string{"session_id"}
	}

	return ToolSchema{
		Name:        toolName,
		Description: "Protected tool: " + toolName,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "OAuth session ID for authentication",
				},
			},
			"required": required,
		},
	}
}

// GenerateUserToolSchemaFromMetadata creates a tool schema from tool metadata.
// Supports both the old format (plain tool name string) and the new format
// (map with name, description, inputSchema, rbac).
func (m *MCPToolsManager) GenerateUserToolSchemaFromMetadata(toolMeta interface{}) ToolSchema {
	// Old format: plain string.
	if name, ok := toolMeta.(string); ok {
		return m.GenerateUserToolSchema(name)
	}

	// New format: map.
	meta, ok := toolMeta.(map[string]interface{})
	if !ok {
		return m.GenerateUserToolSchema("unknown")
	}

	toolName, _ := meta["name"].(string)
	if toolName == "" {
		toolName = "unknown"
	}

	// If inputSchema is provided, use it.
	if rawSchema, exists := meta["inputSchema"]; exists && rawSchema != nil {
		schema, ok := rawSchema.(map[string]interface{})
		if !ok {
			return m.GenerateUserToolSchema(toolName)
		}

		desc, _ := meta["description"].(string)
		if desc == "" {
			desc = "Protected tool: " + toolName
		}

		ts := ToolSchema{
			Name:        toolName,
			Description: desc,
			InputSchema: schema,
		}

		// Ensure properties map exists.
		props, _ := ts.InputSchema["properties"].(map[string]interface{})
		if props == nil {
			props = make(map[string]interface{})
			ts.InputSchema["properties"] = props
		}

		// Ensure session_id property exists.
		if _, ok := props["session_id"]; !ok {
			props["session_id"] = map[string]interface{}{
				"type":        "string",
				"description": "OAuth session ID for authentication",
			}
		}

		// Ensure required array exists; add session_id if configured.
		reqRaw, _ := ts.InputSchema["required"].([]interface{})
		var required []string
		for _, r := range reqRaw {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
		if requireExplicitSessionID() && !contains(required, "session_id") {
			required = append([]string{"session_id"}, required...)
		}
		ts.InputSchema["required"] = required

		return ts
	}

	// No inputSchema provided — use default with optional description override.
	base := m.GenerateUserToolSchema(toolName)
	if desc, ok := meta["description"].(string); ok && desc != "" {
		base.Description = desc
	}
	return base
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
