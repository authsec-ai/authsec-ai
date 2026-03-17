-- SDK-Manager migration: Playground tables in tenant databases.
-- Conversations, messages, and MCP server connections for the AI playground.

CREATE TABLE IF NOT EXISTS playground_conversations (
    id              VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title           VARCHAR(255) NOT NULL DEFAULT 'New Conversation',
    model           VARCHAR(100),
    system_prompt   TEXT,
    temperature     NUMERIC(3,2) DEFAULT 0.7,
    max_tokens      INTEGER DEFAULT 2048,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playground_messages (
    id              VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    conversation_id VARCHAR(36) NOT NULL REFERENCES playground_conversations(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,
    content         TEXT NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_playground_messages_conversation
    ON playground_messages(conversation_id, created_at ASC);

CREATE TABLE IF NOT EXISTS playground_mcp_servers (
    id                  VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    conversation_id     VARCHAR(36) NOT NULL REFERENCES playground_conversations(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    protocol            VARCHAR(20) NOT NULL,
    server_url          TEXT NOT NULL,
    config              JSONB DEFAULT '{}',
    is_connected        BOOLEAN DEFAULT false,
    oauth_access_token  TEXT,
    oauth_refresh_token TEXT,
    oauth_token_expiry  TIMESTAMP WITH TIME ZONE,
    oauth_config        JSONB DEFAULT '{}',
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_playground_mcp_servers_conversation
    ON playground_mcp_servers(conversation_id);
