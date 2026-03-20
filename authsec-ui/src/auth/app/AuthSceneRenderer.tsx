import { AdminLoginHubPage } from "../adminauth/AdminLoginHubPage";
import { OTPVerificationPage } from "../adminauth/OTPVerificationPage";
import { WebAuthnPage } from "../webauthn/WebAuthnPage";
import AdminOIDCCallbackPage from "../adminauth/AdminOIDCCallbackPage";
import { CreateWorkspacePage } from "../adminauth/CreateWorkspacePage";
import { OIDCLoginPage } from "../enduser/OIDCLoginPage";
import OIDCCallbackPage from "../enduser/OIDCCallbackPage";
import type { AuthRouteDescriptor } from "./auth-flow-types";

export interface AuthSceneRendererProps {
  descriptor: AuthRouteDescriptor;
}

export function AuthSceneRenderer({ descriptor }: AuthSceneRendererProps) {
  switch (descriptor.sceneId) {
    case "admin-login":
      return <AdminLoginHubPage />;
    case "admin-verify-otp":
      return <OTPVerificationPage />;
    case "admin-webauthn":
      return <WebAuthnPage />;
    case "admin-oidc-callback":
      return <AdminOIDCCallbackPage />;
    case "admin-create-workspace":
      return <CreateWorkspacePage />;
    case "enduser-login":
      return <OIDCLoginPage />;
    case "enduser-oidc-callback":
      return <OIDCCallbackPage />;
    default:
      return null;
  }
}

export default AuthSceneRenderer;
