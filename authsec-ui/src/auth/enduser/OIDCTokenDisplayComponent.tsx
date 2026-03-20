/**
 * OIDC Token Display Component
 *
 * Displays the final token for OIDC end-user flow (no storage)
 */

import React, { useState } from "react";
import { Button } from "../../components/ui/button";
import { IconCopy, IconCheck, IconCircleCheck } from "@tabler/icons-react";
import { DeviceManagementPanel } from "./device-management";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface OIDCTokenDisplayComponentProps {
  token: string;
  email: string;
}

export function OIDCTokenDisplayComponent({ token, email }: OIDCTokenDisplayComponentProps) {
  const [copied, setCopied] = useState(false);

  const handleCopyToken = async () => {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error("Failed to copy token:", error);
    }
  };

  return (
    <div className="space-y-5">
      <AuthStepHeader
        title="Authentication successful"
        subtitle="Your sign-in is complete and token is ready."
        meta={
          email ? (
            <span className="inline-flex items-center gap-2">
              <IconCircleCheck className="h-4 w-4 text-green-600" />
              Authenticated as {email}
            </span>
          ) : undefined
        }
      />

      <div className="space-y-3 auth-callout">
        <div className="space-y-2">
          <div className="text-sm font-medium text-slate-800">Access Token</div>
          <div className="max-h-60 overflow-auto rounded-md border border-slate-200 bg-white p-3 font-mono text-xs break-all text-slate-700">
            {token}
          </div>
        </div>

        <Button onClick={handleCopyToken} variant="outline" className="h-11 w-full">
          {copied ? (
            <>
              <IconCheck className="mr-2 h-4 w-4" />
              Copied
            </>
          ) : (
            <>
              <IconCopy className="mr-2 h-4 w-4" />
              Copy Token
            </>
          )}
        </Button>
      </div>

      {token && (
        <div className="auth-panel-divider pt-4">
          <DeviceManagementPanel token={token} />
        </div>
      )}
    </div>
  );
}
