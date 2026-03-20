import * as SecureStore from 'expo-secure-store';

const TOKEN_KEY = 'authsec_token';
const DEVICE_TOKEN_KEY = 'authsec_device_token';
const EMAIL_KEY = 'authsec_email';
const APP_PIN_KEY = 'authsec_app_pin';
const BIOMETRIC_ENABLED_KEY = 'authsec_biometric_enabled';
const BIOMETRIC_AUTH_REQUESTS_KEY = 'authsec_biometric_auth_requests';
const PIN_SETUP_COMPLETED_KEY = 'authsec_pin_setup_completed';
const APP_LOCK_ENABLED_KEY = 'authsec_app_lock_enabled';
const CLIENT_ID_KEY = 'authsec_client_id';

/**
 * Store JWT token securely
 */
export const storeToken = async (token: string): Promise<void> => {
  try {
    await SecureStore.setItemAsync(TOKEN_KEY, token);
  } catch (error) {
    console.error('Error storing token:', error);
  }
};

/**
 * Get stored JWT token
 */
export const getStoredToken = async (): Promise<string | null> => {
  try {
    return await SecureStore.getItemAsync(TOKEN_KEY);
  } catch (error) {
    console.error('Error getting token:', error);
    return null;
  }
};

/**
 * Store device push token
 */
export const storeDeviceToken = async (token: string): Promise<void> => {
  try {
    await SecureStore.setItemAsync(DEVICE_TOKEN_KEY, token);
  } catch (error) {
    console.error('Error storing device token:', error);
  }
};

/**
 * Get stored device token
 */
export const getDeviceToken = async (): Promise<string | null> => {
  try {
    return await SecureStore.getItemAsync(DEVICE_TOKEN_KEY);
  } catch (error) {
    console.error('Error getting device token:', error);
    return null;
  }
};

/**
 * Store email
 */
export const storeEmail = async (email: string): Promise<void> => {
  try {
    await SecureStore.setItemAsync(EMAIL_KEY, email);
  } catch (error) {
    console.error('Error storing email:', error);
  }
};

/**
 * Get stored email
 */
export const getStoredEmail = async (): Promise<string | null> => {
  try {
    return await SecureStore.getItemAsync(EMAIL_KEY);
  } catch (error) {
    console.error('Error getting email:', error);
    return null;
  }
};

/**
 * Store app PIN securely
 */
export const storeAppPin = async (pin: string): Promise<void> => {
  try {
    await SecureStore.setItemAsync(APP_PIN_KEY, pin);
  } catch (error) {
    console.error('Error storing app PIN:', error);
  }
};

/**
 * Get stored app PIN
 */
export const getAppPin = async (): Promise<string | null> => {
  try {
    return await SecureStore.getItemAsync(APP_PIN_KEY);
  } catch (error) {
    console.error('Error getting app PIN:', error);
    return null;
  }
};

/**
 * Store biometric enabled preference
 */
export const setBiometricEnabled = async (enabled: boolean): Promise<void> => {
  try {
    await SecureStore.setItemAsync(BIOMETRIC_ENABLED_KEY, enabled ? 'true' : 'false');
  } catch (error) {
    console.error('Error storing biometric preference:', error);
  }
};

/**
 * Get biometric enabled preference
 */
export const getBiometricEnabled = async (): Promise<boolean> => {
  try {
    const value = await SecureStore.getItemAsync(BIOMETRIC_ENABLED_KEY);
    return value === 'true';
  } catch (error) {
    console.error('Error getting biometric preference:', error);
    return false;
  }
};

/**
 * Store PIN setup completed status
 */
export const setPinSetupCompleted = async (completed: boolean): Promise<void> => {
  try {
    await SecureStore.setItemAsync(PIN_SETUP_COMPLETED_KEY, completed ? 'true' : 'false');
  } catch (error) {
    console.error('Error storing PIN setup status:', error);
  }
};

