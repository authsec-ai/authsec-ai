/**
 * CIBA Devices List Component - Compact Professional UI
 */

import React from "react";
import { motion } from "framer-motion";
import { IconDevices, IconTrash, IconRefresh, IconAlertCircle } from "@tabler/icons-react";
import {
  useLazyGetCIBADevicesQuery,
  useDeleteCIBADeviceMutation,
  type CIBADevice,
} from "../../../app/api/deviceApi";

interface CIBADevicesListProps {
  token: string;
  onBack: () => void;
}

const formatDate = (timestamp: number | null): string => {
  if (!timestamp) return "Never";
  return new Date(timestamp * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  });
};

const getPlatformIcon = (platform: string) => {
  switch (platform.toLowerCase()) {
    case "android":
      return "🤖";
    case "ios":
      return "🍎";
    default:
      return "📱";
  }
};

export const CIBADevicesList: React.FC<CIBADevicesListProps> = ({ token, onBack }) => {
  const [fetchDevices, { data, isLoading, error, isFetching }] = useLazyGetCIBADevicesQuery();
  const [deleteDevice, { isLoading: isDeleting }] = useDeleteCIBADeviceMutation();
  const [deletingId, setDeletingId] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (token) {
      fetchDevices({ token });
    }
  }, [token, fetchDevices]);

  const handleDelete = async (deviceId: string, deviceName: string) => {
    if (!confirm(`Deactivate "${deviceName}"?`)) return;

    setDeletingId(deviceId);
    try {
      await deleteDevice({ token, deviceId }).unwrap();
      fetchDevices({ token });
    } catch (err) {
      console.error("Failed to delete device:", err);
      alert("Failed to deactivate device.");
    } finally {
      setDeletingId(null);
    }
  };

  const handleRefresh = () => {
    fetchDevices({ token });
  };

  const devices = data?.devices || [];

  return (
    <motion.div
      initial={{ opacity: 0, x: 10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: -10 }}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <button
            onClick={onBack}
            className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded transition-colors"
          >
            <IconDevices className="w-4 h-4" />
          </button>
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
            CIBA Devices
          </span>
          <span className="text-xs text-slate-400">
            {devices.length > 0 && `(${devices.length})`}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={handleRefresh}
            disabled={isFetching}
            className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded transition-colors disabled:opacity-50"
          >
            <IconRefresh className={`w-3.5 h-3.5 ${isFetching ? "animate-spin" : ""}`} />
          </button>
          <button
            onClick={onBack}
            className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded transition-colors"
          >
            <span className="text-xs">← Back</span>
          </button>
        </div>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="flex items-center justify-center py-6">
          <div className="w-5 h-5 border-2 border-slate-200 border-t-blue-500 rounded-full animate-spin" />
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="flex items-center gap-2 p-2 bg-red-50 dark:bg-red-900/20 rounded-md text-xs">
          <IconAlertCircle className="w-3.5 h-3.5 text-red-500" />
          <span className="text-red-600 dark:text-red-400">Failed to load</span>
          <button
            onClick={handleRefresh}
            className="ml-auto text-red-600 hover:text-red-700 underline"
          >
            Retry
          </button>
        </div>
      )}

      {/* Empty State */}
      {!isLoading && !error && devices.length === 0 && (
        <div className="text-center py-6">
          <IconDevices className="w-8 h-8 mx-auto text-slate-300 dark:text-slate-600 mb-2" />
          <p className="text-xs text-slate-500 dark:text-slate-400">No CIBA devices registered</p>
        </div>
      )}

      {/* Devices List */}
      {!isLoading && !error && devices.length > 0 && (
        <div className="space-y-1.5 max-h-[200px] overflow-y-auto">
          {devices.map((device: CIBADevice) => (
            <div
              key={device.id}
              className="group flex items-center gap-2 p-2 rounded-md hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
            >
              <span className="text-base">{getPlatformIcon(device.platform)}</span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium text-slate-700 dark:text-slate-300 truncate">
                    {device.device_name}
                  </span>
                  {device.is_active && (
                    <span className="w-1.5 h-1.5 bg-green-500 rounded-full flex-shrink-0" />
                  )}
                </div>
                <p className="text-[10px] text-slate-400 truncate">
                  {device.platform} · {device.device_model} · {formatDate(device.created_at)}
                </p>
              </div>
              <button
                onClick={() => handleDelete(device.id, device.device_name)}
                disabled={isDeleting && deletingId === device.id}
                className="p-1 opacity-0 group-hover:opacity-100 text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-all"
              >
                {isDeleting && deletingId === device.id ? (
                  <div className="w-3 h-3 border border-red-300 border-t-red-500 rounded-full animate-spin" />
                ) : (
                  <IconTrash className="w-3 h-3" />
                )}
              </button>
            </div>
          ))}
        </div>
      )}
    </motion.div>
  );
};

export default CIBADevicesList;
