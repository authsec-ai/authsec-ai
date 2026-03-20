import React, {useCallback, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  Alert,
  BackHandler,
  Switch,
  TextInput,
  Modal,
} from 'react-native';
import {useTheme} from '../context/ThemeContext';
import {
  clearStorage,
  getBiometricEnabled,
  setBiometricEnabled,
  getBiometricAuthRequests,
  setBiometricAuthRequests,
  getAppPin,
  storeAppPin,
  getAppLockEnabled,
  setAppLockEnabled,
  isPinSetupCompleted,
  setPinSetupCompleted,
} from '../services/storage';
import {Ionicons} from '@expo/vector-icons';
import {useFocusEffect} from '@react-navigation/native';

const SettingsScreen = ({navigation}: any) => {
  const {colors, isDark, themeMode, setThemeMode} = useTheme();
  const [appLockEnabled, setAppLockEnabledState] = useState(false);
  const [biometricEnabled, setBiometricEnabledState] = useState(false);
  const [biometricAuthRequests, setBiometricAuthRequestsState] = useState(false);
  const [changePinModalVisible, setChangePinModalVisible] = useState(false);
  const [currentPin, setCurrentPin] = useState('');
  const [newPin, setNewPin] = useState('');
  const [confirmPin, setConfirmPin] = useState('');
  const [hasPinSetup, setHasPinSetup] = useState(false);

  // Load settings when screen is focused
  useFocusEffect(
    useCallback(() => {
      const loadSettings = async () => {
        const appLock = await getAppLockEnabled();
        const biometric = await getBiometricEnabled();
        const biometricAuth = await getBiometricAuthRequests();
        const pinSetup = await isPinSetupCompleted();
        setAppLockEnabledState(appLock);
        setBiometricEnabledState(biometric);
        setBiometricAuthRequestsState(biometricAuth);
        setHasPinSetup(pinSetup);
      };
      loadSettings();
    }, []),
  );

  // Handle Android hardware back button on Settings screen
  useFocusEffect(
    useCallback(() => {
      const backHandler = BackHandler.addEventListener('hardwareBackPress', () => {
        navigation.goBack();
        return true; // prevent default behavior
      });

      return () => backHandler.remove();
    }, [navigation]),
  );

  const handleThemeToggle = () => {
    // Cycle: system → light → dark → system
    if (themeMode === 'system') {
      setThemeMode('light');
    } else if (themeMode === 'light') {
      setThemeMode('dark');
    } else {
      setThemeMode('system');
    }
  };

  const handleAppLockToggle = async (value: boolean) => {
    // Check if PIN is set up
    const pinExists = await getAppPin();
    const pinSetupDone = await isPinSetupCompleted();

    if (value && (!pinExists || !pinSetupDone)) {
      // If enabling app lock but no PIN set, prompt to set one
      Alert.alert(
        'Set App PIN',
        'Please set a 6-digit PIN to enable app lock',
        [
          {text: 'Cancel', style: 'cancel'},
          {text: 'Set PIN', onPress: () => {
            setHasPinSetup(false);
            setCurrentPin('');
            setNewPin('');
            setConfirmPin('');
            setChangePinModalVisible(true);
          }},
        ],
      );
      return;
    }

    // Enable or disable app lock
    await setAppLockEnabled(value);
    setAppLockEnabledState(value);
  };

  const handleBiometricToggle = async (value: boolean) => {
    await setBiometricEnabled(value);
    setBiometricEnabledState(value);
  };

  const handleBiometricAuthRequestsToggle = async (value: boolean) => {
    await setBiometricAuthRequests(value);
    setBiometricAuthRequestsState(value);
  };

  const handleChangePin = async () => {
    setCurrentPin('');
    setNewPin('');
    setConfirmPin('');

    // Check if PIN already exists
    const existingPin = await getAppPin();
    setHasPinSetup(!!existingPin);

    setChangePinModalVisible(true);
  };

  const handleSavePin = async () => {
    // Check if current PIN is correct (if PIN already exists)
    const existingPin = await getAppPin();
    if (existingPin && currentPin !== existingPin) {
      Alert.alert('Error', 'Current PIN is incorrect');
      return;
    }

    // Validate new PIN
    if (newPin.length !== 6) {
      Alert.alert('Error', 'PIN must be 6 digits');
      return;
    }

    if (newPin !== confirmPin) {
      Alert.alert('Error', 'PINs do not match');
      return;
    }

    // Save the new PIN
    await storeAppPin(newPin);

    // If this is the first time setting up PIN (from app lock toggle), enable app lock
    if (!hasPinSetup) {
      await setPinSetupCompleted(true);
      await setAppLockEnabled(true);
      setAppLockEnabledState(true);
      setHasPinSetup(true);
    }

    setChangePinModalVisible(false);
    Alert.alert('Success', 'PIN updated successfully');
  };

  const handleLogout = () => {
    Alert.alert(
      'Logout',
      'Are you sure you want to logout?',
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Logout',
          style: 'destructive',
          onPress: async () => {
            await clearStorage();
            navigation.reset({
              index: 0,
              routes: [{name: 'ClientID'}],
            });
          },
        },
      ],
    );
  };

  const styles = createStyles(colors, isDark);

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Account</Text>
      </View>

      <ScrollView style={styles.scrollView} contentContainerStyle={styles.content}>
        {/* Appearance Section */}
        <Text style={styles.sectionTitle}>Appearance</Text>
        <View style={styles.card}>
          <TouchableOpacity style={styles.optionRow} onPress={handleThemeToggle}>
            <View style={styles.optionIcon}>
              <Ionicons
                name={themeMode === 'system' ? 'phone-portrait' : themeMode === 'light' ? 'sunny' : 'moon'}
                size={20}
                color={themeMode === 'system' ? colors.info : themeMode === 'light' ? colors.warning : colors.info}
              />
            </View>
            <View style={styles.optionContent}>
              <Text style={styles.optionTitle}>Theme</Text>
              <Text style={styles.optionDescription}>
                {themeMode === 'system' ? 'Using system settings' : themeMode === 'light' ? 'Light mode' : 'Dark mode'}
              </Text>
            </View>
            <View style={styles.themeBadge}>
              <Text style={styles.themeBadgeText}>
                {themeMode === 'system' ? 'System' : themeMode === 'light' ? 'Light' : 'Dark'}
              </Text>
            </View>
          </TouchableOpacity>
        </View>

        {/* Security Section */}
        <Text style={styles.sectionTitle}>Security</Text>
        <View style={styles.card}>
          {/* App Lock Toggle */}
          <View style={styles.optionRow}>
            <View style={styles.optionIcon}>
              <Ionicons name="lock-closed" size={20} color={colors.info} />
            </View>
            <View style={styles.optionContent}>
              <Text style={styles.optionTitle}>App Lock</Text>
              <Text style={styles.optionDescription}>
                {appLockEnabled ? 'Require PIN or biometric on app start' : 'Enable to secure your app'}
              </Text>
            </View>
            <Switch
              value={appLockEnabled}
              onValueChange={handleAppLockToggle}
              trackColor={{false: colors.border, true: colors.primary}}
              thumbColor={isDark && appLockEnabled ? colors.background : '#FFFFFF'}
              ios_backgroundColor={colors.border}
            />
          </View>

          {/* Biometric Unlock - only show if app lock is enabled and PIN is set up */}
          {appLockEnabled && hasPinSetup && (
            <>
              <View style={styles.divider} />
              <View style={styles.optionRow}>
                <View style={styles.optionIcon}>
                  <Ionicons name="finger-print" size={20} color={colors.success} />
                </View>
                <View style={styles.optionContent}>
                  <Text style={styles.optionTitle}>Biometric Unlock</Text>
                  <Text style={styles.optionDescription}>Use biometric to unlock app instead of PIN</Text>
                </View>
                <Switch
                  value={biometricEnabled}
                  onValueChange={handleBiometricToggle}
                  trackColor={{false: colors.border, true: colors.primary}}
                  thumbColor={isDark && biometricEnabled ? colors.background : '#FFFFFF'}
                  ios_backgroundColor={colors.border}
                />
              </View>
            </>
          )}

          {/* Biometric for Auth Requests - only show if PIN is set up */}
          {hasPinSetup && (
            <>
              <View style={styles.divider} />
              <View style={styles.optionRow}>
                <View style={styles.optionIcon}>
                  <Ionicons name="shield-checkmark" size={20} color={colors.primary} />
                </View>
                <View style={styles.optionContent}>
                  <Text style={styles.optionTitle}>Biometric for Auth Requests</Text>
                  <Text style={styles.optionDescription}>Use biometric to approve sign-in requests (PIN always available)</Text>
                </View>
                <Switch
                  value={biometricAuthRequests}
                  onValueChange={handleBiometricAuthRequestsToggle}
                  trackColor={{false: colors.border, true: colors.primary}}
                  thumbColor={isDark && biometricAuthRequests ? colors.background : '#FFFFFF'}
                  ios_backgroundColor={colors.border}
                />
              </View>
            </>
          )}

          {/* Change PIN - always show if PIN is set up since it's used for authentication */}
          {hasPinSetup && (
            <>
              <View style={styles.divider} />
              <TouchableOpacity style={styles.optionRow} onPress={handleChangePin}>
                <View style={styles.optionIcon}>
                  <Ionicons name="keypad" size={20} color={colors.warning} />
                </View>
                <View style={styles.optionContent}>
                  <Text style={styles.optionTitle}>Change App PIN</Text>
                  <Text style={styles.optionDescription}>Update your 6-digit PIN used for authentication</Text>
                </View>
                <Ionicons name="chevron-forward" size={20} color={colors.textMuted} />
              </TouchableOpacity>
            </>
          )}
        </View>

        {/* About Section */}
        <Text style={styles.sectionTitle}>About</Text>
        <View style={styles.card}>
          <View style={styles.optionRow}>
            <View style={styles.optionIcon}>
              <Ionicons name="information-circle" size={20} color={colors.info} />
            </View>
            <View style={styles.optionContent}>
              <Text style={styles.optionTitle}>Version</Text>
              <Text style={styles.optionDescription}>1.0.0</Text>
            </View>
          </View>
        </View>

        {/* Logout Button */}
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Text style={styles.logoutText}>Sign Out</Text>
        </TouchableOpacity>
      </ScrollView>

      {/* Change PIN Modal */}
      <Modal
        visible={changePinModalVisible}
        transparent={true}
        animationType="fade"
        onRequestClose={() => setChangePinModalVisible(false)}>
        <View style={styles.modalOverlay}>
          <View style={[styles.modalContent, {backgroundColor: colors.card}]}>
            <Text style={[styles.modalTitle, {color: colors.text}]}>
              {hasPinSetup ? 'Change App PIN' : 'Set App PIN'}
            </Text>

            {/* Current PIN (only if PIN already exists) */}
            {hasPinSetup && (
              <TextInput
                style={[styles.pinInput, {color: colors.text, borderColor: colors.border}]}
                placeholder="Current PIN (6 digits)"
                placeholderTextColor={colors.textMuted}
                secureTextEntry
                keyboardType="numeric"
                maxLength={6}
                value={currentPin}
                onChangeText={setCurrentPin}
              />
            )}

            {/* New PIN */}
            <TextInput
              style={[styles.pinInput, {color: colors.text, borderColor: colors.border}]}
              placeholder="New PIN (6 digits)"
              placeholderTextColor={colors.textMuted}
              secureTextEntry
              keyboardType="numeric"
              maxLength={6}
              value={newPin}
              onChangeText={setNewPin}
            />

            {/* Confirm PIN */}
            <TextInput
              style={[styles.pinInput, {color: colors.text, borderColor: colors.border}]}
              placeholder="Confirm New PIN (6 digits)"
              placeholderTextColor={colors.textMuted}
              secureTextEntry
              keyboardType="numeric"
              maxLength={6}
              value={confirmPin}
              onChangeText={setConfirmPin}
            />

            {/* Action Buttons */}
            <View style={styles.modalButtons}>
              <TouchableOpacity
                style={[styles.modalButton, styles.cancelButton, {backgroundColor: colors.errorLight}]}
                onPress={() => setChangePinModalVisible(false)}>
                <Text style={[styles.modalButtonText, {color: colors.error}]}>Cancel</Text>
              </TouchableOpacity>

              <TouchableOpacity
                style={[styles.modalButton, styles.saveButton, {backgroundColor: colors.primary}]}
                onPress={handleSavePin}>
                <Text style={[styles.modalButtonText, {color: isDark ? '#000000' : '#FFFFFF'}]}>
                  {hasPinSetup ? 'Save PIN' : 'Set PIN'}
                </Text>
              </TouchableOpacity>
            </View>
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
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      padding: 20,
      paddingTop: 60,
      backgroundColor: colors.card,
      borderBottomWidth: 1,
      borderBottomColor: colors.border,
    },
    headerTitle: {
      fontSize: 28,
      fontWeight: '700',
      color: colors.text,
    },
    scrollView: {
      flex: 1,
    },
    content: {
      padding: 20,
    },
    sectionTitle: {
      fontSize: 13,
      fontWeight: '600',
      color: colors.textMuted,
      textTransform: 'uppercase',
      letterSpacing: 0.5,
      marginBottom: 12,
      marginTop: 24,
      marginLeft: 4,
    },
    card: {
      backgroundColor: colors.card,
      borderRadius: 16,
      overflow: 'hidden',
      shadowColor: colors.shadow,
      shadowOffset: {width: 0, height: 2},
      shadowOpacity: isDark ? 0.3 : 0.08,
      shadowRadius: 8,
      elevation: 3,
    },
    optionRow: {
      flexDirection: 'row',
      alignItems: 'center',
      padding: 16,
    },
    optionRowSelected: {
      backgroundColor: colors.primaryLight,
    },
    optionIcon: {
      width: 40,
      height: 40,
      borderRadius: 12,
      backgroundColor: colors.background,
      justifyContent: 'center',
      alignItems: 'center',
      marginRight: 12,
    },
    optionContent: {
      flex: 1,
    },
    optionTitle: {
      fontSize: 16,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 2,
    },
    optionDescription: {
      fontSize: 13,
      color: colors.textSecondary,
    },
    checkmark: {
      width: 24,
      height: 24,
      borderRadius: 12,
      backgroundColor: colors.primary,
      justifyContent: 'center',
      alignItems: 'center',
    },
    themeBadge: {
      paddingHorizontal: 12,
      paddingVertical: 6,
      borderRadius: 12,
      backgroundColor: colors.primaryLight,
    },
    themeBadgeText: {
      fontSize: 13,
      fontWeight: '600',
      color: colors.primary,
    },
    chevron: {
      fontSize: 24,
      color: colors.textMuted,
    },
    divider: {
      height: 1,
      backgroundColor: colors.divider,
      marginLeft: 68,
    },
    logoutButton: {
      backgroundColor: colors.errorLight,
      borderRadius: 16,
      padding: 18,
      alignItems: 'center',
      marginTop: 32,
      marginBottom: 40,
    },
    logoutText: {
      fontSize: 16,
      fontWeight: '600',
      color: colors.error,
    },
    modalOverlay: {
      flex: 1,
      backgroundColor: 'rgba(0, 0, 0, 0.5)',
      justifyContent: 'center',
      alignItems: 'center',
      padding: 20,
    },
    modalContent: {
      width: '100%',
      maxWidth: 400,
      borderRadius: 20,
      padding: 24,
      shadowColor: '#000',
      shadowOffset: {width: 0, height: 4},
      shadowOpacity: 0.3,
      shadowRadius: 8,
      elevation: 8,
    },
    modalTitle: {
      fontSize: 20,
      fontWeight: '700',
      marginBottom: 24,
      textAlign: 'center',
    },
    pinInput: {
      borderWidth: 1,
      borderRadius: 12,
      padding: 16,
      fontSize: 16,
      marginBottom: 16,
      textAlign: 'left',
    },
    modalButtons: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      marginTop: 8,
      gap: 12,
    },
    modalButton: {
      flex: 1,
      padding: 16,
      borderRadius: 12,
      alignItems: 'center',
    },
    cancelButton: {},
    saveButton: {},
    modalButtonText: {
      fontSize: 16,
      fontWeight: '600',
    },
  });

export default SettingsScreen;
