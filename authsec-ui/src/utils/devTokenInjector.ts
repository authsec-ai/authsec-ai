/**
 * Dev Token Injector - Simple utility for localhost development
 * Allows injecting production tokens to test against production APIs locally
 */

interface JWTPayload {
  sub: string;
  email?: string;
  email_id?: string;
  tenant_id?: string;
  tenant_domain?: string;
  project_id?: string;
  client_id?: string;
  roles?: string[];
  resources?: string[];
  scopes?: string[];
  scope?: string;
  groups?: string[];
  token_type?: string;
  exp?: number;
  [key: string]: any;
}

interface SessionData {
  token: string;
  user: {
    id: string;
    email: string;
    name?: string;
  };
  projects?: any[];
  currentProject?: any;
  tenant_id?: string;
  tenant_domain?: string;
  project_id?: string;
  client_id?: string;
  user_id: string;
  jwtPayload: JWTPayload;
  roles: string[];
  resources: string[];
  scopes: string[];
  groups: string[];
  expiresAt: number;
  token_type?: string;
}

function decodeJWT(token: string): JWTPayload | null {
  try {
    const parts = token.split('.');
    console.log('Token parts:', parts.length);

    if (parts.length !== 3) {
      console.error('Invalid JWT format: expected 3 parts, got', parts.length);
      return null;
    }

    const base64Url = parts[1];
    if (!base64Url) {
      console.error('Missing payload section');
      return null;
    }

    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );

    const payload = JSON.parse(jsonPayload);
    console.log('Decoded payload:', payload);
    return payload;
  } catch (error) {
    console.error('Failed to decode JWT:', error);
    return null;
  }
}

function createSessionFromToken(token: string): SessionData | null {
  const payload = decodeJWT(token);

  if (!payload) {
    return null;
  }

  const userId = payload.sub || payload.user_id || payload.client_id || 'unknown';
  const email = payload.email || payload.email_id || 'dev@localhost';
  const scopes = Array.isArray(payload.scopes)
    ? payload.scopes
    : typeof payload.scope === "string"
      ? payload.scope.split(/\s+/).filter(Boolean)
      : [];

  // Calculate expiry (use JWT exp or default to 30 days)
  const expiresAt = payload.exp
    ? payload.exp * 1000
    : Date.now() + 30 * 24 * 60 * 60 * 1000;

  const sessionData: SessionData = {
    token,
    user: {
      id: userId,
      email: email,
      name: payload.name || email,
    },
    tenant_id: payload.tenant_id || '',
    tenant_domain: payload.tenant_domain,
    project_id: payload.project_id || '',
    client_id: payload.client_id || userId,
    user_id: userId,
    jwtPayload: { ...payload, scopes },
    roles: Array.isArray(payload.roles) ? payload.roles : [],
    resources: Array.isArray(payload.resources) ? payload.resources : [],
    scopes,
    groups: Array.isArray(payload.groups) ? payload.groups : [],
    expiresAt,
    token_type: payload.token_type,
  };

  return sessionData;
}

function saveSession(sessionData: SessionData): void {
  try {
    localStorage.setItem('authsec_session_v2', JSON.stringify(sessionData));
    console.log('✅ Session saved successfully!');
  } catch (error) {
    console.error('❌ Failed to save session:', error);
    throw error;
  }
}

