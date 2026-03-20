import * as LocalAuthentication from 'expo-local-authentication';

/**
 * Check if biometric authentication is available
 */
export const isBiometricAvailable = async (): Promise<boolean> => {
  try {
    const hasHardware = await LocalAuthentication.hasHardwareAsync();
    const isEnrolled = await LocalAuthentication.isEnrolledAsync();
    return hasHardware && isEnrolled;
  } catch (error) {
    console.log('Biometric not supported:', error);
    return false;
  }
};

/**
 * Authenticate with biometric (Face ID / Touch ID / Fingerprint)
 */
export const authenticateWithBiometric = async (message?: string): Promise<boolean> => {
  try {
    const result = await LocalAuthentication.authenticateAsync({
      promptMessage: message || 'Authenticate to access the app',
      fallbackLabel: 'Use Passcode',
      cancelLabel: 'Cancel',
    });

    return result.success;
  } catch (error: any) {
    console.log('Biometric authentication failed:', error);
    return false;
  }
};

/**
 * Get biometric type name for UI display
 */
export const getBiometricType = async (): Promise<string> => {
  try {
    const types = await LocalAuthentication.supportedAuthenticationTypesAsync();

    if (types.includes(LocalAuthentication.AuthenticationType.FACIAL_RECOGNITION)) {
      return 'Face ID';
    }
    if (types.includes(LocalAuthentication.AuthenticationType.FINGERPRINT)) {
      return 'Fingerprint';
    }
    if (types.includes(LocalAuthentication.AuthenticationType.IRIS)) {
      return 'Iris';
    }

    return 'Biometric';
  } catch (error) {
    return 'Biometric';
  }
};
