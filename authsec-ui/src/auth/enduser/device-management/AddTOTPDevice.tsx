/**
 * Add TOTP Device - Compact Professional UI
 * Multi-step: Details → QR Scan → Confirm
 */

import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  IconDeviceMobile,
  IconQrcode,
  IconShieldCheck,
  IconCopy,
  IconCheck,
  IconAlertCircle,
  IconKey,
} from "@tabler/icons-react";
import {
  useRegisterTOTPDeviceMutation,
  useConfirmTOTPDeviceMutation,
} from "../../../app/api/deviceApi";
import { DEVICE_TYPE_OPTIONS, type DeviceType } from "./types";
import QRCode from "qrcode";

interface AddTOTPDeviceProps {
  token: string;
  onBack: () => void;
  onComplete: () => void;
}

type Step = "details" | "qr-scan" | "confirm" | "success";

export const AddTOTPDevice: React.FC<AddTOTPDeviceProps> = ({ token, onBack, onComplete }) => {
  const [step, setStep] = useState<Step>("details");
  const [deviceName, setDeviceName] = useState("");
  const [deviceType, setDeviceType] = useState<DeviceType>("microsoft_auth");
  const [totpCode, setTotpCode] = useState("");
  const [qrDataUrl, setQrDataUrl] = useState<string>("");
  const [secretCopied, setSecretCopied] = useState(false);
  const [backupCodesCopied, setBackupCodesCopied] = useState(false);

  const [pendingDevice, setPendingDevice] = useState<{
    deviceId: string;
    secret: string;
    qrCodeUrl: string;
    backupCodes: string[];
  } | null>(null);

  const [registerDevice, { isLoading: isRegistering, error: registerError }] =
    useRegisterTOTPDeviceMutation();
  const [confirmDevice, { isLoading: isConfirming, error: confirmError }] =
    useConfirmTOTPDeviceMutation();

  const generateQRCode = async (url: string) => {
    try {
      const dataUrl = await QRCode.toDataURL(url, { width: 180, margin: 1 });
      setQrDataUrl(dataUrl);
    } catch (err) {
      console.error("Failed to generate QR code:", err);
    }
  };

  const handleRegister = async () => {
    if (!deviceName.trim()) return;
    try {
      const result = await registerDevice({
        token,
        data: { device_name: deviceName.trim(), device_type: deviceType },
      }).unwrap();
      if (result.success) {
        setPendingDevice({
          deviceId: result.device_id,
          secret: result.secret,
          qrCodeUrl: result.qr_code_url,
          backupCodes: result.backup_codes,
        });
        await generateQRCode(result.qr_code_url);
        setStep("qr-scan");
      }
    } catch (err) {
      console.error("Registration failed:", err);
    }
  };

  const handleConfirm = async () => {
    if (!totpCode.trim() || totpCode.length !== 6 || !pendingDevice) return;
    try {
      const result = await confirmDevice({
        token,
        data: { device_id: pendingDevice.deviceId, totp_code: totpCode.trim() },
      }).unwrap();
      if (result.success) setStep("success");
    } catch (err) {
      console.error("Confirmation failed:", err);
    }
  };

  const copyToClipboard = async (text: string, type: "secret" | "backup") => {
    try {
      await navigator.clipboard.writeText(text);
      if (type === "secret") {
        setSecretCopied(true);
        setTimeout(() => setSecretCopied(false), 2000);
      } else {
        setBackupCodesCopied(true);
        setTimeout(() => setBackupCodesCopied(false), 2000);
      }
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const steps = ["details", "qr-scan", "confirm"];
  const currentStepIdx = steps.indexOf(step);

  return (
    <motion.div
      initial={{ opacity: 0, x: 10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: -10 }}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <IconDeviceMobile className="w-4 h-4 text-slate-400" />
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">Add Device</span>
        </div>
        {step !== "success" && (
          <button
            onClick={onBack}
            className="text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
          >
            ← Back
          </button>
        )}
      </div>

      {/* Progress dots */}
      {step !== "success" && (
        <div className="flex items-center justify-center gap-1.5 mb-4">
          {steps.map((s, i) => (
            <div
              key={s}
              className={`w-1.5 h-1.5 rounded-full transition-colors ${
                i <= currentStepIdx ? "bg-blue-500" : "bg-slate-200 dark:bg-slate-700"
              }`}
            />
          ))}
        </div>
      )}

      <AnimatePresence mode="wait">
        {/* Step 1: Details */}
        {step === "details" && (
          <motion.div
            key="details"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="space-y-3"
          >
            <div>
              <label className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-1 block">
                Device Name
              </label>
              <input
                value={deviceName}
                onChange={(e) => setDeviceName(e.target.value)}
                placeholder="My Phone"
                className="w-full px-2.5 py-1.5 text-sm rounded-md border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder:text-slate-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-1 block">
                App Type
              </label>
              <div className="flex gap-1.5">
                {DEVICE_TYPE_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => setDeviceType(opt.value)}
                    className={`flex-1 py-1.5 px-2 text-xs rounded-md border transition-all ${
                      deviceType === opt.value
                        ? "border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
                        : "border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:border-slate-300"
                    }`}
                  >
                    {opt.value === "microsoft_auth" && "🔐"}
                    {opt.value === "google_auth" && "🔑"}
                    {opt.value === "mobile" && "📱"}
                    <span className="ml-1">{opt.label.split(" ")[0]}</span>
                  </button>
                ))}
              </div>
            </div>

            {registerError && (
              <div className="flex items-center gap-1.5 p-2 bg-red-50 dark:bg-red-900/20 rounded text-xs text-red-600 dark:text-red-400">
                <IconAlertCircle className="w-3.5 h-3.5" />
                Registration failed
              </div>
            )}

            <button
              onClick={handleRegister}
              disabled={!deviceName.trim() || isRegistering}
              className="w-full py-1.5 px-3 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors"
            >
              {isRegistering ? "Setting up..." : "Continue →"}
            </button>
          </motion.div>
        )}

        {/* Step 2: QR Scan */}
        {step === "qr-scan" && pendingDevice && (
          <motion.div
            key="qr-scan"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="space-y-3"
          >
            <div className="text-center">
              <IconQrcode className="w-5 h-5 mx-auto text-blue-500 mb-1" />
              <p className="text-xs text-slate-600 dark:text-slate-400">
                Scan with your authenticator app
              </p>
            </div>

            <div className="flex justify-center">
              <div className="p-2 bg-white rounded-lg shadow-sm">
                {qrDataUrl ? (
                  <img src={qrDataUrl} alt="QR" className="w-36 h-36" />
                ) : (
                  <div className="w-36 h-36 flex items-center justify-center">
                    <div className="w-5 h-5 border-2 border-slate-200 border-t-blue-500 rounded-full animate-spin" />
                  </div>
                )}
              </div>
            </div>

            {/* Secret */}
            <div className="p-2 bg-slate-50 dark:bg-slate-800/50 rounded text-xs">
              <div className="flex items-center justify-between mb-1">
                <span className="text-slate-500">Manual entry:</span>
                <button
                  onClick={() => copyToClipboard(pendingDevice.secret, "secret")}
                  className="p-1 hover:bg-slate-200 dark:hover:bg-slate-700 rounded"
                >
                  {secretCopied ? (
                    <IconCheck className="w-3 h-3 text-green-500" />
                  ) : (
                    <IconCopy className="w-3 h-3 text-slate-400" />
                  )}
                </button>
              </div>
              <code className="block text-[10px] font-mono text-slate-600 dark:text-slate-400 break-all">
                {pendingDevice.secret}
              </code>
            </div>

            {/* Backup codes */}
            <details className="p-2 bg-amber-50 dark:bg-amber-900/20 rounded text-xs">
              <summary className="cursor-pointer flex items-center gap-1 text-amber-700 dark:text-amber-400">
                <IconKey className="w-3 h-3" />
                Backup codes (save these!)
              </summary>
              <div className="mt-2 grid grid-cols-2 gap-1">
                {pendingDevice.backupCodes.map((code, i) => (
                  <code
                    key={i}
                    className="py-0.5 px-1 bg-white dark:bg-slate-900 rounded text-[10px] font-mono text-center"
                  >
                    {code}
                  </code>
                ))}
              </div>
              <button
                onClick={() => copyToClipboard(pendingDevice.backupCodes.join("\n"), "backup")}
                className="w-full mt-2 py-1 text-[10px] text-amber-700 hover:bg-amber-100 dark:hover:bg-amber-900/30 rounded flex items-center justify-center gap-1"
              >
                {backupCodesCopied ? (
                  <IconCheck className="w-3 h-3" />
                ) : (
                  <IconCopy className="w-3 h-3" />
                )}
                {backupCodesCopied ? "Copied!" : "Copy all"}
              </button>
            </details>

            <button
              onClick={() => setStep("confirm")}
              className="w-full py-1.5 px-3 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
            >
              I've scanned it →
            </button>
          </motion.div>
        )}

        {/* Step 3: Confirm */}
        {step === "confirm" && (
          <motion.div
            key="confirm"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="space-y-3"
          >
            <div className="text-center">
              <IconShieldCheck className="w-5 h-5 mx-auto text-green-500 mb-1" />
              <p className="text-xs text-slate-600 dark:text-slate-400">
                Enter 6-digit code from your app
              </p>
            </div>

            <input
              value={totpCode}
              onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="000000"
              className="w-full py-2 px-3 text-center text-lg font-mono tracking-[0.3em] rounded-md border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder:text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
              maxLength={6}
              autoFocus
            />

            {confirmError && (
              <div className="flex items-center gap-1.5 p-2 bg-red-50 dark:bg-red-900/20 rounded text-xs text-red-600 dark:text-red-400">
                <IconAlertCircle className="w-3.5 h-3.5" />
                Invalid code
              </div>
            )}

            <div className="flex gap-2">
              <button
                onClick={() => setStep("qr-scan")}
                className="flex-1 py-1.5 px-3 text-xs font-medium text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 rounded-md transition-colors"
              >
                ← Back
              </button>
              <button
                onClick={handleConfirm}
                disabled={totpCode.length !== 6 || isConfirming}
                className="flex-1 py-1.5 px-3 text-xs font-medium text-white bg-green-600 hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-md transition-colors"
              >
                {isConfirming ? "Verifying..." : "Verify ✓"}
              </button>
            </div>
          </motion.div>
        )}

        {/* Success */}
        {step === "success" && (
          <motion.div
            key="success"
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="text-center py-4"
          >
            <div className="w-10 h-10 mx-auto mb-2 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
              <IconCheck className="w-5 h-5 text-green-600" />
            </div>
            <p className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Device Added!
            </p>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
              "{deviceName}" is ready to use
            </p>
            <button
              onClick={onComplete}
              className="py-1.5 px-4 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
            >
              Done
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};

export default AddTOTPDevice;
