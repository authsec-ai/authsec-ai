import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  StatusBar,
  RefreshControl,
  FlatList,
  Modal,
  TextInput,
  Alert,
  Animated,
  Dimensions,
  Image,
  BackHandler,
} from 'react-native';
import { useFocusEffect } from '@react-navigation/native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { generateSync } from 'otplib';
import { getStoredEmail, getDeviceToken, getUnreadNotificationCount, addActivityLog } from '../services/storage';
import { useTheme } from '../context/ThemeContext';
import { Ionicons } from '@expo/vector-icons';

const { width } = Dimensions.get('window');

// Conditional import for camera
let CameraView: any = null;
try {
  const expoCamera = require('expo-camera');
  CameraView = expoCamera.CameraView;
} catch (e) {
  console.log('Camera not available');
}

interface TOTPAccount {
  id: string;
  name: string;
  issuer: string;
  secret: string;
  addedAt: number;
}

const HomeScreen = ({ navigation }: any) => {
  const { colors, isDark } = useTheme();
  const [email, setEmail] = useState('');
  const [deviceRegistered, setDeviceRegistered] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);

  // TOTP state
  const [accounts, setAccounts] = useState<TOTPAccount[]>([]);
  const [totpCodes, setTotpCodes] = useState<{ [key: string]: string }>({});
  const [timeRemaining, setTimeRemaining] = useState(30);
  const lastPeriodRef = useRef<number>(0);

  // Add modal state
  const [showAddModal, setShowAddModal] = useState(false);
  const [addMethod, setAddMethod] = useState<'qr' | 'manual' | null>(null);
  const [scanning, setScanning] = useState(false);
  const [manualName, setManualName] = useState('');
  const [manualIssuer, setManualIssuer] = useState('');
  const [manualSecret, setManualSecret] = useState('');

  // Animation
  const fabScale = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    loadEmail();
    loadAccounts();
  }, []);

  // Timer for TOTP codes
  useEffect(() => {
    const updateTick = () => {
      const now = Math.floor(Date.now() / 1000);
      const currentPeriod = Math.floor(now / 30);
      const secondsRemaining = 30 - (now % 30);

      setTimeRemaining(secondsRemaining);

      if (currentPeriod !== lastPeriodRef.current) {
        lastPeriodRef.current = currentPeriod;
        generateAllCodesForAccounts(accounts);
      }
    };

    updateTick();
    const interval = setInterval(updateTick, 1000);
    return () => clearInterval(interval);
  }, [accounts]);

  useEffect(() => {
    if (accounts.length > 0) {
      lastPeriodRef.current = Math.floor(Date.now() / 1000 / 30);
      generateAllCodesForAccounts(accounts);
    }
  }, [accounts.length]);

  useFocusEffect(
    useCallback(() => {
      checkDeviceStatus();
      loadAccounts();
      loadUnreadCount();

      // Handle back button on Android
      const backHandler = BackHandler.addEventListener('hardwareBackPress', () => {
        // Show confirmation dialog
        Alert.alert(
          'Exit App',
          'Do you want to exit the app?',
          [
            {
              text: 'No',
              style: 'cancel',
            },
            {
              text: 'Yes',
              onPress: () => {
                // Close the app completely
                BackHandler.exitApp();
              },
            },
          ],
          { cancelable: true }
        );
        return true; // Prevent default back behavior
      });

      return () => backHandler.remove();
    }, [navigation]),
  );

  const loadEmail = async () => {
    const storedEmail = await getStoredEmail();
    if (storedEmail) setEmail(storedEmail);
  };

  const checkDeviceStatus = async () => {
    const token = await getDeviceToken();
    setDeviceRegistered(!!token);
  };

  const loadUnreadCount = async () => {
    const count = await getUnreadNotificationCount();
    setUnreadCount(count);
  };

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
      generateAllCodesForAccounts(newAccounts);
    } catch (error) {
      console.error('Failed to save accounts:', error);
    }
  };

  const generateTOTP = (secret: string): string => {
    try {
      let cleanSecret = secret.replace(/[\s-]/g, '').toUpperCase().replace(/=+$/, '');
      const base32Regex = /^[A-Z2-7]+$/;
      if (!base32Regex.test(cleanSecret)) return '------';
      return generateSync({ secret: cleanSecret });
    } catch (error) {
      console.error('TOTP error:', error);
      return '------';
    }
  };

  const generateAllCodesForAccounts = (accountList: TOTPAccount[]) => {
    const codes: { [key: string]: string } = {};
    accountList.forEach(account => {
      codes[account.id] = generateTOTP(account.secret);
    });
    setTotpCodes(codes);
  };

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await loadEmail();
    await checkDeviceStatus();
    await loadAccounts();
    setRefreshing(false);
  }, []);

  const handleFabPress = () => {
    Animated.sequence([
      Animated.timing(fabScale, { toValue: 0.9, duration: 100, useNativeDriver: true }),
      Animated.timing(fabScale, { toValue: 1, duration: 100, useNativeDriver: true }),
    ]).start();
    setShowAddModal(true);
  };

  const parseOtpauthUrl = (url: string) => {
    try {
      if (!url.startsWith('otpauth://totp/')) return null;
      const withoutPrefix = url.replace('otpauth://totp/', '');
      const [labelPart, paramsPart] = withoutPrefix.split('?');
      let label = decodeURIComponent(labelPart);
      let issuer = '', name = label;
      if (label.includes(':')) {
        const parts = label.split(':');
        issuer = parts[0];
        name = parts.slice(1).join(':');
      }
      const params = new URLSearchParams(paramsPart);
      const secret = params.get('secret');
      const issuerParam = params.get('issuer');
      if (issuerParam) issuer = decodeURIComponent(issuerParam);
      if (!secret) return null;
      return { name: name || 'Unknown', issuer: issuer || 'Unknown', secret };
    } catch {
      return null;
    }
  };

  const handleQRScan = async () => {
    if (!CameraView) {
      Alert.alert('Unavailable', 'QR scanning requires a development build. Use manual entry.');
      return;
    }
    try {
      const Camera = require('expo-camera').Camera;
      const { status } = await Camera.requestCameraPermissionsAsync();
      if (status === 'granted') {
        setScanning(true);
        setAddMethod('qr');
      } else {
        Alert.alert('Permission Denied', 'Camera permission is required');
      }
    } catch {
      Alert.alert('Error', 'QR scanning unavailable. Use manual entry.');
    }
  };

  const handleBarCodeScanned = ({ data }: { data: string }) => {
    setScanning(false);
    const parsed = parseOtpauthUrl(data);
    if (parsed) {
      addAccount(parsed.name, parsed.issuer, parsed.secret);
    } else {
      Alert.alert('Invalid QR Code', 'Not a valid TOTP QR code');
    }
  };

  const handleManualAdd = () => {
    if (!manualName.trim() || !manualSecret.trim()) {
      Alert.alert('Error', 'Enter account name and secret key');
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
    await saveAccounts([...accounts, newAccount]);

    // Log the TOTP add activity
    await addActivityLog({
      type: 'totp_added',
      title: 'TOTP Account Added',
      description: `Added ${issuer}: ${name} to authenticator`,
      metadata: { issuer, name },
    });

    setShowAddModal(false);
    setAddMethod(null);
    Alert.alert('Success', `${issuer}: ${name} added`);
  };

  const deleteAccount = (id: string) => {
    const account = accounts.find(a => a.id === id);
    Alert.alert('Delete Account', 'Are you sure?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: async () => {
          await saveAccounts(accounts.filter(a => a.id !== id));

          // Log the TOTP delete activity
          if (account) {
            await addActivityLog({
              type: 'totp_deleted',
              title: 'TOTP Account Removed',
              description: `Removed ${account.issuer}: ${account.name} from authenticator`,
              metadata: { issuer: account.issuer, name: account.name },
            });
          }
        }
      },
    ]);
  };

  const styles = createStyles(colors, isDark);
  const isLowTime = timeRemaining <= 5;
  const progress = timeRemaining / 30;

  const renderAccount = ({ item }: { item: TOTPAccount }) => {
    const code = totpCodes[item.id] || '------';
    const formattedCode = `${code.slice(0, 3)} ${code.slice(3)}`;

    return (
      <TouchableOpacity
        style={styles.accountCard}
        onLongPress={() => deleteAccount(item.id)}
        activeOpacity={0.7}>
        <View style={styles.accountLeft}>
          <View style={[styles.accountIcon, { backgroundColor: colors.primary + '20' }]}>
            <Text style={[styles.accountIconText, { color: colors.primary }]}>
              {item.issuer.charAt(0).toUpperCase()}
            </Text>
          </View>
          <View style={styles.accountInfo}>
            <Text style={styles.accountIssuer}>{item.issuer}</Text>
            <Text style={styles.accountName}>{item.name}</Text>
          </View>
        </View>
        <View style={styles.accountRight}>
          <Text style={[styles.accountCode, isLowTime && { color: colors.error }]}>
            {formattedCode}
          </Text>
          <View style={styles.timerContainer}>
            <View style={[styles.timerCircle, isLowTime && { borderColor: colors.error }]}>
              <Text style={[styles.timerText, isLowTime && { color: colors.error }]}>
                {timeRemaining}
              </Text>
            </View>
          </View>
        </View>
      </TouchableOpacity>
    );
  };

  return (
    <View style={styles.container}>
      <StatusBar
        barStyle={isDark ? 'light-content' : 'dark-content'}
        backgroundColor={colors.background}
      />

      {/* Header */}
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Image source={isDark ? require('../../appicon_dark.png') : require('../../appicon.png')} style={styles.headerLogo} resizeMode="contain" />
          <View>
            <Text style={styles.headerTitle}>AuthSec Authenticator</Text>
            <Text style={styles.headerSubtitle}>{email}</Text>
          </View>
        </View>
        <TouchableOpacity style={styles.notificationBtn} onPress={() => navigation.navigate('Notifications')}>
          <Ionicons name="notifications-outline" size={28} color={colors.text} />
          {unreadCount > 0 && (
            <View style={styles.notificationBadge}>
              <Text style={styles.notificationBadgeText}>
                {unreadCount > 99 ? '99+' : unreadCount}
              </Text>
            </View>
          )}
        </TouchableOpacity>
      </View>

      {/* Status Banner */}
      <View style={[styles.statusBanner, { backgroundColor: deviceRegistered ? colors.success + '15' : colors.warning + '15' }]}>
        <View style={[styles.statusDot, { backgroundColor: deviceRegistered ? colors.success : colors.warning }]} />
        <Text style={[styles.statusText, { color: deviceRegistered ? colors.success : colors.warning }]}>
          {deviceRegistered ? 'Push notifications active' : 'Setting up notifications...'}
        </Text>
      </View>

      {accounts.length === 0 ? (
        <View style={styles.emptyState}>
          <Ionicons name="lock-closed" size={64} color={colors.textMuted} />
          <Text style={styles.emptyTitle}>No Accounts Yet</Text>
          <Text style={styles.emptyText}>Tap the + button to add your first 2FA account</Text>
        </View>
      ) : (
        <FlatList
          data={accounts}
          renderItem={renderAccount}
          keyExtractor={item => item.id}
          contentContainerStyle={styles.listContent}
          showsVerticalScrollIndicator={false}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} colors={[colors.primary]} />
          }
        />
      )}

      {/* FAB - Add Button */}
      <Animated.View style={[styles.fab, { transform: [{ scale: fabScale }] }]}>
        <TouchableOpacity style={styles.fabButton} onPress={handleFabPress} activeOpacity={0.8}>
          <Ionicons name="add" size={32} color={isDark ? '#000000' : '#FFFFFF'} />
        </TouchableOpacity>
      </Animated.View>

      {/* Add Modal */}
      <Modal visible={showAddModal} animationType="slide" transparent onRequestClose={() => { setShowAddModal(false); setAddMethod(null); setScanning(false); }}>
        <View style={styles.modalOverlay}>
          <TouchableOpacity style={styles.modalBackdrop} onPress={() => { setShowAddModal(false); setAddMethod(null); setScanning(false); }} />
          <View style={styles.modalContent}>
            <View style={styles.modalHandle} />

            {scanning && CameraView ? (
              <View style={styles.scannerContainer}>
                <CameraView
                  style={styles.camera}
                  facing="back"
                  onBarcodeScanned={handleBarCodeScanned}
                  barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
                />
                <View style={styles.scannerOverlay}>
                  <View style={styles.scannerFrame} />
                </View>
                <TouchableOpacity style={styles.cancelScanBtn} onPress={() => { setScanning(false); setAddMethod(null); }}>
                  <Text style={styles.cancelScanText}>Cancel</Text>
                </TouchableOpacity>
              </View>
            ) : !addMethod ? (
              <>
                <Text style={styles.modalTitle}>Add Account</Text>
                <Text style={styles.modalSubtitle}>Choose how to add your 2FA account</Text>

                <TouchableOpacity style={styles.methodCard} onPress={handleQRScan}>
                  <View style={[styles.methodIcon, { backgroundColor: colors.primary + '15' }]}>
                    <Ionicons name="camera" size={24} color={colors.primary} />
                  </View>
                  <View style={styles.methodContent}>
                    <Text style={styles.methodTitle}>Scan QR Code</Text>
                    <Text style={styles.methodDesc}>Quick and easy setup</Text>
                  </View>
                </TouchableOpacity>

                <TouchableOpacity style={styles.methodCard} onPress={() => setAddMethod('manual')}>
                  <View style={[styles.methodIcon, { backgroundColor: colors.info + '15' }]}>
                    <Ionicons name="pencil" size={24} color={colors.info} />
                  </View>
                  <View style={styles.methodContent}>
                    <Text style={styles.methodTitle}>Enter Manually</Text>
                    <Text style={styles.methodDesc}>Type the secret key</Text>
                  </View>
                </TouchableOpacity>
              </>
            ) : (
              <>
                <Text style={styles.modalTitle}>Manual Entry</Text>

                <View style={styles.inputGroup}>
                  <Text style={styles.inputLabel}>Account Name *</Text>
                  <TextInput
                    style={styles.input}
                    placeholder="e.g., user@example.com"
                    placeholderTextColor={colors.inputPlaceholder}
                    value={manualName}
                    onChangeText={setManualName}
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
                  />
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
                </View>

                <View style={styles.modalButtons}>
                  <TouchableOpacity style={styles.cancelBtn} onPress={() => setAddMethod(null)}>
                    <Text style={styles.cancelBtnText}>Back</Text>
                  </TouchableOpacity>
                  <TouchableOpacity style={styles.addBtn} onPress={handleManualAdd}>
                    <Text style={styles.addBtnText}>Add Account</Text>
                  </TouchableOpacity>
                </View>
              </>
            )}
          </View>
        </View>
      </Modal>
    </View>
  );
};

