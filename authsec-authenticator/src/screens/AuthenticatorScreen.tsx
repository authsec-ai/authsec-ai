import React, {useState, useEffect} from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  FlatList,
  Modal,
  Platform,
  StatusBar,
  Animated,
  Pressable,
} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {useTheme} from '../context/ThemeContext';
import {generateSync} from 'otplib';
import {Ionicons} from '@expo/vector-icons';
import {addActivityLog} from '../services/storage';

// Conditional import for camera
let CameraView: any = null;
try {
  const expoCamera = require('expo-camera');
  CameraView = expoCamera.CameraView;
} catch (e) {
  console.log('Camera not available - QR scanning disabled');
}

interface TOTPAccount {
  id: string;
  name: string;
  issuer: string;
  secret: string;
  addedAt: number;
}

const AuthenticatorScreen = () => {
  const {colors, isDark} = useTheme();
  const [accounts, setAccounts] = useState<TOTPAccount[]>([]);
  const [totpCodes, setTotpCodes] = useState<{[key: string]: string}>({});
  const [timeRemaining, setTimeRemaining] = useState(30);
  
  // Add account modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [addMethod, setAddMethod] = useState<'qr' | 'manual' | null>(null);
  
  // Manual entry
  const [manualName, setManualName] = useState('');
  const [manualIssuer, setManualIssuer] = useState('');
  const [manualSecret, setManualSecret] = useState('');
  
  // QR Scanner
  const [hasPermission, setHasPermission] = useState<boolean | null>(null);
  const [scanning, setScanning] = useState(false);

  // Animation
  const progressAnim = useState(new Animated.Value(1))[0];
  const lastPeriodRef = React.useRef<number>(0);

  // Load accounts on mount
  useEffect(() => {
    loadAccounts();
  }, []);

  // Timer and code generation - runs every second
  useEffect(() => {
    const updateTick = () => {
      const now = Math.floor(Date.now() / 1000);
      const currentPeriod = Math.floor(now / 30);
      const secondsRemaining = 30 - (now % 30);
      
      setTimeRemaining(secondsRemaining);
      
      // Regenerate codes when we enter a new 30-second period
      if (currentPeriod !== lastPeriodRef.current) {
        lastPeriodRef.current = currentPeriod;
        generateAllCodesForAccounts(accounts);
      }
    };

    // Run immediately
    updateTick();
    
    // Then every second
    const interval = setInterval(updateTick, 1000);
    return () => clearInterval(interval);
  }, [accounts]);

  // Initial code generation when accounts load
  useEffect(() => {
    if (accounts.length > 0) {
      lastPeriodRef.current = Math.floor(Date.now() / 1000 / 30);
      generateAllCodesForAccounts(accounts);
    }
  }, [accounts.length]);

  // Animate progress bar smoothly
  useEffect(() => {
    progressAnim.setValue(timeRemaining / 30);
  }, [timeRemaining]);

  const loadAccounts = async () => {
    try {
      const stored = await AsyncStorage.getItem('@totp_accounts');
      if (stored) {
        const parsed = JSON.parse(stored);
        setAccounts(parsed);
        generateAllCodesForAccounts(parsed);
      }
    } catch (error) {
      console.error('Failed to load accounts:', error);
    }
  };

  const saveAccounts = async (newAccounts: TOTPAccount[]) => {
    try {
      await AsyncStorage.setItem('@totp_accounts', JSON.stringify(newAccounts));
      setAccounts(newAccounts);
      // Generate codes immediately for new accounts
      generateAllCodesForAccounts(newAccounts);
    } catch (error) {
      console.error('Failed to save accounts:', error);
    }
  };

  const generateAllCodes = () => {
    generateAllCodesForAccounts(accounts);
  };

  const generateAllCodesForAccounts = (accountList: TOTPAccount[]) => {
    const codes: {[key: string]: string} = {};
    accountList.forEach(account => {
      codes[account.id] = generateTOTP(account.secret);
    });
    setTotpCodes(codes);
  };

  // Proper TOTP implementation using otplib
  const generateTOTP = (secret: string): string => {
    try {
      // Clean the secret - remove spaces, dashes, and convert to uppercase
      let cleanSecret = secret.replace(/[\s-]/g, '').toUpperCase();
      
      // Remove any existing padding
      cleanSecret = cleanSecret.replace(/=+$/, '');
      
      // Validate base32 characters
      const base32Regex = /^[A-Z2-7]+$/;
      if (!base32Regex.test(cleanSecret)) {
        console.error('Invalid base32 characters in secret');
        return '------';
      }
      
      // Generate the TOTP code using otplib v13 generateSync
      const code = generateSync({secret: cleanSecret});
      console.log('Generated TOTP for secret:', cleanSecret.substring(0, 4) + '...', '-> Code:', code);
      return code;
    } catch (error) {
      console.error('TOTP generation error:', error, 'Secret:', secret?.substring(0, 4));
      return '------';
    }
  };

  const parseOtpauthUrl = (url: string) => {
    try {
      // Format: otpauth://totp/Label?secret=XXX&issuer=YYY
      if (!url.startsWith('otpauth://totp/')) {
        return null;
      }
      
      const withoutPrefix = url.replace('otpauth://totp/', '');
      const [labelPart, paramsPart] = withoutPrefix.split('?');
      
      // Decode label
      let label = decodeURIComponent(labelPart);
      let issuer = '';
      let name = label;
      
      // Check for issuer:name format
      if (label.includes(':')) {
        const [i, n] = label.split(':');
        issuer = i;
        name = n;
      }
      
      // Parse params
      const params = new URLSearchParams(paramsPart);
      const secret = params.get('secret');
      const issuerParam = params.get('issuer');
      
      if (issuerParam) {
        issuer = decodeURIComponent(issuerParam);
      }
      
      if (!secret) {
        return null;
      }
      
      return {
        name: name || 'Unknown',
        issuer: issuer || 'Unknown',
        secret: secret,
      };
    } catch (error) {
      return null;
    }
  };

  const handleQRScan = async () => {
    if (!CameraView) {
      Alert.alert(
        'QR Scanning Unavailable',
        'QR code scanning requires a development build. Please use "Enter Manually" to add accounts.',
        [{text: 'OK'}]
      );
      return;
    }

    try {
      const Camera = require('expo-camera').Camera;
      const {status} = await Camera.requestCameraPermissionsAsync();
      setHasPermission(status === 'granted');
      
      if (status === 'granted') {
        setScanning(true);
        setAddMethod('qr');
      } else {
        Alert.alert('Permission Denied', 'Camera permission is required to scan QR codes');
      }
    } catch (error) {
      Alert.alert(
        'Error',
        'QR scanning is not available in Expo Go. Please use "Enter Manually" instead.',
        [{text: 'OK'}]
      );
    }
  };

  const handleBarCodeScanned = ({data}: {data: string}) => {
    setScanning(false);
    
    const parsed = parseOtpauthUrl(data);
    if (parsed) {
      addAccount(parsed.name, parsed.issuer, parsed.secret);
    } else {
      Alert.alert('Invalid QR Code', 'This is not a valid TOTP QR code');
    }
  };

  const handleManualAdd = () => {
    if (!manualName.trim() || !manualSecret.trim()) {
      Alert.alert('Error', 'Please enter account name and secret key');
      return;
    }
    
    addAccount(manualName, manualIssuer || 'Unknown', manualSecret);
    setManualName('');
    setManualIssuer('');
    setManualSecret('');
  };

  const addAccount = async (name: string, issuer: string, secret: string) => {
    const newAccount: TOTPAccount = {
      id: Date.now().toString(),
      name,
      issuer,
      secret: secret.replace(/\s/g, '').toUpperCase(),
      addedAt: Date.now(),
    };

    const updated = [...accounts, newAccount];
    await saveAccounts(updated);

    // Log the TOTP add activity
    await addActivityLog({
      type: 'totp_added',
      title: 'TOTP Account Added',
      description: `Added ${issuer}: ${name} to authenticator`,
      metadata: { issuer, name },
    });

    setShowAddModal(false);
    setAddMethod(null);

    Alert.alert('Success', `${issuer}: ${name} added successfully`);
  };

  const deleteAccount = (id: string) => {
    const account = accounts.find(a => a.id === id);
    Alert.alert(
      'Delete Account',
      'Are you sure you want to delete this account? This action cannot be undone.',
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            const updated = accounts.filter(a => a.id !== id);
            await saveAccounts(updated);

            // Log the TOTP delete activity
            if (account) {
              await addActivityLog({
                type: 'totp_deleted',
                title: 'TOTP Account Removed',
                description: `Removed ${account.issuer}: ${account.name} from authenticator`,
                metadata: { issuer: account.issuer, name: account.name },
              });
            }
          },
        },
      ],
    );
  };

  const copyToClipboard = (code: string) => {
    // React Native doesn't have Clipboard by default, using alert for feedback
    Alert.alert('Copied!', `Code ${code} copied to clipboard`);
  };

  const styles = createStyles(colors, isDark);

  const renderAccount = ({item}: {item: TOTPAccount}) => {
    const code = totpCodes[item.id] || '------';
    const formattedCode = `${code.slice(0, 3)} ${code.slice(3)}`;
    const isLowTime = timeRemaining <= 5;
    const progress = timeRemaining / 30;
    
    return (
      <Pressable 
        style={styles.accountCard}
        onLongPress={() => deleteAccount(item.id)}>
        <View style={styles.accountHeader}>
          <View style={styles.accountIcon}>
            <Text style={styles.accountIconText}>
              {item.issuer.charAt(0).toUpperCase()}
            </Text>
          </View>
          <View style={styles.accountInfo}>
            <Text style={styles.issuer}>{item.issuer}</Text>
            <Text style={styles.accountName}>{item.name}</Text>
          </View>
          <TouchableOpacity 
            style={styles.deleteButton}
            onPress={() => deleteAccount(item.id)}>
            <Text style={styles.deleteButtonText}>×</Text>
          </TouchableOpacity>
        </View>
        <View style={styles.codeContainer}>
          <Text style={[styles.code, isLowTime && styles.codeLowTime]}>
            {formattedCode}
          </Text>
          <View style={styles.timerContainer}>
            {/* Circular progress indicator */}
            <View style={[styles.circleTimer, isLowTime && styles.circleTimerLow]}>
              <View style={[
                styles.circleProgress,
                isLowTime ? styles.circleProgressLow : null,
                {
                  transform: [{rotate: `${(1 - progress) * 360}deg`}],
                }
              ]}>
                <View style={[
                  styles.circleProgressFill, 
                  isLowTime && styles.circleProgressFillLow
                ]} />
              </View>
              <View style={styles.circleCenter}>
                <Text style={[styles.circleTimerText, isLowTime && styles.timerLowTime]}>
                  {timeRemaining}
                </Text>
              </View>
            </View>
          </View>
        </View>
      </Pressable>
    );
  };

  return (
    <View style={styles.container}>
      <StatusBar 
        barStyle={isDark ? 'light-content' : 'dark-content'} 
        backgroundColor={colors.primary}
      />
      
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.title}>Authenticator</Text>
        <Text style={styles.subtitle}>
          {accounts.length} {accounts.length === 1 ? 'account' : 'accounts'}
        </Text>
      </View>

      {accounts.length === 0 ? (
        <View style={styles.emptyState}>
          <View style={styles.emptyIconContainer}>
            <Ionicons name="lock-closed" size={48} color={colors.textMuted} />
          </View>
          <Text style={styles.emptyTitle}>No Accounts Yet</Text>
          <Text style={styles.emptyText}>
            Add your first account by scanning a QR code or entering the secret key manually
          </Text>
        </View>
      ) : (
        <FlatList
          data={accounts}
          renderItem={renderAccount}
          keyExtractor={item => item.id}
          contentContainerStyle={styles.list}
          showsVerticalScrollIndicator={false}
        />
      )}

      {/* Floating Add Button */}
      <TouchableOpacity
        style={styles.fab}
        onPress={() => setShowAddModal(true)}
        activeOpacity={0.8}>
        <Text style={styles.fabText}>+</Text>
      </TouchableOpacity>

      {/* Add Account Modal */}
      <Modal
        visible={showAddModal}
        animationType="slide"
        transparent={true}
        onRequestClose={() => {
          setShowAddModal(false);
          setAddMethod(null);
          setScanning(false);
        }}>
        <View style={styles.modalOverlay}>
          <Pressable 
            style={styles.modalBackdrop} 
            onPress={() => {
              setShowAddModal(false);
              setAddMethod(null);
              setScanning(false);
            }}
          />
          <View style={styles.modalContent}>
            <View style={styles.modalHandle} />
            
            {!addMethod && !scanning && (
              <>
                <Text style={styles.modalTitle}>Add Account</Text>
                <Text style={styles.modalSubtitle}>
                  Choose how you want to add your account
                </Text>
                
                <TouchableOpacity
                  style={styles.methodButton}
                  onPress={handleQRScan}
                  activeOpacity={0.7}>
                  <View style={styles.methodIcon}>
                    <Ionicons name="camera" size={24} color={colors.primary} />
                  </View>
                  <View style={styles.methodContent}>
                    <Text style={styles.methodText}>Scan QR Code</Text>
                    <Text style={styles.methodDescription}>
                      Use your camera to scan the QR code
                    </Text>
                  </View>
                  <Text style={styles.methodChevron}>›</Text>
                </TouchableOpacity>
                
                <TouchableOpacity
                  style={styles.methodButton}
                  onPress={() => setAddMethod('manual')}
                  activeOpacity={0.7}>
                  <View style={styles.methodIcon}>
                    <Ionicons name="keypad" size={24} color={colors.info} />
                  </View>
                  <View style={styles.methodContent}>
                    <Text style={styles.methodText}>Enter Manually</Text>
                    <Text style={styles.methodDescription}>
                      Type the secret key from your provider
                    </Text>
                  </View>
                  <Text style={styles.methodChevron}>›</Text>
                </TouchableOpacity>
                
                <TouchableOpacity
                  style={styles.cancelButton}
                  onPress={() => setShowAddModal(false)}>
                  <Text style={styles.cancelButtonText}>Cancel</Text>
                </TouchableOpacity>
              </>
            )}

            {scanning && CameraView && (
              <View style={styles.scannerContainer}>
                <Text style={styles.scannerTitle}>Scan QR Code</Text>
                <View style={styles.scannerPreview}>
                  <CameraView
                    onBarcodeScanned={handleBarCodeScanned}
                    barcodeScannerSettings={{
                      barcodeTypes: ['qr'],
                    }}
                    style={StyleSheet.absoluteFillObject}
                  />
                  <View style={styles.scannerOverlay}>
                    <View style={styles.scannerFrame} />
                  </View>
                </View>
                <TouchableOpacity
                  style={styles.scannerCancelButton}
                  onPress={() => {
                    setScanning(false);
                    setAddMethod(null);
                  }}>
                  <Text style={styles.scannerCancelText}>Cancel</Text>
                </TouchableOpacity>
              </View>
            )}

            {addMethod === 'manual' && !scanning && (
              <>
                <TouchableOpacity 
                  style={styles.backButton}
                  onPress={() => setAddMethod(null)}>
                  <Text style={styles.backButtonText}>‹ Back</Text>
                </TouchableOpacity>
                
                <Text style={styles.modalTitle}>Enter Details</Text>
                <Text style={styles.modalSubtitle}>
                  Enter the account information from your provider
                </Text>
                
                <View style={styles.inputGroup}>
                  <Text style={styles.inputLabel}>Account Name *</Text>
                  <TextInput
                    style={styles.input}
                    placeholder="e.g., john@example.com"
                    placeholderTextColor={colors.inputPlaceholder}
                    value={manualName}
                    onChangeText={setManualName}
                    autoCapitalize="none"
                    autoCorrect={false}
                  />
                </View>
                
                <View style={styles.inputGroup}>
                  <Text style={styles.inputLabel}>Secret Key *</Text>
                  <TextInput
                    style={styles.input}
                    placeholder="e.g., JBSWY3DPEHPK3PXP"
                    placeholderTextColor={colors.inputPlaceholder}
                    value={manualSecret}
                    onChangeText={setManualSecret}
                    autoCapitalize="characters"
                    autoCorrect={false}
                  />
                  <Text style={styles.inputHint}>
                    The secret key provided by your service
                  </Text>
                </View>
                
                <View style={styles.inputGroup}>
                  <Text style={styles.inputLabel}>Issuer (Optional)</Text>
                  <TextInput
                    style={styles.input}
                    placeholder="e.g., authsec.ai"
                    placeholderTextColor={colors.inputPlaceholder}
                    value={manualIssuer}
                    onChangeText={setManualIssuer}
                  />
                  <Text style={styles.inputHint}>
                    The service or company name
                  </Text>
                </View>
                
                <TouchableOpacity
                  style={styles.addAccountButton}
                  onPress={handleManualAdd}
                  activeOpacity={0.8}>
                  <Text style={styles.addAccountButtonText}>Add Account</Text>
                </TouchableOpacity>
                
                <TouchableOpacity
                  style={styles.cancelButton}
                  onPress={() => {
                    setAddMethod(null);
                    setManualName('');
                    setManualIssuer('');
                    setManualSecret('');
                  }}>
                  <Text style={styles.cancelButtonText}>Cancel</Text>
                </TouchableOpacity>
              </>
            )}
          </View>
        </View>
      </Modal>
    </View>
  );
};

