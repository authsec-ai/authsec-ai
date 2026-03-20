import React, { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Modal,
  Animated,
  Dimensions,
  Platform,
  Vibration,
  TextInput,
  Alert,
  KeyboardAvoidingView,
  ScrollView,
} from 'react-native';
import * as LocalAuthentication from 'expo-local-authentication';
import * as Haptics from 'expo-haptics';
import { useTheme } from '../context/ThemeContext';
import { respondToCIBA } from '../services/api';
import { getStoredToken, addActivityLog, getBiometricAuthRequests, getAppPin } from '../services/storage';
import { Ionicons } from '@expo/vector-icons';

const { width } = Dimensions.get('window');

interface AuthRequestPopupProps {
  visible: boolean;
  authReqId: string;
  onClose: () => void;
  onResponse: (approved: boolean) => void;
}

const AuthRequestPopup: React.FC<AuthRequestPopupProps> = ({
  visible,
  authReqId,
  onClose,
  onResponse,
}) => {
  const { colors, isDark } = useTheme();
  const styles = createStyles(colors, isDark);

  const [showPinEntry, setShowPinEntry] = useState(false);
  const [pin, setPin] = useState('');
  const [loading, setLoading] = useState(false);

  const slideAnim = useRef(new Animated.Value(300)).current;
  const fadeAnim = useRef(new Animated.Value(0)).current;
  const scaleAnim = useRef(new Animated.Value(0.8)).current;
  const pulseAnim = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    if (visible) {
      // Haptic feedback when popup appears
      if (Platform.OS === 'ios') {
        Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
      } else {
        Vibration.vibrate([0, 100, 50, 100]);
      }

      // Animate in
      Animated.parallel([
        Animated.spring(slideAnim, {
          toValue: 0,
          useNativeDriver: true,
          tension: 65,
          friction: 10,
        }),
        Animated.timing(fadeAnim, {
          toValue: 1,
          duration: 200,
          useNativeDriver: true,
        }),
        Animated.spring(scaleAnim, {
          toValue: 1,
          useNativeDriver: true,
          tension: 65,
          friction: 8,
        }),
      ]).start();

      // Start pulse animation
      Animated.loop(
        Animated.sequence([
          Animated.timing(pulseAnim, {
            toValue: 1.05,
            duration: 1000,
            useNativeDriver: true,
          }),
          Animated.timing(pulseAnim, {
            toValue: 1,
            duration: 1000,
            useNativeDriver: true,
          }),
        ])
      ).start();
    } else {
      // Animate out
      Animated.parallel([
        Animated.timing(slideAnim, {
          toValue: 300,
          duration: 200,
          useNativeDriver: true,
        }),
        Animated.timing(fadeAnim, {
          toValue: 0,
          duration: 150,
          useNativeDriver: true,
        }),
      ]).start();
    }
  }, [visible]);

  const handleApprove = async () => {
    try {
      // Haptic feedback
      if (Platform.OS === 'ios') {
        Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
      }

      // Check if biometric is enabled for auth requests
      const biometricAuthEnabled = await getBiometricAuthRequests();

      if (biometricAuthEnabled) {
        // Try biometric authentication first
        const result = await LocalAuthentication.authenticateAsync({
          promptMessage: 'Approve Sign-in Request',
          fallbackLabel: 'Use PIN',
          disableDeviceFallback: false,
        });

        if (result.success) {
          await approveRequest();
        } else {
          // Biometric failed or cancelled, show PIN entry
          setShowPinEntry(true);
        }
      } else {
        // Biometric not enabled, go directly to PIN entry
        setShowPinEntry(true);
      }
    } catch (error) {
      console.error('Approval error:', error);
      // On error, fallback to PIN entry
      setShowPinEntry(true);
    }
  };

  const approveRequest = async () => {
    try {
      const token = await getStoredToken();
      if (token) {
        await respondToCIBA(authReqId, true, true, token);

        // Log the approval activity
        await addActivityLog({
          type: 'auth_approved',
          title: 'Sign-in Approved',
          description: `Approved authentication request from Web Browser`,
          metadata: { authReqId },
        });

        if (Platform.OS === 'ios') {
          Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
        }
        onResponse(true);
      }
    } catch (error) {
      console.error('Approve request error:', error);
      Alert.alert('Error', 'Failed to approve request');
    }
  };

  const handlePinSubmit = async (pinValue?: string) => {
    const pinToCheck = pinValue || pin;

    if (pinToCheck.length !== 6) {
      Alert.alert('Invalid PIN', 'Please enter your 6-digit PIN');
      return;
    }

    setLoading(true);
    const storedPin = await getAppPin();

    if (pinToCheck === storedPin) {
      await approveRequest();
      setPin('');
      setShowPinEntry(false);
    } else {
      Alert.alert('Incorrect PIN', 'Please try again');
      setPin('');
    }
    setLoading(false);
  };

  const handlePinChange = (text: string) => {
    if (/^\d*$/.test(text) && text.length <= 6) {
      setPin(text);
      // Auto-submit when 6 digits entered
      if (text.length === 6) {
        setTimeout(() => handlePinSubmit(text), 100);
      }
    }
  };

  const handleDeny = async () => {
    try {
      if (Platform.OS === 'ios') {
        Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
      }

      const token = await getStoredToken();
      if (token) {
        await respondToCIBA(authReqId, false, false, token);

        // Log the denial activity
        await addActivityLog({
          type: 'auth_denied',
          title: 'Sign-in Denied',
          description: `Denied authentication request from Web Browser`,
          metadata: { authReqId },
        });

        Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
        onResponse(false);
      }
    } catch (error) {
      console.error('Deny error:', error);
    }
  };

  return (
    <Modal
      visible={visible}
      transparent
      animationType="none"
      statusBarTranslucent
      onRequestClose={onClose}>
      <Animated.View style={[styles.overlay, { opacity: fadeAnim }]}>
        <TouchableOpacity
          style={styles.backdrop}
          activeOpacity={1}
          onPress={onClose}
        />

        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          style={styles.keyboardAvoidingView}>
          <ScrollView
            contentContainerStyle={styles.scrollContent}
            keyboardShouldPersistTaps="handled"
            showsVerticalScrollIndicator={false}>
            <Animated.View
              style={[
                styles.popup,
                {
                  transform: [
                    { translateY: slideAnim },
                    { scale: scaleAnim },
                  ],
                },
              ]}>
          {/* Header with pulse animation */}
          <Animated.View style={[styles.header, { transform: [{ scale: pulseAnim }] }]}>
            <View style={styles.alertIcon}>
              <Ionicons name="lock-closed" size={36} color={isDark ? '#FFFFFF' : '#000000'} />
            </View>
          </Animated.View>

          <Text style={styles.title}>Sign-in Request</Text>
          <Text style={styles.subtitle}>
            Someone is trying to sign in to your account
          </Text>

          {/* Info Cards */}
          <View style={styles.infoCard}>
            <View style={styles.infoRow}>
              <Ionicons name="globe" size={20} color={isDark ? '#FFFFFF' : '#000000'} style={{ marginRight: 12 }} />
              <View style={styles.infoContent}>
                <Text style={styles.infoLabel}>Application</Text>
                <Text style={styles.infoValue}>Web Browser Login</Text>
              </View>
            </View>
            <View style={styles.infoDivider} />
            <View style={styles.infoRow}>
              <Ionicons name="time" size={20} color={isDark ? '#FFFFFF' : '#000000'} style={{ marginRight: 12 }} />
              <View style={styles.infoContent}>
                <Text style={styles.infoLabel}>Time</Text>
                <Text style={styles.infoValue}>{new Date().toLocaleTimeString()}</Text>
              </View>
            </View>
            <View style={styles.infoDivider} />
            <View style={styles.infoRow}>
              <Ionicons name="key" size={20} color={isDark ? '#FFFFFF' : '#000000'} style={{ marginRight: 12 }} />
              <View style={styles.infoContent}>
                <Text style={styles.infoLabel}>Request ID</Text>
                <Text style={styles.infoValueSmall} numberOfLines={1}>{authReqId}</Text>
              </View>
            </View>
          </View>

          {/* Warning */}
          <View style={styles.warningBox}>
            <Ionicons name="warning" size={18} color={isDark ? '#F5A623' : '#92400E'} style={{ marginRight: 10 }} />
            <Text style={styles.warningText}>
              Only approve if you initiated this sign-in
            </Text>
          </View>

          {/* PIN Entry or Action Buttons */}
          {showPinEntry ? (
            <View style={styles.pinEntryContainer}>
              <Text style={styles.pinEntryTitle}>Enter Your PIN</Text>
              <Text style={styles.pinEntrySubtitle}>Enter your 6-digit PIN to approve</Text>

              <TextInput
                style={styles.pinInput}
                value={pin}
                onChangeText={handlePinChange}
                keyboardType="numeric"
                secureTextEntry
                maxLength={6}
                placeholder="••••••"
                placeholderTextColor={colors.textMuted}
                editable={!loading}
              />

              <View style={styles.pinButtonContainer}>
                <TouchableOpacity
                  style={styles.pinCancelButton}
                  onPress={() => {
                    setShowPinEntry(false);
                    setPin('');
                  }}
                  disabled={loading}>
                  <Text style={styles.pinCancelText}>Cancel</Text>
                </TouchableOpacity>

                <TouchableOpacity
                  style={[styles.pinSubmitButton, loading && styles.pinSubmitButtonDisabled]}
                  onPress={() => handlePinSubmit()}
                  disabled={loading || pin.length !== 6}>
                  <Text style={styles.pinSubmitText}>
                    {loading ? 'Verifying...' : 'Submit'}
                  </Text>
                </TouchableOpacity>
              </View>
            </View>
          ) : (
            <View style={styles.buttonContainer}>
              <TouchableOpacity
                style={styles.denyButton}
                onPress={handleDeny}
                activeOpacity={0.8}>
                <Ionicons name="close" size={18} color={colors.error} />
                <Text style={styles.denyText}>Deny</Text>
              </TouchableOpacity>

              <TouchableOpacity
                style={styles.approveButton}
                onPress={handleApprove}
                activeOpacity={0.8}>
                <Ionicons name="checkmark" size={18} color={isDark ? '#000000' : '#FFFFFF'} />
                <Text style={styles.approveText}>Approve</Text>
              </TouchableOpacity>
            </View>
          )}
            </Animated.View>
          </ScrollView>
        </KeyboardAvoidingView>
      </Animated.View>
    </Modal>
  );
};

