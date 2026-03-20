import React, {useState, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  Alert,
  ScrollView,
  TextInput,
} from 'react-native';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import {registerDeviceToken} from '../services/api';
import {getStoredToken, getDeviceToken, storeDeviceToken} from '../services/storage';
import {useTheme} from '../context/ThemeContext';

const DeviceRegistrationScreen = ({navigation}: any) => {
  const {colors, isDark} = useTheme();
  const styles = createStyles(colors, isDark);
  
  const [loading, setLoading] = useState(false);
  const [deviceToken, setDeviceToken] = useState<string | null>(null);
  const [manualToken, setManualToken] = useState('');
  const [registrationStatus, setRegistrationStatus] = useState<string>('');

  useEffect(() => {
    checkDeviceToken();
  }, []);

  const checkDeviceToken = async () => {
    const token = await getDeviceToken();
    if (token) {
      setDeviceToken(token);
      setRegistrationStatus('Device token found in storage');
    } else {
      setRegistrationStatus('No device token found');
    }
  };

  const requestPushPermission = async () => {
    setLoading(true);
    setRegistrationStatus('Requesting push notification permission...');

    try {
      if (!Device.isDevice) {
        Alert.alert(
          'Expo Go Limitation',
          'Push notifications do not work in Expo Go. You need to:\n\n' +
            '1. Build a standalone APK: eas build --platform android\n' +
            '2. Install the APK on your phone\n' +
            '3. Then device registration will work',
        );
        setRegistrationStatus('⚠️ Running in Expo Go - push not supported');
        setLoading(false);
        return;
      }

      const {status: existingStatus} = await Notifications.getPermissionsAsync();
      let finalStatus = existingStatus;

      if (existingStatus !== 'granted') {
        const {status} = await Notifications.requestPermissionsAsync();
        finalStatus = status;
      }

      if (finalStatus !== 'granted') {
        Alert.alert(
          'Permission Denied',
          'Please enable push notifications in Settings',
        );
        setRegistrationStatus('❌ Permission denied');
        setLoading(false);
        return;
      }

      setRegistrationStatus('Getting Expo Push token...');
      const token = (await Notifications.getExpoPushTokenAsync({
        projectId: 'YOUR_EAS_PROJECT_ID'
      })).data;
      console.log('Got Expo Push Token:', token);

      await storeDeviceToken(token);
      setDeviceToken(token);
      setRegistrationStatus('✅ Device token obtained: ' + token.substring(0, 30) + '...');

      Alert.alert('Success', 'Device token obtained! Now register with backend.');
    } catch (error: any) {
      console.error('Error getting push token:', error);
      Alert.alert('Error', error.message);
      setRegistrationStatus('❌ Error: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const registerWithBackend = async () => {
    const token = deviceToken || manualToken;

    if (!token) {
      Alert.alert('Error', 'No device token available. Get token first.');
      return;
    }

    setLoading(true);
    setRegistrationStatus('Registering device with backend...');

    try {
      const authToken = await getStoredToken();
      if (!authToken) {
        Alert.alert('Error', 'Not logged in. Please login first.');
        setLoading(false);
        return;
      }

      await registerDeviceToken(token, authToken);
      setRegistrationStatus('✅ Device registered with backend successfully!');
      Alert.alert(
        'Success',
        'Your device is now registered for push notifications!',
        [
          {
            text: 'OK',
            onPress: () => navigation.goBack(),
          },
        ],
      );
    } catch (error: any) {
      console.error('Registration error:', error);
      Alert.alert('Registration Failed', error.message);
      setRegistrationStatus('❌ Registration failed: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const useManualToken = () => {
    if (!manualToken) {
      Alert.alert('Error', 'Please enter a device token');
      return;
    }
    setDeviceToken(manualToken);
    storeDeviceToken(manualToken);
    setRegistrationStatus('Manual token set: ' + manualToken.substring(0, 30) + '...');
  };

  return (
    <ScrollView style={styles.container}>
      <View style={styles.content}>
        <View style={styles.header}>
          <Text style={styles.title}>Device Registration</Text>
          <Text style={styles.subtitle}>
            Register this device for push notifications
          </Text>
        </View>

        {/* Status Card */}
        <View style={styles.statusCard}>
          <View style={styles.statusHeader}>
            <View style={styles.statusIcon}>
              <Text style={styles.statusIconText}>📱</Text>
            </View>
            <Text style={styles.statusTitle}>Current Status</Text>
          </View>
          <Text style={styles.statusText}>{registrationStatus || 'Ready to register'}</Text>
          {deviceToken && (
            <View style={styles.tokenBox}>
              <Text style={styles.tokenLabel}>Device Token</Text>
              <Text style={styles.tokenText} numberOfLines={2}>
                {deviceToken}
              </Text>
            </View>
          )}
        </View>

        {/* Option 1: Automatic Registration */}
        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <View style={[styles.sectionBadge, {backgroundColor: colors.primary + '20'}]}>
              <Text style={[styles.sectionBadgeText, {color: colors.primary}]}>1</Text>
            </View>
            <View style={styles.sectionTitleContainer}>
              <Text style={styles.sectionTitle}>Automatic Registration</Text>
              <Text style={styles.sectionBadgeLabel}>Recommended</Text>
            </View>
          </View>
          <Text style={styles.sectionDesc}>
            Get device token and register automatically
          </Text>
          <TouchableOpacity
            style={[styles.button, styles.buttonPrimary, loading && styles.buttonDisabled]}
            onPress={requestPushPermission}
            disabled={loading}>
            {loading ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <>
                <Text style={styles.buttonIcon}>🔑</Text>
                <Text style={styles.buttonText}>Get Device Token</Text>
              </>
            )}
          </TouchableOpacity>

          {deviceToken && (
            <TouchableOpacity
              style={[styles.button, styles.buttonSuccess]}
              onPress={registerWithBackend}
              disabled={loading}>
              <Text style={styles.buttonIcon}>✅</Text>
              <Text style={styles.buttonText}>Register with Backend</Text>
            </TouchableOpacity>
          )}
        </View>

        {/* Option 2: Manual Token Entry */}
        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <View style={[styles.sectionBadge, {backgroundColor: colors.textSecondary + '20'}]}>
              <Text style={[styles.sectionBadgeText, {color: colors.textSecondary}]}>2</Text>
            </View>
            <Text style={styles.sectionTitle}>Manual Token</Text>
          </View>
          <Text style={styles.sectionDesc}>
            If you have a token from another source
          </Text>
          <View style={styles.inputContainer}>
            <Text style={styles.inputLabel}>Device Token</Text>
            <TextInput
              style={styles.input}
              placeholder="ExponentPushToken[...]"
              placeholderTextColor={colors.textSecondary}
              value={manualToken}
              onChangeText={setManualToken}
              multiline
              numberOfLines={3}
            />
          </View>
          <TouchableOpacity
            style={[styles.button, styles.buttonOutline]}
            onPress={useManualToken}>
            <Text style={styles.buttonIcon}>📝</Text>
            <Text style={styles.buttonTextOutline}>Use This Token</Text>
          </TouchableOpacity>
        </View>

        {/* Important Notes */}
        <View style={styles.warningBox}>
          <View style={styles.infoHeader}>
            <Text style={styles.infoIcon}>⚠️</Text>
            <Text style={styles.warningTitle}>Important Notes</Text>
          </View>
          <View style={styles.notesList}>
            <Text style={styles.warningText}>• Push notifications DON'T work in Expo Go (SDK 53+)</Text>
            <Text style={styles.warningText}>• You must build standalone APK for testing</Text>
            <Text style={styles.warningText}>• Run: eas build --platform android --profile preview</Text>
            <Text style={styles.warningText}>• Then install APK on your phone</Text>
          </View>
        </View>

        {/* Help Section */}
        <View style={styles.helpBox}>
          <View style={styles.infoHeader}>
            <Text style={styles.infoIcon}>💡</Text>
            <Text style={styles.helpTitle}>Need Help?</Text>
          </View>
          <View style={styles.notesList}>
            <Text style={styles.helpText}>1. Check logs: App.tsx for "Expo Push Token"</Text>
            <Text style={styles.helpText}>2. Verify backend: POST /uflow/auth/ciba/register-device</Text>
            <Text style={styles.helpText}>3. Check database: SELECT * FROM device_tokens</Text>
          </View>
        </View>
      </View>
    </ScrollView>
  );
};

const createStyles = (colors: any, isDark: boolean) => StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    padding: 20,
  },
  header: {
    marginBottom: 24,
  },
  title: {
    fontSize: 32,
    fontWeight: '800',
    color: colors.text,
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: colors.textSecondary,
    lineHeight: 22,
  },
  statusCard: {
    backgroundColor: colors.card,
    borderRadius: 16,
    padding: 20,
    marginBottom: 20,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: isDark ? 0.3 : 0.08,
    shadowRadius: 8,
    elevation: 3,
  },
  statusHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  statusIcon: {
    width: 40,
    height: 40,
    borderRadius: 12,
    backgroundColor: colors.primary + '20',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  statusIconText: {
    fontSize: 20,
  },
  statusTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: colors.text,
  },
  statusText: {
    fontSize: 14,
    color: colors.textSecondary,
    marginBottom: 12,
    lineHeight: 20,
  },
  tokenBox: {
    backgroundColor: isDark ? colors.surface : '#f8f9fa',
    padding: 14,
    borderRadius: 12,
    marginTop: 8,
    borderWidth: 1,
    borderColor: colors.border,
  },
  tokenLabel: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.textSecondary,
    marginBottom: 6,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  tokenText: {
    fontSize: 12,
    color: colors.text,
    fontFamily: 'monospace',
  },
  section: {
    backgroundColor: colors.card,
    borderRadius: 16,
    padding: 20,
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: isDark ? 0.3 : 0.08,
    shadowRadius: 8,
    elevation: 3,
  },
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  sectionBadge: {
    width: 28,
    height: 28,
    borderRadius: 8,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  sectionBadgeText: {
    fontSize: 14,
    fontWeight: '700',
  },
  sectionTitleContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: colors.text,
  },
  sectionBadgeLabel: {
    fontSize: 11,
    fontWeight: '600',
    color: colors.success,
    backgroundColor: colors.success + '20',
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 6,
    marginLeft: 10,
    overflow: 'hidden',
  },
  sectionDesc: {
    fontSize: 14,
    color: colors.textSecondary,
    marginBottom: 16,
    lineHeight: 20,
  },
  button: {
    flexDirection: 'row',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 10,
  },
  buttonPrimary: {
    backgroundColor: colors.primary,
  },
  buttonSuccess: {
    backgroundColor: colors.success,
  },
  buttonOutline: {
    backgroundColor: 'transparent',
    borderWidth: 2,
    borderColor: colors.primary,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonIcon: {
    fontSize: 18,
    marginRight: 10,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '700',
  },
  buttonTextOutline: {
    color: colors.primary,
    fontSize: 16,
    fontWeight: '700',
  },
  inputContainer: {
    marginBottom: 12,
  },
  inputLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
  input: {
    backgroundColor: isDark ? colors.surface : '#f8f9fa',
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 12,
    padding: 14,
    fontSize: 14,
    color: colors.text,
    fontFamily: 'monospace',
    textAlignVertical: 'top',
  },
  warningBox: {
    backgroundColor: isDark ? '#3d3520' : '#fff9e6',
    borderRadius: 16,
    padding: 16,
    marginBottom: 16,
    borderLeftWidth: 4,
    borderLeftColor: '#f5a623',
  },
  infoHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  infoIcon: {
    fontSize: 20,
    marginRight: 10,
  },
  warningTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: isDark ? '#f5a623' : '#8b6914',
  },
  notesList: {
    gap: 6,
  },
  warningText: {
    fontSize: 14,
    color: isDark ? '#dbc88d' : '#8b6914',
    lineHeight: 20,
  },
  helpBox: {
    backgroundColor: isDark ? '#1e3a4d' : '#e8f4fd',
    borderRadius: 16,
    padding: 16,
    marginBottom: 24,
    borderLeftWidth: 4,
    borderLeftColor: colors.info,
  },
  helpTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: isDark ? '#64b5f6' : '#1565c0',
  },
  helpText: {
    fontSize: 13,
    color: isDark ? '#90caf9' : '#1976d2',
    lineHeight: 20,
  },
});

export default DeviceRegistrationScreen;
