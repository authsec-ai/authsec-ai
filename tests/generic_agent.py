"""
AuthSec Generic AI Agent — Permission-Gated Tools via Delegated Trust.

This is a GENERAL-PURPOSE agent where AuthSec delegation controls
which tools are available. The admin decides what the agent can do
by setting permissions in the delegation policy.

Permission → Tool Mapping:
  files:read     → read_file, list_directory
  files:write    → write_file
  shell:execute  → run_shell_command
  web:search     → web_search
  web:fetch      → fetch_url
  math:compute   → calculator
  data:query     → query_json
  users:read     → list_users (AuthSec API)
  clients:read   → list_clients (AuthSec API)

If the admin only delegates ["files:read", "math:compute"],
the agent can ONLY read files and do math — no shell, no web, no writes.

Usage:
  export CLIENT_ID="<your-agent-client-id>"
  export USERFLOW_URL="https://prod.api.authsec.ai/uflow"
  export OPENAI_API_KEY="sk-..."

  python tests/generic_agent.py
"""

import asyncio
import argparse
import json
import math
import os
import subprocess
import sys
from pathlib import Path
from typing import Any

# Add SDK to path
sys.path.insert(0, os.path.join(
    os.path.dirname(__file__), "..", "..", "sdk-authsec", "packages", "python-sdk", "src"
))

import aiohttp
from openai import OpenAI

from authsec_sdk.delegation_sdk import (
    DelegationClient,
    DelegationError,
    DelegationTokenNotFound,
)

# ─────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────

CLIENT_ID = os.getenv("CLIENT_ID", "9b477e30-3989-4c12-926c-3945a59f761f")
USERFLOW_URL = os.getenv("USERFLOW_URL", "https://prod.api.authsec.ai/uflow")
BASE_API_URL = os.getenv("BASE_API_URL", "https://prod.api.authsec.ai")
WORKSPACE_DIR = os.getenv("WORKSPACE_DIR", os.getcwd())

# ─────────────────────────────────────────────────────
# Tool Registry — permission → tool definitions
# Each entry maps a required permission to an OpenAI
# tool definition + its executor function name
# ─────────────────────────────────────────────────────

TOOL_REGISTRY = {
    # ── File Operations ──
    "files:read": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "read_file",
                    "description": "Read the contents of a file. Requires 'files:read' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "path": {
                                "type": "string",
                                "description": "File path (relative to workspace or absolute)",
                            },
                            "max_lines": {
                                "type": "integer",
                                "description": "Max lines to read (default: 200)",
                                "default": 200,
                            },
                        },
                        "required": ["path"],
                    },
                },
            },
        },
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "list_directory",
                    "description": "List files and folders in a directory. Requires 'files:read' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "path": {
                                "type": "string",
                                "description": "Directory path (default: workspace root)",
                                "default": ".",
                            },
                            "pattern": {
                                "type": "string",
                                "description": "Glob pattern to filter (e.g. '*.py', '**/*.json')",
                                "default": "*",
                            },
                        },
                    },
                },
            },
        },
    ],
    "files:write": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "write_file",
                    "description": "Write content to a file. Creates parent dirs if needed. Requires 'files:write' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "path": {
                                "type": "string",
                                "description": "File path to write to",
                            },
                            "content": {
                                "type": "string",
                                "description": "Content to write",
                            },
                            "append": {
                                "type": "boolean",
                                "description": "Append instead of overwrite (default: false)",
                                "default": False,
                            },
                        },
                        "required": ["path", "content"],
                    },
                },
            },
        },
    ],
    # ── Shell Execution ──
    "shell:execute": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "run_shell_command",
                    "description": "Execute a shell command and return stdout/stderr. Requires 'shell:execute' permission. Timeout: 30s.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "command": {
                                "type": "string",
                                "description": "Shell command to run",
                            },
                            "working_dir": {
                                "type": "string",
                                "description": "Working directory (default: workspace)",
                            },
                        },
                        "required": ["command"],
                    },
                },
            },
        },
    ],
    # ── Web Operations ──
    "web:search": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "web_search",
                    "description": "Search the web using DuckDuckGo. Requires 'web:search' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "query": {
                                "type": "string",
                                "description": "Search query",
                            },
                            "max_results": {
                                "type": "integer",
                                "description": "Max results to return (default: 5)",
                                "default": 5,
                            },
                        },
                        "required": ["query"],
                    },
                },
            },
        },
    ],
    "web:fetch": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "fetch_url",
                    "description": "Fetch content from a URL. Requires 'web:fetch' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "url": {
                                "type": "string",
                                "description": "URL to fetch",
                            },
                            "method": {
                                "type": "string",
                                "description": "HTTP method (default: GET)",
                                "default": "GET",
                                "enum": ["GET", "POST", "PUT", "DELETE"],
                            },
                            "headers": {
                                "type": "object",
                                "description": "Optional HTTP headers",
                            },
                            "body": {
                                "type": "string",
                                "description": "Optional request body (for POST/PUT)",
                            },
                        },
                        "required": ["url"],
                    },
                },
            },
        },
    ],
    # ── Math / Compute ──
    "math:compute": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "calculator",
                    "description": "Evaluate a mathematical expression safely. Requires 'math:compute' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "expression": {
                                "type": "string",
                                "description": "Math expression (e.g. '2**10', 'math.sqrt(144)', '(45*3)+17')",
                            },
                        },
                        "required": ["expression"],
                    },
                },
            },
        },
    ],
    # ── Data / JSON Query ──
    "data:query": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "query_json",
                    "description": "Query a JSON file using a JMESPath-like dot notation. Requires 'data:query' permission.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "file_path": {
                                "type": "string",
                                "description": "Path to JSON file",
                            },
                            "query": {
                                "type": "string",
                                "description": "Dot-separated key path (e.g. 'users.0.name', 'config.database.host')",
                            },
                        },
                        "required": ["file_path"],
                    },
                },
            },
        },
    ],
    # ── AuthSec API Tools ──
    "users:read": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "list_users",
                    "description": "List end-users in the AuthSec tenant. Requires 'users:read' permission.",
                    "parameters": {"type": "object", "properties": {}},
                },
            },
        },
    ],
    "clients:read": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "list_clients",
                    "description": "List registered clients/apps in the AuthSec tenant. Requires 'clients:read' permission.",
                    "parameters": {"type": "object", "properties": {}},
                },
            },
        },
    ],
    "secrets:read": [
        {
            "tool": {
                "type": "function",
                "function": {
                    "name": "list_secrets",
                    "description": "List secrets in the AuthSec tenant. Requires 'secrets:read' permission.",
                    "parameters": {"type": "object", "properties": {}},
                },
            },
        },
    ],
}