const createStyles = (colors: any, isDark: boolean) =>
  StyleSheet.create({
    overlay: {
      flex: 1,
      justifyContent: 'center',
      alignItems: 'center',
      backgroundColor: 'rgba(0, 0, 0, 0.6)',
    },
    backdrop: {
      ...StyleSheet.absoluteFillObject,
    },
    keyboardAvoidingView: {
      flex: 1,
      justifyContent: 'center',
      width: '100%',
    },
    scrollContent: {
      flexGrow: 1,
      justifyContent: 'center',
      alignItems: 'center',
    },
    popup: {
      width: width - 40,
      backgroundColor: colors.card,
      borderRadius: 28,
      padding: 24,
      alignItems: 'center',
      shadowColor: '#000',
      shadowOffset: { width: 0, height: 10 },
      shadowOpacity: 0.3,
      shadowRadius: 20,
      elevation: 20,
      marginVertical: 20,
    },
    header: {
      marginBottom: 16,
    },
    alertIcon: {
      width: 80,
      height: 80,
      borderRadius: 40,
      backgroundColor: colors.primaryLight,
      justifyContent: 'center',
      alignItems: 'center',
    },
    title: {
      fontSize: 24,
      fontWeight: '700',
      color: colors.text,
      marginBottom: 8,
    },
    subtitle: {
      fontSize: 15,
      color: colors.textSecondary,
      textAlign: 'center',
      marginBottom: 20,
    },
    infoCard: {
      width: '100%',
      backgroundColor: isDark ? colors.surface : colors.background,
      borderRadius: 16,
      padding: 16,
      marginBottom: 16,
    },
    infoRow: {
      flexDirection: 'row',
      alignItems: 'center',
      paddingVertical: 8,
    },
    infoContent: {
      flex: 1,
    },
    infoLabel: {
      fontSize: 12,
      color: colors.textMuted,
      marginBottom: 2,
    },
    infoValue: {
      fontSize: 15,
      fontWeight: '600',
      color: colors.text,
    },
    infoValueSmall: {
      fontSize: 13,
      fontWeight: '500',
      color: colors.textSecondary,
      fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
    },
    infoDivider: {
      height: 1,
      backgroundColor: colors.divider,
      marginVertical: 4,
    },
    warningBox: {
      flexDirection: 'row',
      alignItems: 'center',
      backgroundColor: isDark ? '#3d3520' : '#FEF3C7',
      borderRadius: 12,
      padding: 12,
      marginBottom: 20,
      width: '100%',
    },
    warningText: {
      fontSize: 13,
      color: isDark ? '#F5A623' : '#92400E',
      flex: 1,
    },
    buttonContainer: {
      flexDirection: 'row',
      gap: 12,
      width: '100%',
    },
    denyButton: {
      flex: 1,
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: colors.surface,
      borderRadius: 16,
      paddingVertical: 16,
      gap: 8,
    },
    denyText: {
      fontSize: 16,
      fontWeight: '700',
      color: colors.text,
    },
    approveButton: {
      flex: 1,
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: colors.primary,
      borderRadius: 16,
      paddingVertical: 16,
      gap: 8,
      borderColor: isDark ? '#000000' : 'transparent',
      borderWidth: isDark ? 1.5 : 0,
    },
    approveText: {
      fontSize: 16,
      fontWeight: '700',
      color: isDark ? '#000000' : '#FFFFFF',
    },
    pinEntryContainer: {
      width: '100%',
      marginTop: 8,
    },
    pinEntryTitle: {
      fontSize: 18,
      fontWeight: '700',
      color: colors.text,
      textAlign: 'center',
      marginBottom: 8,
    },
    pinEntrySubtitle: {
      fontSize: 14,
      color: colors.textSecondary,
      textAlign: 'center',
      marginBottom: 20,
    },
    pinInput: {
      backgroundColor: colors.background,
      borderWidth: 2,
      borderColor: colors.primary,
      borderRadius: 12,
      padding: 16,
      fontSize: 24,
      fontWeight: '600',
      color: colors.text,
      textAlign: 'center',
      letterSpacing: 8,
      marginBottom: 20,
    },
    pinButtonContainer: {
      flexDirection: 'row',
      gap: 12,
      width: '100%',
    },
    pinCancelButton: {
      flex: 1,
      paddingVertical: 14,
      borderRadius: 12,
      backgroundColor: colors.surface,
      alignItems: 'center',
      borderWidth: 1,
      borderColor: colors.border,
    },
    pinCancelText: {
      fontSize: 16,
      fontWeight: '600',
      color: colors.text,
    },
    pinSubmitButton: {
      flex: 1,
      paddingVertical: 14,
      borderRadius: 12,
      backgroundColor: colors.primary,
      alignItems: 'center',
    },
    pinSubmitButtonDisabled: {
      opacity: 0.5,
    },
    pinSubmitText: {
      fontSize: 16,
      fontWeight: '700',
      color: isDark ? '#000000' : '#FFFFFF',
    },
  });

export default AuthRequestPopup;
