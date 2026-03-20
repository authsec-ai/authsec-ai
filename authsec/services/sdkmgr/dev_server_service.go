package sdkmgr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DevServerInfo tracks a running dev MCP server subprocess.
type DevServerInfo struct {
	ServerID       string
	Process        *exec.Cmd
	TempFile       string
	PID            int
	LastActivity   time.Time
	TenantID       string
	ConversationID string
	Tools          []map[string]interface{}

	stdin  *bufio.Writer
	stdout *bufio.Reader
}

// DevServerService manages subprocess execution of user-provided MCP server code.
type DevServerService struct {
	mu              sync.Mutex
	runningServers  map[string]*DevServerInfo // conversationID → info
	inactiveTimeout time.Duration
	playgroundSvc   *MCPPlaygroundService
}

// NewDevServerService creates a new dev server service.
func NewDevServerService(playgroundSvc *MCPPlaygroundService) *DevServerService {
	svc := &DevServerService{
		runningServers:  make(map[string]*DevServerInfo),
		inactiveTimeout: 10 * time.Minute,
		playgroundSvc:   playgroundSvc,
	}
	go svc.cleanupLoop()
	return svc
}

// StartServer starts an MCP server from user-supplied Python code.
func (s *DevServerService) StartServer(code, conversationID, tenantID string) map[string]interface{} {
	s.mu.Lock()
	// Stop existing server for this conversation if any.
	if info, ok := s.runningServers[conversationID]; ok {
		s.mu.Unlock()
		s.stopLocked(info)
		s.mu.Lock()
		delete(s.runningServers, conversationID)
	}
	s.mu.Unlock()

	serverID := fmt.Sprintf("dev_%s", uuid.New().String()[:8])

	// Write code to a temp file.
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("mcp_dev_%s_*.py", serverID))
	if err != nil {
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("temp file: %v", err)}
	}
	fullCode := generateServerCode(code, tenantID)
	if _, err := tmpFile.WriteString(fullCode); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("write: %v", err)}
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0o755)

	// Find python3 executable
	pythonExe := "python3"
	if p, err := exec.LookPath("python3"); err == nil {
		pythonExe = p
	}

	cmd := exec.Command(pythonExe, "-u", tmpFile.Name())
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		os.Remove(tmpFile.Name())
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("stdin pipe: %v", err)}
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		os.Remove(tmpFile.Name())
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("stdout pipe: %v", err)}
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		os.Remove(tmpFile.Name())
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("start: %v", err)}
	}

	// Give the subprocess a moment to boot.
	time.Sleep(500 * time.Millisecond)

	// Check if process exited immediately.
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		os.Remove(tmpFile.Name())
		return map[string]interface{}{"success": false, "error": "process exited immediately"}
	}

	info := &DevServerInfo{
		ServerID:       serverID,
		Process:        cmd,
		TempFile:       tmpFile.Name(),
		PID:            cmd.Process.Pid,
		LastActivity:   time.Now(),
		TenantID:       tenantID,
		ConversationID: conversationID,
		stdin:          bufio.NewWriter(stdinPipe),
		stdout:         bufio.NewReader(stdoutPipe),
	}

	// Discover tools via MCP JSON-RPC over stdio.
	tools := s.discoverTools(info)
	info.Tools = tools

	// Register server in the playground_mcp_servers table.
	if s.playgroundSvc != nil {
		s.registerInDB(info)
	}

	s.mu.Lock()
	s.runningServers[conversationID] = info
	s.mu.Unlock()

	logrus.Infof("Dev server started: %s (PID %d, %d tools)", serverID, info.PID, len(tools))

	return map[string]interface{}{
		"success":     true,
		"server_id":   serverID,
		"pid":         info.PID,
		"tools":       tools,
		"tools_count": len(tools),
	}
}

// StopServer stops a running dev server.
func (s *DevServerService) StopServer(serverID, conversationID, tenantID string) map[string]interface{} {
	s.mu.Lock()
	info, ok := s.runningServers[conversationID]
	if !ok {
		s.mu.Unlock()
		return map[string]interface{}{"success": false, "error": "no dev server running for this conversation"}
	}
	if info.ServerID != serverID {
		s.mu.Unlock()
		return map[string]interface{}{"success": false, "error": "server ID mismatch"}
	}
	delete(s.runningServers, conversationID)
	s.mu.Unlock()

	s.stopLocked(info)

	// Remove from playground DB.
	if s.playgroundSvc != nil {
		s.removeFromDB(info)
	}

	logrus.Infof("Dev server stopped: %s", serverID)
	return map[string]interface{}{"success": true, "message": "Server stopped successfully"}
}

// GetServerStatus returns the status of a dev server for a conversation.
func (s *DevServerService) GetServerStatus(conversationID, tenantID string) map[string]interface{} {
	s.mu.Lock()
	info, ok := s.runningServers[conversationID]
	s.mu.Unlock()

	if !ok {
		return map[string]interface{}{"running": false}
	}

	// Check if the process is still alive.
	if info.Process.ProcessState != nil && info.Process.ProcessState.Exited() {
		s.mu.Lock()
		delete(s.runningServers, conversationID)
		s.mu.Unlock()
		os.Remove(info.TempFile)
		return map[string]interface{}{"running": false}
	}

	return map[string]interface{}{
		"running":   true,
		"server_id": info.ServerID,
		"pid":       info.PID,
	}
}

// ── internal helpers ─────────────────────────────────────────────────────

func (s *DevServerService) stopLocked(info *DevServerInfo) {
	if info.Process.Process != nil {
		info.Process.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- info.Process.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			info.Process.Process.Kill()
			<-done
		}
	}
	os.Remove(info.TempFile)
}