# ── Always-available tools (no permission needed) ──
ALWAYS_AVAILABLE_TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "get_my_permissions",
            "description": "Show which permissions this agent has and which tools are available.",
            "parameters": {"type": "object", "properties": {}},
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_my_identity",
            "description": "Show the agent's SPIFFE ID, client_id, tenant, and token expiry.",
            "parameters": {"type": "object", "properties": {}},
        },
    },
]


def _resolve_path(path_str: str) -> Path:
    """Resolve a path relative to workspace, prevent directory traversal."""
    p = Path(path_str)
    if p.is_absolute():
        return p
    resolved = (Path(WORKSPACE_DIR) / p).resolve()
    # Prevent escaping workspace
    workspace = Path(WORKSPACE_DIR).resolve()
    if not str(resolved).startswith(str(workspace)):
        raise PermissionError(f"Path escapes workspace: {path_str}")
    return resolved


class GenericAgent:
    """
    General-purpose AI agent where AuthSec delegation controls
    which tools are available. Permission → Tool gating.
    """

    def __init__(self, client_id: str, userflow_url: str, base_api_url: str):
        self.delegation = DelegationClient(
            client_id=client_id,
            userflow_url=userflow_url,
        )
        self.base_api_url = base_api_url.rstrip("/")
        self.openai = OpenAI()
        self.messages = []
        self.active_tools = []  # Tools available based on permissions
        self.tool_name_to_permission = {}  # Reverse lookup: tool_name → permission

    async def initialize(self):
        """Pull delegation token and build the tool set from permissions."""
        print("\n[Agent] Pulling delegation token...")
        try:
            await self.delegation.pull_token()
            print(f"[Agent] Token obtained!")
            print(f"  SPIFFE ID:   {self.delegation.spiffe_id}")
            print(f"  Permissions: {self.delegation.permissions}")
            print(f"  Expires in:  {self.delegation.expires_in_seconds}s")
        except DelegationTokenNotFound:
            print("[Agent] ERROR: No delegation token found.")
            print("  An admin must first delegate a token to this agent.")
            sys.exit(1)
        except DelegationError as e:
            print(f"[Agent] ERROR: {e}")
            sys.exit(1)

        # ── Build active tool set based on permissions ──
        self.active_tools = list(ALWAYS_AVAILABLE_TOOLS)
        enabled_perms = []
        blocked_perms = []

        for permission, tool_entries in TOOL_REGISTRY.items():
            if self.delegation.has_permission(permission):
                enabled_perms.append(permission)
                for entry in tool_entries:
                    self.active_tools.append(entry["tool"])
                    tool_name = entry["tool"]["function"]["name"]
                    self.tool_name_to_permission[tool_name] = permission
            else:
                blocked_perms.append(permission)

        print(f"\n[Agent] Tool gating results:")
        print(f"  Enabled permissions:  {enabled_perms}")
        print(f"  Blocked permissions:  {blocked_perms}")
        print(f"  Available tools:      {[t['function']['name'] for t in self.active_tools]}")

        # ── System prompt ──
        tool_list = "\n".join(
            f"  - {t['function']['name']}: {t['function']['description']}"
            for t in self.active_tools
        )
        self.messages = [
            {
                "role": "system",
                "content": f"""You are a general-purpose AI agent with identity-based access control powered by AuthSec.

Your identity:
- SPIFFE ID: {self.delegation.spiffe_id}
- Client ID: {self.delegation.client_id}
- Delegated permissions: {json.dumps(self.delegation.permissions)}
- Token expires in: {self.delegation.expires_in_seconds} seconds

Available tools (gated by your permissions):
{tool_list}

Rules:
- You can ONLY use the tools listed above. These are the only tools your admin has authorized.
- If a user asks you to do something you don't have a tool for, explain which permission is needed.
- Be helpful, concise, and format output nicely.
- For file operations, paths are relative to the workspace: {WORKSPACE_DIR}""",
            }
        ]

    # ─────────────────────────────────────────────────────
    # Tool Executors
    # ─────────────────────────────────────────────────────

    async def execute_tool(self, name: str, args: dict) -> str:
        """Route tool call to the correct executor."""
        executors = {
            # Always available
            "get_my_permissions": self._exec_get_permissions,
            "get_my_identity": self._exec_get_identity,
            # files:read
            "read_file": self._exec_read_file,
            "list_directory": self._exec_list_directory,
            # files:write
            "write_file": self._exec_write_file,
            # shell:execute
            "run_shell_command": self._exec_shell,
            # web:search
            "web_search": self._exec_web_search,
            # web:fetch
            "fetch_url": self._exec_fetch_url,
            # math:compute
            "calculator": self._exec_calculator,
            # data:query
            "query_json": self._exec_query_json,
            # AuthSec API
            "list_users": self._exec_list_users,
            "list_clients": self._exec_list_clients,
            "list_secrets": self._exec_list_secrets,
        }
        executor = executors.get(name)
        if not executor:
            return json.dumps({"error": f"Unknown tool: {name}"})
        try:
            return await executor(args)
        except Exception as e:
            return json.dumps({"error": str(e)})

    # ── Always Available ──

    async def _exec_get_permissions(self, args: dict) -> str:
        perm_tool_map = {}
        for perm, entries in TOOL_REGISTRY.items():
            tools = [e["tool"]["function"]["name"] for e in entries]
            has = self.delegation.has_permission(perm)
            perm_tool_map[perm] = {"enabled": has, "tools": tools}
        return json.dumps({
            "delegated_permissions": self.delegation.permissions,
            "tool_gating": perm_tool_map,
            "active_tool_count": len(self.active_tools),
        }, indent=2)

    async def _exec_get_identity(self, args: dict) -> str:
        claims = self.delegation.decode_token_claims()
        return json.dumps({
            "spiffe_id": self.delegation.spiffe_id,
            "client_id": self.delegation.client_id,
            "tenant_id": claims.get("tenant_id"),
            "email": claims.get("email"),
            "agent_type": claims.get("agent_type"),
            "expires_in_seconds": self.delegation.expires_in_seconds,
        }, indent=2)

    # ── files:read ──

    async def _exec_read_file(self, args: dict) -> str:
        path = _resolve_path(args["path"])
        max_lines = args.get("max_lines", 200)
        if not path.exists():
            return json.dumps({"error": f"File not found: {path}"})
        if not path.is_file():
            return json.dumps({"error": f"Not a file: {path}"})
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
        truncated = len(lines) > max_lines
        content = "\n".join(lines[:max_lines])
        return json.dumps({
            "path": str(path),
            "total_lines": len(lines),
            "truncated": truncated,
            "content": content,
        }, indent=2)[:4000]

    async def _exec_list_directory(self, args: dict) -> str:
        path = _resolve_path(args.get("path", "."))
        pattern = args.get("pattern", "*")
        if not path.exists():
            return json.dumps({"error": f"Directory not found: {path}"})
        entries = sorted(path.glob(pattern))[:100]
        items = []
        for e in entries:
            items.append({
                "name": e.name,
                "type": "dir" if e.is_dir() else "file",
                "size": e.stat().st_size if e.is_file() else None,
            })
        return json.dumps({"path": str(path), "count": len(items), "entries": items}, indent=2)[:4000]

    # ── files:write ──

    async def _exec_write_file(self, args: dict) -> str:
        path = _resolve_path(args["path"])
        content = args["content"]
        append = args.get("append", False)
        path.parent.mkdir(parents=True, exist_ok=True)
        mode = "a" if append else "w"
        path.write_text(content, encoding="utf-8") if not append else open(path, mode).write(content)
        return json.dumps({"success": True, "path": str(path), "bytes_written": len(content)})

    # ── shell:execute ──

    async def _exec_shell(self, args: dict) -> str:
        command = args["command"]
        cwd = args.get("working_dir", WORKSPACE_DIR)
        try:
            result = subprocess.run(
                command, shell=True, capture_output=True, text=True,
                timeout=30, cwd=cwd
            )
            return json.dumps({
                "exit_code": result.returncode,
                "stdout": result.stdout[:3000],
                "stderr": result.stderr[:1000],
            }, indent=2)
        except subprocess.TimeoutExpired:
            return json.dumps({"error": "Command timed out after 30 seconds"})

    # ── web:search ──

    async def _exec_web_search(self, args: dict) -> str:
        query = args["query"]
        max_results = args.get("max_results", 5)
        # Use DuckDuckGo HTML (no API key needed)
        url = f"https://html.duckduckgo.com/html/?q={query}"
        timeout = aiohttp.ClientTimeout(total=10)
        async with aiohttp.ClientSession(timeout=timeout) as session:
            async with session.get(url, headers={"User-Agent": "AuthSec-Agent/1.0"}) as resp:
                html = await resp.text()
        # Simple extraction of result snippets
        results = []
        import re
        links = re.findall(r'class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>', html)
        snippets = re.findall(r'class="result__snippet">(.*?)</(?:td|span|a)', html, re.DOTALL)
        for i, (link, title) in enumerate(links[:max_results]):
            snippet = snippets[i].strip() if i < len(snippets) else ""
            snippet = re.sub(r"<[^>]+>", "", snippet).strip()
            title = re.sub(r"<[^>]+>", "", title).strip()
            results.append({"title": title, "url": link, "snippet": snippet[:200]})
        return json.dumps({"query": query, "results": results}, indent=2)[:4000]

    # ── web:fetch ──

    async def _exec_fetch_url(self, args: dict) -> str:
        url = args["url"]
        method = args.get("method", "GET")
        headers = args.get("headers", {})
        body = args.get("body")
        timeout = aiohttp.ClientTimeout(total=15)
        async with aiohttp.ClientSession(timeout=timeout) as session:
            kwargs = {"headers": headers}
            if body and method in ("POST", "PUT"):
                kwargs["data"] = body
            async with session.request(method, url, **kwargs) as resp:
                content_type = resp.headers.get("Content-Type", "")
                if "json" in content_type:
                    data = await resp.json()
                    body_text = json.dumps(data, indent=2)
                else:
                    body_text = await resp.text()
                return json.dumps({
                    "status": resp.status,
                    "content_type": content_type,
                    "body": body_text[:3000],
                }, indent=2)

    # ── math:compute ──

    async def _exec_calculator(self, args: dict) -> str:
        expr = args["expression"]
        # Safe evaluation: only allow math operations
        allowed = {
            "abs": abs, "round": round, "min": min, "max": max,
            "sum": sum, "len": len, "int": int, "float": float,
            "pow": pow, "divmod": divmod,
            **{k: getattr(math, k) for k in dir(math) if not k.startswith("_")},
        }
        try:
            result = eval(expr, {"__builtins__": {}}, allowed)
            return json.dumps({"expression": expr, "result": result})
        except Exception as e:
            return json.dumps({"expression": expr, "error": str(e)})

    # ── data:query ──

    async def _exec_query_json(self, args: dict) -> str:
        path = _resolve_path(args["file_path"])
        query = args.get("query", "")
        if not path.exists():
            return json.dumps({"error": f"File not found: {path}"})
        data = json.loads(path.read_text(encoding="utf-8"))
        if query:
            for key in query.split("."):
                if isinstance(data, dict):
                    data = data.get(key)
                elif isinstance(data, list):
                    try:
                        data = data[int(key)]
                    except (ValueError, IndexError):
                        data = None
                else:
                    data = None
                if data is None:
                    break
        return json.dumps({"path": str(path), "query": query, "result": data}, indent=2, default=str)[:4000]

    # ── AuthSec API Tools ──

    async def _api_call(self, method: str, path: str) -> dict:
        token = await self.delegation.ensure_token()
        url = f"{self.base_api_url}{path}"
        headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
        timeout = aiohttp.ClientTimeout(total=10)
        async with aiohttp.ClientSession(timeout=timeout) as session:
            async with session.request(method, url, headers=headers) as resp:
                try:
                    body = await resp.json()
                except Exception:
                    body = {"raw": await resp.text(), "status": resp.status}
                return {"status": resp.status, "data": body}

    async def _exec_list_users(self, args: dict) -> str:
        result = await self._api_call("GET", "/uflow/user/enduser/list")
        return json.dumps(result, indent=2, default=str)[:3000]

    async def _exec_list_clients(self, args: dict) -> str:
        result = await self._api_call("GET", "/uflow/user/clients")
        return json.dumps(result, indent=2, default=str)[:3000]

    async def _exec_list_secrets(self, args: dict) -> str:
        result = await self._api_call("GET", "/uflow/user/secrets")
        return json.dumps(result, indent=2, default=str)[:3000]

    # ─────────────────────────────────────────────────────
    # Chat Loop
    # ─────────────────────────────────────────────────────

    async def chat(self, user_input: str) -> str:
        """Process user message, call tools if needed, return response."""
        self.messages.append({"role": "user", "content": user_input})

        kwargs = {"model": "gpt-4o-mini", "messages": self.messages}
        if self.active_tools:
            kwargs["tools"] = self.active_tools
            kwargs["tool_choice"] = "auto"

        response = self.openai.chat.completions.create(**kwargs)
        msg = response.choices[0].message

        while msg.tool_calls:
            self.messages.append(msg)
            for tc in msg.tool_calls:
                fn_name = tc.function.name
                fn_args = json.loads(tc.function.arguments or "{}")
                perm = self.tool_name_to_permission.get(fn_name, "always")
                print(f"  [Tool] {fn_name}({json.dumps(fn_args)[:100]})  [perm: {perm}]")
                result = await self.execute_tool(fn_name, fn_args)
                print(f"  [Result] {result[:150]}{'...' if len(result) > 150 else ''}")
                self.messages.append({
                    "role": "tool",
                    "tool_call_id": tc.id,
                    "content": result,
                })

            response = self.openai.chat.completions.create(**kwargs)
            msg = response.choices[0].message

        self.messages.append({"role": "assistant", "content": msg.content})
        return msg.content