const createStyles = (colors: any, isDark: boolean) => StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingTop: 75,
    paddingBottom: 16,
    backgroundColor: colors.background,
    borderBottomWidth: 1,
    borderBottomColor: colors.divider,
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  headerLogo: {
    width: 40,
    height: 40,
    marginRight: 12,
    borderRadius: 10,
  },
  headerTitle: {
    fontSize: 22,
    fontWeight: '700',
    color: colors.text,
  },
  headerSubtitle: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  notificationBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: colors.surface,
    justifyContent: 'center',
    alignItems: 'center',
    position: 'relative',
  },
  notificationBadge: {
    position: 'absolute',
    top: -2,
    right: -2,
    backgroundColor: colors.error,
    borderRadius: 10,
    minWidth: 20,
    height: 20,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 5,
  },
  notificationBadgeText: {
    color: '#FFFFFF',
    fontSize: 11,
    fontWeight: '700',
  },
  statusBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingVertical: 10,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 8,
  },
  statusText: {
    fontSize: 13,
    fontWeight: '500',
  },
  listContent: {
    padding: 16,
    paddingBottom: 100,
  },
  accountCard: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: colors.card,
    borderRadius: 16,
    padding: 16,
    marginBottom: 12,
    shadowColor: colors.shadow,
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 2,
  },
  accountLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  accountIcon: {
    width: 48,
    height: 48,
    borderRadius: 14,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 14,
  },
  accountIconText: {
    fontSize: 20,
    fontWeight: '700',
  },
  accountInfo: {
    flex: 1,
  },
  accountIssuer: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  accountName: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  accountRight: {
    alignItems: 'flex-end',
  },
  accountCode: {
    fontSize: 24,
    fontWeight: '700',
    color: colors.text,
    letterSpacing: 2,
    fontFamily: 'monospace',
    marginBottom: 4,
  },
  timerContainer: {
    alignItems: 'center',
  },
  timerCircle: {
    width: 28,
    height: 28,
    borderRadius: 14,
    borderWidth: 2,
    borderColor: colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
  },
  timerText: {
    fontSize: 11,
    fontWeight: '700',
    color: colors.primary,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 40,
    marginBottom: 80,
  },
  emptyTitle: {
    fontSize: 22,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 8,
  },
  emptyText: {
    fontSize: 15,
    color: colors.textSecondary,
    textAlign: 'center',
  },
  fab: {
    position: 'absolute',
    bottom: 30,
    right: 20,
  },
  fabButton: {
    width: 60,
    height: 60,
    borderRadius: 30,
    backgroundColor: colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
    shadowColor: colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 8,
    elevation: 8,
  },
  modalOverlay: {
    flex: 1,
    justifyContent: 'flex-end',
    backgroundColor: colors.overlay,
  },
  modalBackdrop: {
    flex: 1,
  },
  modalContent: {
    backgroundColor: colors.card,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    padding: 24,
    maxHeight: '80%',
  },
  modalHandle: {
    width: 40,
    height: 4,
    backgroundColor: colors.border,
    borderRadius: 2,
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
  },
  methodCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.surface,
    borderRadius: 16,
    padding: 16,
    marginBottom: 12,
  },
  methodIcon: {
    width: 50,
    height: 50,
    borderRadius: 14,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 14,
  },
  methodContent: {
    flex: 1,
  },
  methodTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  methodDesc: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  inputGroup: {
    marginBottom: 16,
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
    padding: 14,
    fontSize: 16,
    color: colors.text,
  },
  modalButtons: {
    flexDirection: 'row',
    gap: 12,
    marginTop: 8,
  },
  cancelBtn: {
    flex: 1,
    padding: 16,
    borderRadius: 12,
    backgroundColor: colors.surface,
    alignItems: 'center',
    borderColor: colors.inputBorder,
    borderWidth: 1,
  },
  cancelBtnText: {
    fontSize: 16,
    fontWeight: '600',
    color: isDark ? '#ffffff' : '#000000',
  },
  addBtn: {
    flex: 1,
    padding: 16,
    borderRadius: 12,
    backgroundColor: colors.primary,
    alignItems: 'center',
    borderColor: isDark ? '#000000' : colors.inputBorder,
    borderWidth: isDark ? 1.5 : 0,
  },
  addBtnText: {
    fontSize: 16,
    fontWeight: '600',
    color: isDark ? '#000000' : '#FFFFFF',
  },
  scannerContainer: {
    height: 400,
    borderRadius: 16,
    overflow: 'hidden',
  },
  camera: {
    flex: 1,
  },
  scannerOverlay: {
    ...StyleSheet.absoluteFillObject,
    justifyContent: 'center',
    alignItems: 'center',
  },
  scannerFrame: {
    width: 250,
    height: 250,
    borderWidth: 2,
    borderColor: '#FFFFFF',
    borderRadius: 20,
  },
  cancelScanBtn: {
    position: 'absolute',
    bottom: 20,
    alignSelf: 'center',
    backgroundColor: 'rgba(0,0,0,0.6)',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 20,
  },
  cancelScanText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});

export default HomeScreen;