// discoverTools sends MCP initialize + tools/list over stdio.
func (s *DevServerService) discoverTools(info *DevServerInfo) []map[string]interface{} {
	// Initialize
	initReq := map[string]interface{}{
		"jsonrpc": "2.0", "id": 0, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]interface{}{"name": "dev-server-client", "version": "1.0.0"},
		},
	}
	if _, err := s.sendRPC(info, initReq, 3*time.Second); err != nil {
		logrus.Errorf("Dev server init failed: %v", err)
		return nil
	}

	// List tools
	listReq := map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list",
	}
	resp, err := s.sendRPC(info, listReq, 3*time.Second)
	if err != nil {
		logrus.Errorf("Dev server tools/list failed: %v", err)
		return nil
	}

	result, _ := resp["result"].(map[string]interface{})
	toolsRaw, _ := result["tools"].([]interface{})
	var tools []map[string]interface{}
	for _, t := range toolsRaw {
		if tm, ok := t.(map[string]interface{}); ok {
			tools = append(tools, tm)
		}
	}
	return tools
}

func (s *DevServerService) sendRPC(info *DevServerInfo, req map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	data, _ := json.Marshal(req)
	data = append(data, '\n')

	if _, err := info.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	if err := info.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("flush: %w", err)
	}

	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		line, err := info.stdout.ReadString('\n')
		ch <- readResult{line, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(r.line), &resp); err != nil {
			return nil, fmt.Errorf("json: %w", err)
		}
		info.LastActivity = time.Now()
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout")
	}
}

func (s *DevServerService) registerInDB(info *DevServerInfo) {
	db, err := s.playgroundSvc.tenantDB(info.TenantID)
	if err != nil || db == nil {
		return
	}
	db.Exec(
		"DELETE FROM playground_mcp_servers WHERE conversation_id = ? AND server_url LIKE 'dev://%'",
		info.ConversationID,
	)
	db.Exec(
		"INSERT INTO playground_mcp_servers (conversation_id, name, protocol, server_url, config, is_connected) VALUES (?, ?, ?, ?, ?, true)",
		info.ConversationID,
		fmt.Sprintf("Dev Server (%s)", info.ServerID),
		"stdio",
		fmt.Sprintf("dev://%s", info.ServerID),
		"{}",
	)
}

func (s *DevServerService) removeFromDB(info *DevServerInfo) {
	db, err := s.playgroundSvc.tenantDB(info.TenantID)
	if err != nil || db == nil {
		return
	}
	db.Exec(
		"DELETE FROM playground_mcp_servers WHERE conversation_id = ? AND server_url = ?",
		info.ConversationID,
		fmt.Sprintf("dev://%s", info.ServerID),
	)
}

func (s *DevServerService) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupInactive()
	}
}

func (s *DevServerService) cleanupInactive() {
	s.mu.Lock()
	var stale []*DevServerInfo
	for cid, info := range s.runningServers {
		if time.Since(info.LastActivity) > s.inactiveTimeout {
			stale = append(stale, info)
			delete(s.runningServers, cid)
		}
	}
	s.mu.Unlock()

	for _, info := range stale {
		logrus.Infof("Cleaning up inactive dev server: %s", info.ServerID)
		s.stopLocked(info)
		if s.playgroundSvc != nil {
			s.removeFromDB(info)
		}
	}
}

// generateServerCode wraps user Python code with MCP server boilerplate.
func generateServerCode(userCode, tenantID string) string {
	return fmt.Sprintf(`#!/usr/bin/env python3
"""Auto-generated MCP Server from Dev Mode - tenant: %s"""
import sys, os, asyncio, logging, json

logging.basicConfig(level=logging.INFO, format='[%%(levelname)s] %%(message)s', stream=sys.stderr)
logger = logging.getLogger(__name__)

try:
    from mcp.server import Server
    from mcp.server.stdio import stdio_server
    from mcp.types import Tool
    logger.info("Imported MCP SDK")
except ImportError as e:
    logger.error(f"MCP SDK import failed: {e}")
    sys.exit(1)

# ============= USER CODE =============
%s
# ============= END USER CODE =============

def run_mcp_server_stdio():
    server = Server("dev-mcp-server")
    tools_dict = {}
    for name, obj in list(sys.modules[__name__].__dict__.items()):
        if callable(obj) and hasattr(obj, '_mcp_tool_name'):
            tools_dict[name] = obj
            logger.info(f"Registered tool: {name}")

    @server.list_tools()
    async def list_tools():
        tool_list = []
        for name, func in tools_dict.items():
            tool_list.append(Tool(
                name=getattr(func, '_mcp_tool_name', name),
                description=getattr(func, '_mcp_tool_description', func.__doc__ or ''),
                inputSchema=getattr(func, '_mcp_tool_inputSchema', {}),
            ))
        return tool_list

    @server.call_tool()
    async def call_tool(name: str, arguments: dict):
        for fn, func in tools_dict.items():
            if getattr(func, '_mcp_tool_name', fn) == name:
                try:
                    result = await func(arguments)
                    return result if isinstance(result, list) else [{"type": "text", "text": str(result)}]
                except Exception as e:
                    return [{"type": "text", "text": json.dumps({"error": str(e)})}]
        return [{"type": "text", "text": json.dumps({"error": f"Tool '{name}' not found"})}]

    async def main():
        async with stdio_server() as (read_stream, write_stream):
            await server.run(read_stream, write_stream, server.create_initialization_options())
    asyncio.run(main())

if __name__ == "__main__":
    try:
        run_mcp_server_stdio()
    except Exception as e:
        logger.error(f"Server error: {e}")
        sys.exit(1)
`, tenantID, userCode)
}