/**
 * Check if PIN setup is completed
 */
export const isPinSetupCompleted = async (): Promise<boolean> => {
  try {
    const value = await SecureStore.getItemAsync(PIN_SETUP_COMPLETED_KEY);
    return value === 'true';
  } catch (error) {
    console.error('Error getting PIN setup status:', error);
    return false;
  }
};

/**
 * Store app lock enabled preference
 */
export const setAppLockEnabled = async (enabled: boolean): Promise<void> => {
  try {
    await SecureStore.setItemAsync(APP_LOCK_ENABLED_KEY, enabled ? 'true' : 'false');
  } catch (error) {
    console.error('Error storing app lock preference:', error);
  }
};

/**
 * Get app lock enabled preference
 */
export const getAppLockEnabled = async (): Promise<boolean> => {
  try {
    const value = await SecureStore.getItemAsync(APP_LOCK_ENABLED_KEY);
    return value === 'true';
  } catch (error) {
    console.error('Error getting app lock preference:', error);
    return false;
  }
};

/**
 * Store biometric for auth requests preference
 */
export const setBiometricAuthRequests = async (enabled: boolean): Promise<void> => {
  try {
    await SecureStore.setItemAsync(BIOMETRIC_AUTH_REQUESTS_KEY, enabled ? 'true' : 'false');
  } catch (error) {
    console.error('Error storing biometric auth requests preference:', error);
  }
};

/**
 * Get biometric for auth requests preference
 */
export const getBiometricAuthRequests = async (): Promise<boolean> => {
  try {
    const value = await SecureStore.getItemAsync(BIOMETRIC_AUTH_REQUESTS_KEY);
    return value === 'true';
  } catch (error) {
    console.error('Error getting biometric auth requests preference:', error);
    return false;
  }
};

/**
 * Store client ID
 */
export const storeClientId = async (clientId: string): Promise<void> => {
  try {
    await SecureStore.setItemAsync(CLIENT_ID_KEY, clientId);
  } catch (error) {
    console.error('Error storing client ID:', error);
  }
};

/**
 * Get stored client ID
 */
export const getStoredClientId = async (): Promise<string | null> => {
  try {
    return await SecureStore.getItemAsync(CLIENT_ID_KEY);
  } catch (error) {
    console.error('Error getting client ID:', error);
    return null;
  }
};

/**
 * Clear stored client ID
 */
export const clearClientId = async (): Promise<void> => {
  try {
    await SecureStore.deleteItemAsync(CLIENT_ID_KEY);
  } catch (error) {
    console.error('Error clearing client ID:', error);
  }
};


/**
 * Clear all stored data (logout)
 */
export const clearStorage = async (): Promise<void> => {
  try {
    await SecureStore.deleteItemAsync(TOKEN_KEY);
    await SecureStore.deleteItemAsync(EMAIL_KEY);
    await SecureStore.deleteItemAsync(APP_PIN_KEY);
    await SecureStore.deleteItemAsync(BIOMETRIC_ENABLED_KEY);
    await SecureStore.deleteItemAsync(BIOMETRIC_AUTH_REQUESTS_KEY);
    await SecureStore.deleteItemAsync(PIN_SETUP_COMPLETED_KEY);
    await SecureStore.deleteItemAsync(APP_LOCK_ENABLED_KEY);
    await SecureStore.deleteItemAsync(CLIENT_ID_KEY);
    // Keep device token for future logins
  } catch (error) {
    console.error('Error clearing storage:', error);
  }
};

// Activity Log Types
export interface ActivityLog {
  id: string;
  type: 'auth_approved' | 'auth_denied' | 'totp_added' | 'totp_deleted';
  title: string;
  description: string;
  timestamp: number;
  metadata?: any;
}

// Notification Types
export interface AppNotification {
  id: string;
  title: string;
  body: string;
  timestamp: number;
  read: boolean;
  data?: any;
}

