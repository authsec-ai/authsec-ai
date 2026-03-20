/**
 * Cross-tenant handoff helpers for moving minimal auth context across domains
 */
const HANDOFF_MAX_AGE_MS = 5 * 60 * 1000; // 5 minutes

export interface AdminHandoffPayload {
  email: string;
  tenant_domain?: string;
  tenant_id?: string;
  first_login?: boolean;
  target?: "login" | "webauthn";
  flow_stage?: string;
  ts?: number;
}

export const encodeHandoff = (payload: AdminHandoffPayload): string => {
  try {
    const json = JSON.stringify({ ...payload, ts: Date.now() });
    return btoa(json).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  } catch (error) {
    console.error("[Handoff] Failed to encode payload:", error);
    return "";
  }
};

export const decodeHandoff = <T extends AdminHandoffPayload>(token: string): T | null => {
  try {
    const padded = token + "===".slice((token.length + 3) % 4);
    const base64 = padded.replace(/-/g, "+").replace(/_/g, "/");
    const json = atob(base64);
    const payload = JSON.parse(json) as T;

    if (payload.ts && Date.now() - payload.ts > HANDOFF_MAX_AGE_MS) {
      console.warn("[Handoff] Token expired");
      return null;
    }

    return payload;
  } catch (error) {
    console.error("[Handoff] Failed to decode token:", error);
    return null;
  }
};
