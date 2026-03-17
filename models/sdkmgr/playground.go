package sdkmgr

import (
	"time"

	"gorm.io/datatypes"
)

// PlaygroundConversation represents an AI playground conversation.
type PlaygroundConversation struct {
	ID           string    `gorm:"column:id;primaryKey" json:"id"`
	Title        string    `gorm:"column:title" json:"title"`
	Model        *string   `gorm:"column:model" json:"model"`
	SystemPrompt *string   `gorm:"column:system_prompt" json:"system_prompt"`
	Temperature  float64   `gorm:"column:temperature;default:0.7" json:"temperature"`
	MaxTokens    int       `gorm:"column:max_tokens;default:2048" json:"max_tokens"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PlaygroundConversation) TableName() string { return "playground_conversations" }

// PlaygroundMessage represents a message in a conversation.
type PlaygroundMessage struct {
	ID             string    `gorm:"column:id;primaryKey" json:"id"`
	ConversationID string    `gorm:"column:conversation_id" json:"conversation_id"`
	Role           string    `gorm:"column:role" json:"role"`
	Content        string    `gorm:"column:content" json:"content"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PlaygroundMessage) TableName() string { return "playground_messages" }

// PlaygroundMCPServer represents an MCP server connected to a conversation.
type PlaygroundMCPServer struct {
	ID                string         `gorm:"column:id;primaryKey" json:"id"`
	ConversationID    string         `gorm:"column:conversation_id" json:"conversation_id"`
	Name              string         `gorm:"column:name" json:"name"`
	Protocol          string         `gorm:"column:protocol" json:"protocol"`
	ServerURL         string         `gorm:"column:server_url" json:"server_url"`
	Config            datatypes.JSON `gorm:"column:config" json:"config"`
	IsConnected       bool           `gorm:"column:is_connected;default:false" json:"is_connected"`
	OAuthAccessToken  *string        `gorm:"column:oauth_access_token" json:"-"`
	OAuthRefreshToken *string        `gorm:"column:oauth_refresh_token" json:"-"`
	OAuthTokenExpiry  *time.Time     `gorm:"column:oauth_token_expiry" json:"oauth_token_expiry,omitempty"`
	OAuthConfig       datatypes.JSON `gorm:"column:oauth_config" json:"-"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PlaygroundMCPServer) TableName() string { return "playground_mcp_servers" }