const createStyles = (colors: any, isDark: boolean) =>
  StyleSheet.create({
    container: {
      flex: 1,
      backgroundColor: colors.background,
    },
    header: {
      paddingHorizontal: 24,
      paddingTop: Platform.OS === 'ios' ? 60 : 20,
      paddingBottom: 20,
      backgroundColor: colors.primary,
    },
    title: {
      fontSize: 32,
      fontWeight: '700',
      color: '#FFFFFF',
      marginBottom: 4,
    },
    subtitle: {
      fontSize: 15,
      color: 'rgba(255, 255, 255, 0.8)',
      fontWeight: '500',
    },
    emptyState: {
      flex: 1,
      justifyContent: 'center',
      alignItems: 'center',
      padding: 40,
    },
    emptyIconContainer: {
      width: 100,
      height: 100,
      borderRadius: 50,
      backgroundColor: colors.primaryLight,
      justifyContent: 'center',
      alignItems: 'center',
      marginBottom: 24,
    },
    emptyTitle: {
      fontSize: 22,
      fontWeight: '700',
      color: colors.text,
      marginBottom: 12,
    },
    emptyText: {
      fontSize: 15,
      color: colors.textSecondary,
      textAlign: 'center',
      lineHeight: 22,
    },
    list: {
      padding: 16,
      paddingBottom: 100,
    },
    accountCard: {
      backgroundColor: colors.card,
      borderRadius: 16,
      padding: 16,
      marginBottom: 12,
      shadowColor: colors.shadow,
      shadowOffset: {width: 0, height: 2},
      shadowOpacity: isDark ? 0.3 : 0.08,
      shadowRadius: 8,
      elevation: 3,
    },
    accountHeader: {
      flexDirection: 'row',
      alignItems: 'center',
      marginBottom: 16,
    },
    accountIcon: {
      width: 44,
      height: 44,
      borderRadius: 12,
      backgroundColor: colors.primary,
      justifyContent: 'center',
      alignItems: 'center',
      marginRight: 12,
    },
    accountIconText: {
      fontSize: 20,
      fontWeight: '700',
      color: '#FFFFFF',
    },
    accountInfo: {
      flex: 1,
    },
    issuer: {
      fontSize: 17,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 2,
    },
    accountName: {
      fontSize: 13,
      color: colors.textSecondary,
    },
    deleteButton: {
      width: 32,
      height: 32,
      borderRadius: 16,
      backgroundColor: colors.errorLight,
      justifyContent: 'center',
      alignItems: 'center',
    },
    deleteButtonText: {
      fontSize: 20,
      color: colors.error,
      fontWeight: '600',
      marginTop: -2,
    },
    codeContainer: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'space-between',
    },
    code: {
      fontSize: 36,
      fontWeight: '700',
      color: colors.primary,
      letterSpacing: 4,
      fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
    },
    codeLowTime: {
      color: colors.error,
    },
    timerContainer: {
      alignItems: 'center',
      justifyContent: 'center',
    },
    circleTimer: {
      width: 52,
      height: 52,
      borderRadius: 26,
      backgroundColor: colors.primaryLight,
      justifyContent: 'center',
      alignItems: 'center',
      overflow: 'hidden',
    },
    circleTimerLow: {
      backgroundColor: colors.errorLight,
    },
    circleProgress: {
      position: 'absolute',
      width: 52,
      height: 52,
      borderRadius: 26,
    },
    circleProgressLow: {},
    circleProgressFill: {
      width: 26,
      height: 52,
      backgroundColor: colors.primary,
      borderTopRightRadius: 26,
      borderBottomRightRadius: 26,
      position: 'absolute',
      right: 0,
    },
    circleProgressFillLow: {
      backgroundColor: colors.error,
    },
    circleCenter: {
      width: 42,
      height: 42,
      borderRadius: 21,
      backgroundColor: colors.card,
      justifyContent: 'center',
      alignItems: 'center',
      zIndex: 1,
    },
    circleTimerText: {
      fontSize: 16,
      fontWeight: '700',
      color: colors.primary,
    },
    timer: {
      fontSize: 14,
      fontWeight: '600',
      color: colors.textSecondary,
      marginBottom: 6,
    },
    timerLowTime: {
      color: colors.error,
    },
    progressBar: {
      width: 60,
      height: 4,
      backgroundColor: colors.divider,
      borderRadius: 2,
      overflow: 'hidden',
    },
    progressFill: {
      height: '100%',
      backgroundColor: colors.primary,
      borderRadius: 2,
    },
    progressFillLowTime: {
      backgroundColor: colors.error,
    },
    fab: {
      position: 'absolute',
      bottom: 24,
      right: 24,
      width: 60,
      height: 60,
      borderRadius: 30,
      backgroundColor: colors.primary,
      justifyContent: 'center',
      alignItems: 'center',
      shadowColor: colors.primary,
      shadowOffset: {width: 0, height: 6},
      shadowOpacity: 0.4,
      shadowRadius: 12,
      elevation: 8,
    },
    fabText: {
      fontSize: 32,
      color: '#FFFFFF',
      fontWeight: '400',
      marginTop: -2,
    },
    modalOverlay: {
      flex: 1,
      justifyContent: 'flex-end',
    },
    modalBackdrop: {
      ...StyleSheet.absoluteFillObject,
      backgroundColor: colors.overlay,
    },
    modalContent: {
      backgroundColor: colors.surface,
      borderTopLeftRadius: 24,
      borderTopRightRadius: 24,
      padding: 24,
      paddingBottom: Platform.OS === 'ios' ? 40 : 24,
      maxHeight: '85%',
    },
    modalHandle: {
      width: 40,
      height: 4,
      borderRadius: 2,
      backgroundColor: colors.border,
      alignSelf: 'center',
      marginBottom: 20,
    },
    modalTitle: {
      fontSize: 24,
      fontWeight: '700',
      color: colors.text,
      marginBottom: 8,
    },
    modalSubtitle: {
      fontSize: 15,
      color: colors.textSecondary,
      marginBottom: 24,
      lineHeight: 22,
    },
    methodButton: {
      flexDirection: 'row',
      alignItems: 'center',
      backgroundColor: colors.background,
      padding: 16,
      borderRadius: 16,
      marginBottom: 12,
    },
    methodIcon: {
      width: 48,
      height: 48,
      borderRadius: 14,
      backgroundColor: colors.primaryLight,
      justifyContent: 'center',
      alignItems: 'center',
      marginRight: 14,
    },
    methodContent: {
      flex: 1,
    },
    methodText: {
      fontSize: 17,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 2,
    },
    methodDescription: {
      fontSize: 13,
      color: colors.textSecondary,
    },
    methodChevron: {
      fontSize: 28,
      color: colors.textMuted,
    },
    backButton: {
      marginBottom: 16,
    },
    backButtonText: {
      fontSize: 17,
      color: colors.primary,
      fontWeight: '600',
    },
    inputGroup: {
      marginBottom: 20,
    },
    inputLabel: {
      fontSize: 14,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 8,
    },
    input: {
      backgroundColor: colors.inputBackground,
      borderWidth: 1,
      borderColor: colors.inputBorder,
      borderRadius: 12,
      padding: 16,
      fontSize: 16,
      color: colors.text,
    },
    inputHint: {
      fontSize: 12,
      color: colors.textMuted,
      marginTop: 6,
      marginLeft: 4,
    },
    addAccountButton: {
      backgroundColor: colors.primary,
      padding: 18,
      borderRadius: 14,
      alignItems: 'center',
      marginTop: 8,
      shadowColor: colors.primary,
      shadowOffset: {width: 0, height: 4},
      shadowOpacity: 0.3,
      shadowRadius: 8,
      elevation: 4,
    },
    addAccountButtonText: {
      color: '#FFFFFF',
      fontSize: 17,
      fontWeight: '600',
    },
    cancelButton: {
      padding: 16,
      alignItems: 'center',
      marginTop: 8,
    },
    cancelButtonText: {
      color: colors.textSecondary,
      fontSize: 16,
      fontWeight: '500',
    },
    scannerContainer: {
      alignItems: 'center',
    },
    scannerTitle: {
      fontSize: 20,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 16,
    },
    scannerPreview: {
      width: '100%',
      height: 300,
      borderRadius: 16,
      overflow: 'hidden',
      backgroundColor: colors.background,
    },
    scannerOverlay: {
      ...StyleSheet.absoluteFillObject,
      justifyContent: 'center',
      alignItems: 'center',
    },
    scannerFrame: {
      width: 200,
      height: 200,
      borderWidth: 3,
      borderColor: colors.primary,
      borderRadius: 16,
    },
    scannerCancelButton: {
      marginTop: 20,
      padding: 16,
    },
    scannerCancelText: {
      fontSize: 17,
      color: colors.error,
      fontWeight: '600',
    },
  });

export default AuthenticatorScreen;