export function showTokenInjectorPopup(): void {
  const popup = document.createElement('div');
  popup.style.cssText = `
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 999999;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  `;

  popup.innerHTML = `
    <div style="
      background: #1a1a1a;
      padding: 32px;
      border-radius: 12px;
      max-width: 500px;
      width: 90%;
      box-shadow: 0 20px 60px rgba(0,0,0,0.5);
      border: 1px solid #333;
    ">
      <h2 style="margin: 0 0 16px 0; color: #ffffff; font-size: 24px;">
        🔧 Dev Mode: Inject Production Token
      </h2>
      <p style="margin: 0 0 20px 0; color: #b0b0b0; line-height: 1.5;">
        Running on localhost. Paste your production JWT token to authenticate:
      </p>
      <textarea
        id="tokenInput"
        placeholder="Paste JWT token here (starts with eyJ...)"
        style="
          width: 100%;
          min-height: 120px;
          padding: 12px;
          border: 2px solid #404040;
          border-radius: 8px;
          font-family: 'Monaco', 'Courier New', monospace;
          font-size: 13px;
          resize: vertical;
          box-sizing: border-box;
          background: #2a2a2a;
          color: #ffffff;
        "
      ></textarea>
      <div id="errorMsg" style="
        color: #ff6b6b;
        margin: 12px 0 0 0;
        font-size: 14px;
        display: none;
        background: rgba(255, 107, 107, 0.1);
        padding: 8px 12px;
        border-radius: 6px;
        border: 1px solid rgba(255, 107, 107, 0.3);
      "></div>
      <div style="
        margin-top: 20px;
        display: flex;
        gap: 12px;
        justify-content: flex-end;
      ">
        <button id="cancelBtn" style="
          padding: 10px 20px;
          border: 1px solid #404040;
          background: #2a2a2a;
          color: #ffffff;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
          font-weight: 500;
          transition: background 0.2s;
        ">
          Cancel
        </button>
        <button id="injectBtn" style="
          padding: 10px 20px;
          border: none;
          background: #3b82f6;
          color: white;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
          font-weight: 500;
          transition: background 0.2s;
        ">
          Inject Token
        </button>
      </div>
      <p style="
        margin: 20px 0 0 0;
        padding-top: 20px;
        border-top: 1px solid #333;
        color: #808080;
        font-size: 12px;
        line-height: 1.5;
      ">
        <strong style="color: #b0b0b0;">How to get token:</strong><br>
        1. Open production app in another tab<br>
        2. Open console and run: <code style="background: #2a2a2a; color: #60a5fa; padding: 2px 6px; border-radius: 3px; border: 1px solid #404040;">copy(JSON.parse(localStorage.authsec_session_v2).token)</code><br>
        3. Paste here (token will be in your clipboard)<br>
        <span style="color: #999; font-size: 11px;">⚠️ Make sure to copy the ENTIRE token (it's very long, ~4000+ chars)</span>
      </p>
    </div>
  `;

  document.body.appendChild(popup);

  const tokenInput = popup.querySelector('#tokenInput') as HTMLTextAreaElement;
  const injectBtn = popup.querySelector('#injectBtn') as HTMLButtonElement;
  const cancelBtn = popup.querySelector('#cancelBtn') as HTMLButtonElement;
  const errorMsg = popup.querySelector('#errorMsg') as HTMLDivElement;

  // Add hover effects
  injectBtn.addEventListener('mouseenter', () => {
    injectBtn.style.background = '#2563eb';
  });
  injectBtn.addEventListener('mouseleave', () => {
    injectBtn.style.background = '#3b82f6';
  });

  cancelBtn.addEventListener('mouseenter', () => {
    cancelBtn.style.background = '#333';
  });
  cancelBtn.addEventListener('mouseleave', () => {
    cancelBtn.style.background = '#2a2a2a';
  });

  function showError(message: string) {
    errorMsg.textContent = message;
    errorMsg.style.display = 'block';
  }

  function hideError() {
    errorMsg.style.display = 'none';
  }

  injectBtn.addEventListener('click', () => {
    hideError();
    const token = tokenInput.value.trim();

    if (!token) {
      showError('Please enter a token');
      return;
    }

    if (!token.startsWith('eyJ')) {
      showError(`Invalid JWT format. Token should start with "eyJ" but starts with "${token.substring(0, 10)}..."`);
      return;
    }

    const parts = token.split('.');
    if (parts.length !== 3) {
      showError(`Invalid JWT structure. Expected 3 parts (header.payload.signature), found ${parts.length} parts. Token appears to be truncated - make sure you copy the entire token!`);
      return;
    }

    try {
      const sessionData = createSessionFromToken(token);

      if (!sessionData) {
        showError('Failed to decode token. Check browser console for details.');
        return;
      }

      saveSession(sessionData);

      // Remove popup and reload
      popup.remove();
      window.location.reload();
    } catch (error) {
      showError(`Error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  });

  cancelBtn.addEventListener('click', () => {
    popup.remove();
  });

  // Focus textarea
  tokenInput.focus();
}

export function checkAndShowTokenInjector(): void {
  // Only run on localhost
  const isLocalhost =
    window.location.hostname === 'localhost' ||
    window.location.hostname === '127.0.0.1' ||
    window.location.hostname.startsWith('192.168.');

  if (!isLocalhost) {
    return;
  }

  // Check if session already exists
  const existingSession = localStorage.getItem('authsec_session_v2');

  if (existingSession) {
    try {
      const session = JSON.parse(existingSession);
      // Check if session is expired
      if (session.expiresAt && session.expiresAt > Date.now()) {
        console.log('✅ Valid session found, using existing token');
        return;
      }
    } catch (error) {
      // Invalid session, continue to show popup
    }
  }

  // No valid session, show popup
  console.log('🔧 Dev mode: No valid session found, showing token injector');
  showTokenInjectorPopup();
}

// Export for manual use in console
(window as any).injectToken = (token: string) => {
  const sessionData = createSessionFromToken(token);
  if (sessionData) {
    saveSession(sessionData);
    console.log('✅ Token injected! Reloading...');
    window.location.reload();
  } else {
    console.error('❌ Invalid token');
  }
};

(window as any).clearSession = () => {
  localStorage.removeItem('authsec_session_v2');
  console.log('✅ Session cleared! Reloading...');
  window.location.reload();
};