async def main():
    parser = argparse.ArgumentParser(
        description="AuthSec Generic Agent — Permission-gated tools via Delegated Trust"
    )
    parser.add_argument("--client-id", default=None)
    parser.add_argument("--userflow-url", default=None)
    parser.add_argument("--base-api-url", default=None)
    parser.add_argument("--workspace", default=None, help="Workspace directory for file ops")
    args = parser.parse_args()

    client_id = args.client_id or CLIENT_ID
    userflow_url = args.userflow_url or USERFLOW_URL
    base_api_url = args.base_api_url or BASE_API_URL
    global WORKSPACE_DIR
    WORKSPACE_DIR = args.workspace or WORKSPACE_DIR

    if not client_id:
        print("ERROR: --client-id required (or set CLIENT_ID env var)")
        sys.exit(1)

    print(f"[Config] client_id:    {client_id}")
    print(f"[Config] userflow_url: {userflow_url}")
    print(f"[Config] base_api_url: {base_api_url}")
    print(f"[Config] workspace:    {WORKSPACE_DIR}")

    agent = GenericAgent(
        client_id=client_id,
        userflow_url=userflow_url,
        base_api_url=base_api_url,
    )
    await agent.initialize()

    perms = agent.delegation.permissions
    tools = [t["function"]["name"] for t in agent.active_tools]

    print("\n" + "=" * 60)
    print("  AuthSec Generic Agent (Permission-Gated Tools)")
    print("=" * 60)
    print(f"  Permissions: {perms}")
    print(f"  Tools:       {tools}")
    print(f"  Expires in:  {agent.delegation.expires_in_seconds}s")
    print(f"  Type 'quit' to exit")
    print("=" * 60 + "\n")

    while True:
        try:
            user_input = input("You: ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\n[Agent] Goodbye!")
            break

        if not user_input:
            continue
        if user_input.lower() in ("quit", "exit", "q"):
            print("[Agent] Goodbye!")
            break

        try:
            response = await agent.chat(user_input)
            print(f"\nAgent: {response}\n")
        except Exception as e:
            print(f"\n[Error] {e}\n")


if __name__ == "__main__":
    asyncio.run(main())
