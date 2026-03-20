/**
 * Device Management Panel Component
 * Compact, professional UI for device management
 */

import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  IconPlus,
  IconDeviceMobile,
  IconBell,
  IconChevronRight,
  IconArrowLeft,
} from "@tabler/icons-react";
import { default as TOTPDevicesList } from "./TOTPDevicesList";
import { default as CIBADevicesList } from "./CIBADevicesList";
import { default as AddTOTPDevice } from "./AddTOTPDevice";
import type { DeviceManagementView } from "./types";

interface DeviceManagementPanelProps {
  token: string;
  onClose?: () => void;
}

export const DeviceManagementPanel: React.FC<DeviceManagementPanelProps> = ({ token }) => {
  const [currentView, setCurrentView] = useState<DeviceManagementView>("main");

  const handleNavigate = (view: DeviceManagementView) => {
    setCurrentView(view);
  };

  const handleBack = () => {
    switch (currentView) {
      case "totp-devices":
      case "ciba-devices":
        setCurrentView("show-devices");
        break;
      case "add-device":
      case "show-devices":
        setCurrentView("main");
        break;
      default:
        setCurrentView("main");
    }
  };

  const handleAddDeviceComplete = () => {
    setCurrentView("totp-devices");
  };

  return (
    <div className="mt-4">
      <AnimatePresence mode="wait">
        {/* Main Menu */}
        {currentView === "main" && (
          <motion.div
            key="main"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <div className="flex items-center gap-4">
              <button
                onClick={() => handleNavigate("add-device")}
                className="group flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-md transition-colors"
              >
                <IconPlus className="w-3.5 h-3.5" />
                <span>Add Device</span>
              </button>

              <div className="h-4 w-px bg-slate-200 dark:bg-slate-700" />

              <button
                onClick={() => handleNavigate("totp-devices")}
                className="group flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-md transition-colors"
              >
                <IconDeviceMobile className="w-3.5 h-3.5" />
                <span>TOTP</span>
                <IconChevronRight className="w-3 h-3 opacity-50" />
              </button>

              <button
                onClick={() => handleNavigate("ciba-devices")}
                className="group flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-md transition-colors"
              >
                <IconBell className="w-3.5 h-3.5" />
                <span>CIBA</span>
                <IconChevronRight className="w-3 h-3 opacity-50" />
              </button>
            </div>
          </motion.div>
        )}

        {/* Show Devices Selection */}
        {currentView === "show-devices" && (
          <motion.div
            key="show-devices"
            initial={{ opacity: 0, x: 10 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -10 }}
          >
            <div className="flex items-center gap-3 mb-3">
              <button
                onClick={handleBack}
                className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded transition-colors"
              >
                <IconArrowLeft className="w-4 h-4" />
              </button>
              <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                Select Device Type
              </span>
            </div>

            <div className="flex gap-2">
              <button
                onClick={() => handleNavigate("totp-devices")}
                className="flex-1 flex items-center gap-2 p-2.5 text-left rounded-lg border border-slate-200 dark:border-slate-700 hover:border-blue-400 dark:hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-900/10 transition-all"
              >
                <IconDeviceMobile className="w-4 h-4 text-blue-500" />
                <div>
                  <p className="text-xs font-medium text-slate-700 dark:text-slate-300">TOTP</p>
                  <p className="text-[10px] text-slate-500 dark:text-slate-500">
                    Authenticator apps
                  </p>
                </div>
              </button>

              <button
                onClick={() => handleNavigate("ciba-devices")}
                className="flex-1 flex items-center gap-2 p-2.5 text-left rounded-lg border border-slate-200 dark:border-slate-700 hover:border-blue-400 dark:hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-900/10 transition-all"
              >
                <IconBell className="w-4 h-4 text-blue-500" />
                <div>
                  <p className="text-xs font-medium text-slate-700 dark:text-slate-300">CIBA</p>
                  <p className="text-[10px] text-slate-500 dark:text-slate-500">Push devices</p>
                </div>
              </button>
            </div>
          </motion.div>
        )}

        {/* TOTP Devices List */}
        {currentView === "totp-devices" && <TOTPDevicesList token={token} onBack={handleBack} />}

        {/* CIBA Devices List */}
        {currentView === "ciba-devices" && <CIBADevicesList token={token} onBack={handleBack} />}

        {/* Add Device Flow */}
        {currentView === "add-device" && (
          <AddTOTPDevice token={token} onBack={handleBack} onComplete={handleAddDeviceComplete} />
        )}
      </AnimatePresence>
    </div>
  );
};

export default DeviceManagementPanel;
