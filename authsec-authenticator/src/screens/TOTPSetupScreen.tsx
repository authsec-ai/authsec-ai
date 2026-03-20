import React, {useState} from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
  ScrollView,
  Image,
  Clipboard,
  Platform,
} from 'react-native';
import {
  registerTOTPDeviceAdmin,
  registerTOTPDeviceEndUser,
  confirmTOTPDeviceAdmin,
  confirmTOTPDeviceEndUser,
} from '../services/api';
import {getStoredToken} from '../services/storage';
import {useTheme} from '../context/ThemeContext';

const TOTPSetupScreen = ({navigation, route}: any) => {
  const {colors, isDark} = useTheme();
  const styles = createStyles(colors, isDark);
  
  const {userType = 'enduser'} = route.params || {};
  
  const [step, setStep] = useState<'register' | 'confirm'>('register');
  const [deviceName, setDeviceName] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [loading, setLoading] = useState(false);
  
  const [deviceId, setDeviceId] = useState('');
  const [secret, setSecret] = useState('');
  const [qrCodeUrl, setQrCodeUrl] = useState('');
  const [backupCodes, setBackupCodes] = useState<string[]>([]);

  const handleRegister = async () => {
    if (!deviceName.trim()) {
      Alert.alert('Error', 'Please enter a device name');
      return;
    }

    setLoading(true);

    try {
      const token = await getStoredToken();
      if (!token) {
        throw new Error('No authentication token found');
      }

      console.log('Registering TOTP device...');
      const registerFunc = userType === 'admin' 
        ? registerTOTPDeviceAdmin 
        : registerTOTPDeviceEndUser;
      
      const response = await registerFunc(deviceName, token);

      if (response.success) {
        setDeviceId(response.device_id);
        setSecret(response.secret);
        setQrCodeUrl(response.qr_code_url);
        setBackupCodes(response.backup_codes || []);
        setStep('confirm');
        Alert.alert(
          'Device Registered',
          'Scan the QR code with your authenticator app or manually enter the secret.',
        );
      } else {
        throw new Error(response.message || 'Failed to register device');
      }
    } catch (error: any) {
      console.error('Registration error:', error);
      Alert.alert('Registration Failed', error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleConfirm = async () => {
    if (!totpCode.trim() || totpCode.length !== 6) {
      Alert.alert('Error', 'Please enter a valid 6-digit TOTP code');
      return;
    }

    setLoading(true);

    try {
      const token = await getStoredToken();
      if (!token) {
        throw new Error('No authentication token found');
      }

      console.log('Confirming TOTP device...');
      const confirmFunc = userType === 'admin'
        ? confirmTOTPDeviceAdmin
        : confirmTOTPDeviceEndUser;
      
      const response = await confirmFunc(deviceId, totpCode, token);

      if (response.success) {
        Alert.alert(
          'Success',
          'TOTP device confirmed successfully!',
          [
            {
              text: 'OK',
              onPress: () => navigation.goBack(),
            },
          ],
        );
      } else {
        throw new Error(response.message || 'Failed to confirm device');
      }
    } catch (error: any) {
      console.error('Confirmation error:', error);
      Alert.alert('Confirmation Failed', error.message);
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    Clipboard.setString(text);
    Alert.alert('Copied', `${label} copied to clipboard`);
  };

  if (step === 'register') {
    return (
      <ScrollView style={styles.container}>
        <View style={styles.content}>
          <View style={styles.header}>
            <View style={styles.iconContainer}>
              <Text style={styles.headerIcon}>🔒</Text>
            </View>
            <Text style={styles.title}>Setup TOTP</Text>
            <View style={styles.badge}>
              <Text style={styles.badgeText}>
                {userType === 'admin' ? 'Admin' : 'End-User'}
              </Text>
            </View>
          </View>

          <View style={styles.infoBox}>
            <View style={styles.infoHeader}>
              <Text style={styles.infoIcon}>ℹ️</Text>
              <Text style={styles.infoTitle}>What is TOTP?</Text>
            </View>
            <Text style={styles.infoText}>
              Time-based One-Time Password adds an extra layer of security. You'll 
              need an authenticator app like Google Authenticator, Authy, or 
              Microsoft Authenticator.
            </Text>
          </View>

          <View style={styles.card}>
            <View style={styles.inputContainer}>
              <Text style={styles.label}>Device Name *</Text>
              <Text style={styles.hint}>Give this device a memorable name</Text>
              <TextInput
                style={styles.input}
                placeholder="e.g., My iPhone"
                placeholderTextColor={colors.textSecondary}
                value={deviceName}
                onChangeText={setDeviceName}
                autoCapitalize="words"
                editable={!loading}
              />
            </View>

            <TouchableOpacity
              style={[styles.button, loading && styles.buttonDisabled]}
              onPress={handleRegister}
              disabled={loading}>
              {loading ? (
                <ActivityIndicator color="#fff" />
              ) : (
                <>
                  <Text style={styles.buttonIcon}>🚀</Text>
                  <Text style={styles.buttonText}>Register Device</Text>
                </>
              )}
            </TouchableOpacity>
          </View>
        </View>
      </ScrollView>
    );
  }

  // Confirmation step
  return (
    <ScrollView style={styles.container}>
      <View style={styles.content}>
        <View style={styles.header}>
          <Text style={styles.title}>Scan QR Code</Text>
          <Text style={styles.subtitle}>
            Use your authenticator app to scan this code
          </Text>
        </View>

        <View style={styles.qrCard}>
          {qrCodeUrl ? (
            <Image
              source={{uri: qrCodeUrl}}
              style={styles.qrCode}
              resizeMode="contain"
            />
          ) : (
            <ActivityIndicator size="large" color={colors.primary} />
          )}
        </View>

        <View style={styles.secretCard}>
          <View style={styles.secretHeader}>
            <Text style={styles.secretIcon}>🔑</Text>
            <Text style={styles.secretLabel}>Or enter secret manually</Text>
          </View>
          <View style={styles.secretRow}>
            <Text style={styles.secretText} selectable>{secret}</Text>
            <TouchableOpacity
              style={styles.copyBtn}
              onPress={() => copyToClipboard(secret, 'Secret')}>
              <Text style={styles.copyBtnText}>📋</Text>
            </TouchableOpacity>
          </View>
        </View>

        {backupCodes.length > 0 && (
          <View style={styles.backupCard}>
            <View style={styles.backupHeader}>
              <Text style={styles.backupIcon}>⚠️</Text>
              <Text style={styles.backupTitle}>Backup Codes</Text>
            </View>
            <Text style={styles.backupInfo}>
              Save these codes safely. Use them if you lose access to your 
              authenticator app.
            </Text>
            <View style={styles.codesContainer}>
              {backupCodes.map((code, index) => (
                <Text key={index} style={styles.backupCode}>
                  {code}
                </Text>
              ))}
            </View>
            <TouchableOpacity
              style={styles.copyAllBtn}
              onPress={() => copyToClipboard(backupCodes.join('\n'), 'Backup Codes')}>
              <Text style={styles.copyAllText}>📋 Copy All Codes</Text>
            </TouchableOpacity>
          </View>
        )}

        <View style={styles.card}>
          <View style={styles.inputContainer}>
            <Text style={styles.label}>Verification Code *</Text>
            <Text style={styles.hint}>Enter the 6-digit code from your app</Text>
            <TextInput
              style={[styles.input, styles.codeInput]}
              placeholder="000000"
              placeholderTextColor={colors.textSecondary}
              value={totpCode}
              onChangeText={setTotpCode}
              keyboardType="number-pad"
              maxLength={6}
              editable={!loading}
            />
          </View>

          <TouchableOpacity
            style={[styles.button, styles.buttonSuccess, loading && styles.buttonDisabled]}
            onPress={handleConfirm}
            disabled={loading}>
            {loading ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <>
                <Text style={styles.buttonIcon}>✅</Text>
                <Text style={styles.buttonText}>Confirm & Activate</Text>
              </>
            )}
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.backButton}
            onPress={() => setStep('register')}
            disabled={loading}>
            <Text style={styles.backButtonText}>← Back to Registration</Text>
          </TouchableOpacity>
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
    alignItems: 'center',
    marginBottom: 24,
  },
  iconContainer: {
    width: 80,
    height: 80,
    borderRadius: 24,
    backgroundColor: colors.primary + '20',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  headerIcon: {
    fontSize: 40,
  },
  title: {
    fontSize: 28,
    fontWeight: '800',
    color: colors.text,
    marginBottom: 8,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 16,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 22,
  },
  badge: {
    backgroundColor: colors.primary + '20',
    paddingHorizontal: 12,
    paddingVertical: 4,
    borderRadius: 12,
  },
  badgeText: {
    fontSize: 13,
    fontWeight: '600',
    color: colors.primary,
  },
  infoBox: {
    backgroundColor: isDark ? '#1e3a4d' : '#e8f4fd',
    borderRadius: 16,
    padding: 16,
    marginBottom: 20,
    borderLeftWidth: 4,
    borderLeftColor: colors.info,
  },
  infoHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  infoIcon: {
    fontSize: 18,
    marginRight: 8,
  },
  infoTitle: {
    fontSize: 15,
    fontWeight: '700',
    color: isDark ? '#64b5f6' : '#1565c0',
  },
  infoText: {
    fontSize: 14,
    color: isDark ? '#90caf9' : '#1976d2',
    lineHeight: 20,
  },
  card: {
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
  inputContainer: {
    marginBottom: 16,
  },
  label: {
    fontSize: 15,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 4,
  },
  hint: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 10,
  },
  input: {
    backgroundColor: isDark ? colors.surface : '#f8f9fa',
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 12,
    padding: 16,
    fontSize: 16,
    color: colors.text,
  },
  codeInput: {
    fontSize: 24,
    fontWeight: '700',
    textAlign: 'center',
    letterSpacing: 8,
  },
  button: {
    backgroundColor: colors.primary,
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    flexDirection: 'row',
    justifyContent: 'center',
    shadowColor: colors.primary,
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 5,
  },
  buttonSuccess: {
    backgroundColor: colors.success,
    shadowColor: colors.success,
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
  qrCard: {
    backgroundColor: colors.card,
    borderRadius: 20,
    padding: 24,
    alignItems: 'center',
    marginBottom: 20,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: isDark ? 0.3 : 0.1,
    shadowRadius: 12,
    elevation: 5,
  },
  qrCode: {
    width: 220,
    height: 220,
    borderRadius: 12,
  },
  secretCard: {
    backgroundColor: colors.card,
    borderRadius: 16,
    padding: 16,
    marginBottom: 16,
  },
  secretHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  secretIcon: {
    fontSize: 18,
    marginRight: 8,
  },
  secretLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
  },
  secretRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: isDark ? colors.surface : '#f8f9fa',
    borderRadius: 10,
    padding: 12,
  },
  secretText: {
    fontSize: 14,
    fontWeight: '700',
    color: colors.text,
    flex: 1,
    fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
  },
  copyBtn: {
    width: 36,
    height: 36,
    borderRadius: 10,
    backgroundColor: colors.primary + '20',
    justifyContent: 'center',
    alignItems: 'center',
    marginLeft: 10,
  },
  copyBtnText: {
    fontSize: 18,
  },
  backupCard: {
    backgroundColor: isDark ? '#3d3520' : '#fff9e6',
    borderRadius: 16,
    padding: 16,
    marginBottom: 16,
    borderWidth: 1,
    borderColor: isDark ? '#5a4a1e' : '#ffc107',
  },
  backupHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  backupIcon: {
    fontSize: 18,
    marginRight: 8,
  },
  backupTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: isDark ? '#f5a623' : '#8b6914',
  },
  backupInfo: {
    fontSize: 13,
    color: isDark ? '#dbc88d' : '#8b6914',
    marginBottom: 12,
    lineHeight: 18,
  },
  codesContainer: {
    backgroundColor: isDark ? colors.surface : '#fff',
    borderRadius: 10,
    padding: 10,
    marginBottom: 12,
  },
  backupCode: {
    fontSize: 14,
    fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
    color: colors.text,
    padding: 6,
    marginBottom: 2,
  },
  copyAllBtn: {
    alignItems: 'center',
    padding: 10,
    backgroundColor: isDark ? colors.surface : 'rgba(255,255,255,0.5)',
    borderRadius: 10,
  },
  copyAllText: {
    color: colors.primary,
    fontSize: 14,
    fontWeight: '600',
  },
  backButton: {
    marginTop: 16,
    alignItems: 'center',
    padding: 14,
  },
  backButtonText: {
    color: colors.primary,
    fontSize: 15,
    fontWeight: '600',
  },
});

export default TOTPSetupScreen;