const ACTIVITY_LOG_KEY = 'authsec_activity_log';
const NOTIFICATIONS_KEY = 'authsec_notifications';

/**
 * Store activity log entry
 */
export const addActivityLog = async (log: Omit<ActivityLog, 'id' | 'timestamp'>): Promise<void> => {
  try {
    const logs = await getActivityLogs();
    const newLog: ActivityLog = {
      ...log,
      id: Date.now().toString(),
      timestamp: Date.now(),
    };
    logs.unshift(newLog); // Add to beginning

    // Keep only last 100 entries
    const trimmedLogs = logs.slice(0, 100);
    await SecureStore.setItemAsync(ACTIVITY_LOG_KEY, JSON.stringify(trimmedLogs));
  } catch (error) {
    console.error('Error adding activity log:', error);
  }
};

/**
 * Get all activity logs
 */
export const getActivityLogs = async (): Promise<ActivityLog[]> => {
  try {
    const data = await SecureStore.getItemAsync(ACTIVITY_LOG_KEY);
    return data ? JSON.parse(data) : [];
  } catch (error) {
    console.error('Error getting activity logs:', error);
    return [];
  }
};

/**
 * Clear all activity logs
 */
export const clearActivityLogs = async (): Promise<void> => {
  try {
    await SecureStore.deleteItemAsync(ACTIVITY_LOG_KEY);
  } catch (error) {
    console.error('Error clearing activity logs:', error);
  }
};

/**
 * Store notification
 */
export const addNotification = async (notification: Omit<AppNotification, 'id' | 'timestamp' | 'read'>): Promise<void> => {
  try {
    const notifications = await getNotifications();
    const newNotification: AppNotification = {
      ...notification,
      id: Date.now().toString(),
      timestamp: Date.now(),
      read: false,
    };
    notifications.unshift(newNotification); // Add to beginning

    // Keep only last 50 notifications
    const trimmedNotifications = notifications.slice(0, 50);
    await SecureStore.setItemAsync(NOTIFICATIONS_KEY, JSON.stringify(trimmedNotifications));
  } catch (error) {
    console.error('Error adding notification:', error);
  }
};

/**
 * Get all notifications
 */
export const getNotifications = async (): Promise<AppNotification[]> => {
  try {
    const data = await SecureStore.getItemAsync(NOTIFICATIONS_KEY);
    return data ? JSON.parse(data) : [];
  } catch (error) {
    console.error('Error getting notifications:', error);
    return [];
  }
};

/**
 * Mark notification as read
 */
export const markNotificationAsRead = async (notificationId: string): Promise<void> => {
  try {
    const notifications = await getNotifications();
    const updated = notifications.map(n =>
      n.id === notificationId ? { ...n, read: true } : n
    );
    await SecureStore.setItemAsync(NOTIFICATIONS_KEY, JSON.stringify(updated));
  } catch (error) {
    console.error('Error marking notification as read:', error);
  }
};

/**
 * Mark all notifications as read
 */
export const markAllNotificationsAsRead = async (): Promise<void> => {
  try {
    const notifications = await getNotifications();
    const updated = notifications.map(n => ({ ...n, read: true }));
    await SecureStore.setItemAsync(NOTIFICATIONS_KEY, JSON.stringify(updated));
  } catch (error) {
    console.error('Error marking all notifications as read:', error);
  }
};

/**
 * Get unread notification count
 */
export const getUnreadNotificationCount = async (): Promise<number> => {
  try {
    const notifications = await getNotifications();
    return notifications.filter(n => !n.read).length;
  } catch (error) {
    console.error('Error getting unread count:', error);
    return 0;
  }
};

/**
 * Clear all notifications
 */
export const clearNotifications = async (): Promise<void> => {
  try {
    await SecureStore.deleteItemAsync(NOTIFICATIONS_KEY);
  } catch (error) {
    console.error('Error clearing notifications:', error);
  }
};
