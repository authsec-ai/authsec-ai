import type { SDKHelpItem } from "../types";

export const VOICE_AGENT_SDK_HELP: SDKHelpItem[] = [
  {
    id: "voice-agent-quickstart",
    question: "How do I set up voice authentication for my agent?",
    description:
      "Initialize the CIBA SDK to enable passwordless authentication for voice assistants using push notifications.",
    code: {
      python: [
        {
          label: "Step 1: Install SDK",
          code: `pip install git+https://github.com/authsec-ai/sdk-authsec.git`,
        },
        {
          label: "Step 2: Initialize CIBAClient",
          code: `from AuthSec_SDK import CIBAClient

# Initialize with your client ID (from the Clients page)
client = CIBAClient(client_id="your-client-id")

# For platform-level access (admin flow), omit client_id:
# admin_client = CIBAClient()`,
        },
        {
          label: "Step 3: Authenticate User",
          code: `# Send push notification to user's AuthSec mobile app
result = client.initiate_app_approval("user@example.com")

# Wait for user approval (blocks until approved, denied, or timeout)
approval = client.poll_for_approval(
    email="user@example.com",
    auth_req_id=result["auth_req_id"],
    timeout=60  # Wait up to 60 seconds
)

if approval["status"] == "approved":
    token = approval["token"]
    print(f"✅ User authenticated! Token: {token[:50]}...")
elif approval["status"] == "access_denied":
    print("❌ User denied the request")
elif approval["status"] == "timeout":
    print("⏱️ Request timed out")`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "ciba-push-flow",
    question: "How does CIBA push notification authentication work?",
    description:
      "CIBA (Client-Initiated Backchannel Authentication) sends a push notification to the user's mobile app for approval - perfect for voice assistants and hands-free devices.",
    code: {
      python: [
        {
          label: "CIBA Flow Diagram",
          code: `# CIBA Authentication Flow
#
# ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
# │  Your App   │         │   AuthSec   │         │ User's Phone│
# └──────┬──────┘         └──────┬──────┘         └──────┬──────┘
#        │                       │                       │
#        │ 1. initiate_app_approval()                    │
#        ├──────────────────────►│                       │
#        │                       │                       │
#        │                       │ 2. Push notification  │
#        │                       ├──────────────────────►│
#        │                       │                       │
#        │ 3. poll_for_approval()│                       │
#        │    (blocking)         │                       │
#        ├──────────────────────►│                       │
#        │                       │   3a. User approves   │
#        │                       │◄──────────────────────┤
#        │                       │                       │
#        │ 4. Return token       │                       │
#        │◄──────────────────────┤                       │`,
        },
        {
          label: "Implementation",
          code: `from AuthSec_SDK import CIBAClient

client = CIBAClient(client_id="your-client-id")

def authenticate_via_ciba(email: str) -> dict:
    """Authenticate user via CIBA push notification"""
    
    # Step 1: Initiate - sends push to user's phone
    result = client.initiate_app_approval(email)
    print(f"📱 Push notification sent! Request ID: {result['auth_req_id']}")
    
    # Step 2: Poll - wait for user response
    approval = client.poll_for_approval(
        email=email,
        auth_req_id=result["auth_req_id"],
        interval=5,   # Check every 5 seconds
        timeout=300   # Wait up to 5 minutes
    )
    
    return approval`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "totp-fallback",
    question: "How do I implement TOTP as a fallback?",
    description:
      "TOTP (Time-based One-Time Password) provides 6-digit code verification as a backup when push notifications aren't available.",
    code: {
      python: [
        {
          label: "TOTP Flow",
          code: `# TOTP Authentication Flow
#
# ┌─────────────┐         ┌─────────────┐
# │  Your App   │         │   AuthSec   │
# └──────┬──────┘         └──────┬──────┘
#        │                       │
#        │ 1. User enters code   │
#        │    (from auth app)    │
#        │                       │
#        │ 2. verify_totp()      │
#        ├──────────────────────►│
#        │                       │
#        │ 3. Return token       │
#        │◄──────────────────────┤`,
        },
        {
          label: "TOTP Verification",
          code: `from AuthSec_SDK import CIBAClient

client = CIBAClient(client_id="your-client-id")

def verify_totp_code(email: str, code: str) -> dict:
    """Verify a 6-digit TOTP code"""
    
    result = client.verify_totp(email, code)
    
    if result["success"]:
        print(f"✅ Valid code! Token: {result['token'][:50]}...")
        return {"success": True, "token": result["token"]}
    else:
        print(f"❌ Invalid code. {result['remaining']} attempts left")
        return {"success": False, "remaining": result["remaining"]}

# Example usage
result = verify_totp_code("user@example.com", "123456")`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "voice-assistant-example",
    question: "How do I integrate authentication into a voice assistant?",
    description:
      "Complete example of a voice assistant class that handles both CIBA and TOTP authentication flows.",
    code: {
      python: [
        {
          label: "Complete Voice Assistant Class",
          code: `from AuthSec_SDK import CIBAClient

class VoiceAssistant:
    def __init__(self, client_id: str):
        self.ciba = CIBAClient(client_id=client_id)
    
    def authenticate_user(self, email: str) -> str | None:
        """Handle voice authentication with both methods"""
        
        # Ask user which method they prefer
        method = self.ask_user("Would you like to approve via your app or use a code?")
        
        if "app" in method.lower():
            return self._authenticate_ciba(email)
        else:
            return self._authenticate_totp(email)
    
    def _authenticate_ciba(self, email: str) -> str | None:
        """CIBA flow - push notification"""
        self.speak("I've sent a notification to your AuthSec app. Please approve to continue.")
        
        result = self.ciba.initiate_app_approval(email)
        approval = self.ciba.poll_for_approval(
            email, 
            result["auth_req_id"], 
            timeout=60
        )
        
        if approval["status"] == "approved":
            self.speak("Great! You're now authenticated.")
            return approval["token"]
        elif approval["status"] == "access_denied":
            self.speak("Authentication was denied. Please try again.")
        else:
            self.speak(f"Authentication {approval['status']}. Please try again.")
        return None
    
    def _authenticate_totp(self, email: str) -> str | None:
        """TOTP flow - 6-digit code"""
        self.speak("Please tell me your 6-digit authentication code.")
        code = self.listen_for_digits()
        
        result = self.ciba.verify_totp(email, code)
        
        if result["success"]:
            self.speak("Perfect! You're now authenticated.")
            return result["token"]
        else:
            self.speak(f"Invalid code. You have {result['remaining']} attempts remaining.")
        return None
    
    # Placeholder methods - implement based on your voice platform
    def speak(self, text: str): pass
    def ask_user(self, question: str) -> str: pass  
    def listen_for_digits(self) -> str: pass`,
        },
        {
          label: "Usage Example",
          code: `# Initialize and use the voice assistant
assistant = VoiceAssistant(client_id="your-client-id")

# When user requests an action requiring auth
user_email = "user@example.com"
token = assistant.authenticate_user(user_email)

if token:
    # User is authenticated - proceed with protected action
    perform_protected_action(token)
else:
    assistant.speak("I couldn't authenticate you. Please try again.")`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "error-handling",
    question: "How do I handle authentication errors?",
    description:
      "Best practices for handling errors in voice authentication flows including network issues and user actions.",
    code: {
      python: [
        {
          label: "Error Handling Pattern",
          code: `from AuthSec_SDK import CIBAClient
import requests

client = CIBAClient(client_id="your-client-id")

def safe_authenticate(email: str) -> dict:
    """Robust authentication with error handling"""
    try:
        # Initiate CIBA
        result = client.initiate_app_approval(email)
        
        if "error" in result:
            return {"success": False, "error": result["error"]}
        
        # Poll for approval
        approval = client.poll_for_approval(
            email, 
            result["auth_req_id"], 
            timeout=60
        )
        
        if approval["status"] == "approved":
            return {"success": True, "token": approval["token"]}
        else:
            return {"success": False, "error": approval["status"]}
    
    except requests.exceptions.Timeout:
        return {"success": False, "error": "network_timeout"}
    except requests.exceptions.ConnectionError:
        return {"success": False, "error": "connection_failed"}
    except Exception as e:
        return {"success": False, "error": str(e)}`,
        },
        {
          label: "Common Errors Reference",
          code: `# Common Authentication Errors
# 
# | Error               | Cause                              | Solution                        |
# |---------------------|------------------------------------|---------------------------------|
# | too_many_retries    | 3 failed TOTP attempts             | Call cancel_approval() to reset |
# | invalid_code        | Wrong TOTP code                    | Ask user for correct code       |
# | access_denied       | User rejected push notification    | Retry or use TOTP               |
# | expired_token       | CIBA request timed out on server   | Call initiate_app_approval()    |
# | timeout             | Local poll timeout reached         | Increase timeout or retry       |

# Reset after too many failed attempts
client.cancel_approval("user@example.com")`,
        },
      ],
      typescript: [],
    },
  },
  {
    id: "async-threading",
    question: "How do I run authentication without blocking?",
    description:
      "Use threading to run CIBA authentication in the background without blocking your main application.",
    code: {
      python: [
        {
          label: "Non-Blocking Authentication",
          code: `from AuthSec_SDK import CIBAClient
import threading
from typing import Callable

class AsyncAuthHandler:
    def __init__(self, client_id: str):
        self.ciba = CIBAClient(client_id=client_id)
        self.pending_auths = {}
    
    def start_authentication(
        self, 
        email: str, 
        callback: Callable[[str, dict], None]
    ) -> str:
        """Start CIBA authentication in background thread"""
        
        # Initiate the request
        result = self.ciba.initiate_app_approval(email)
        auth_req_id = result["auth_req_id"]
        
        # Poll in background thread
        def poll_thread():
            approval = self.ciba.poll_for_approval(
                email, 
                auth_req_id, 
                timeout=120
            )
            callback(email, approval)
        
        thread = threading.Thread(target=poll_thread, daemon=True)
        thread.start()
        
        self.pending_auths[email] = thread
        return auth_req_id
    
    def cancel_authentication(self, email: str):
        """Cancel a pending authentication"""
        self.ciba.cancel_approval(email)`,
        },
        {
          label: "Usage with Callback",
          code: `# Define callback for when auth completes
def on_auth_complete(email: str, result: dict):
    if result["status"] == "approved":
        print(f"✅ User {email} authenticated!")
        # Handle successful auth...
    else:
        print(f"❌ Auth failed for {email}: {result['status']}")
        # Handle failure...

# Create handler and start auth
handler = AsyncAuthHandler(client_id="your-client-id")
handler.start_authentication("user@example.com", on_auth_complete)

# Your app continues running while auth happens in background
print("Authentication started - continuing with other tasks...")`,
        },
      ],
      typescript: [],
    },
  },
];
