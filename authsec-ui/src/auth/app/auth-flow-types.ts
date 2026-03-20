export type AuthEntrypoint = "admin" | "enduser" | "shared";

export type AuthSceneId =
  | "admin-login"
  | "admin-verify-otp"
  | "admin-webauthn"
  | "admin-oidc-callback"
  | "admin-create-workspace"
  | "enduser-login"
  | "enduser-oidc-callback";

export interface AuthRouteDescriptor {
  path: string;
  sceneId: AuthSceneId;
  entrypoint: AuthEntrypoint;
  title: string;
  description?: string;
}

export interface AuthSceneViewModel {
  descriptor: AuthRouteDescriptor;
  path: string;
}

export interface MfaAdapter {
  kind: "admin" | "oidc";
  supportsWebAuthn: boolean;
  supportsTotp: boolean;
}
