/**
 * Device Management Types
 * Shared types for TOTP and CIBA device management
 */

export type DeviceType = "microsoft_auth" | "google_auth" | "mobile" | "other";

export interface DeviceTypeOption {
  value: DeviceType;
  label: string;
  description: string;
  icon: string;
}

export const DEVICE_TYPE_OPTIONS: DeviceTypeOption[] = [
  {
    value: "microsoft_auth",
    label: "Microsoft Authenticator",
    description: "Use Microsoft Authenticator app",
    icon: "microsoft",
  },
  {
    value: "google_auth",
    label: "Google Authenticator",
    description: "Use Google Authenticator app",
    icon: "google",
  },
  {
    value: "mobile",
    label: "Other Authenticator",
    description: "Use any TOTP-compatible authenticator app",
    icon: "smartphone",
  },
];

export type DeviceManagementView =
  | "main"
  | "add-device"
  | "show-devices"
  | "totp-devices"
  | "ciba-devices"
  | "qr-scan"
  | "confirm-totp";

export interface DeviceManagementState {
  currentView: DeviceManagementView;
  selectedDeviceType?: DeviceType;
  pendingDevice?: {
    deviceId: string;
    deviceName: string;
    secret: string;
    qrCodeUrl: string;
    backupCodes: string[];
  };
}
