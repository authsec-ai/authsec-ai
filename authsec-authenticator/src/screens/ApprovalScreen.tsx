import React, {useState, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  Alert,
  Animated,
  StatusBar,
  Platform,
  Dimensions,
  TextInput,
  KeyboardAvoidingView,
  ScrollView,
} from 'react-native';
import {respondToCIBA} from '../services/api';
import {getStoredToken, getStoredEmail, addActivityLog, getBiometricAuthRequests, getAppPin} from '../services/storage';
import {authenticateWithBiometric} from '../services/biometric';
import {useTheme} from '../context/ThemeContext';
import {Ionicons} from '@expo/vector-icons';

const {width} = Dimensions.get('window');

const ApprovalScreen = ({route, navigation}: any) => {
  const {colors, isDark} = useTheme();
  const {authReqId} = route.params;
  const [loading, setLoading] = useState(false);
  const [email, setEmail] = useState('');
  const [showPinEntry, setShowPinEntry] = useState(false);
  const [pin, setPin] = useState('');
  const fadeAnim = useState(new Animated.Value(0))[0];
  const slideAnim = useState(new Animated.Value(30))[0];
  const scaleAnim = useState(new Animated.Value(0.95))[0];

  useEffect(() => {
    loadEmail();
    
    // Animate in
    Animated.parallel([
      Animated.timing(fadeAnim, {
        toValue: 1,
        duration: 400,
        useNativeDriver: true,
      }),
      Animated.spring(slideAnim, {
        toValue: 0,
        tension: 50,
        friction: 8,
        useNativeDriver: true,
      }),
      Animated.spring(scaleAnim, {
        toValue: 1,
        tension: 50,
        friction: 8,
        useNativeDriver: true,
      }),
    ]).start();
  }, []);

  const loadEmail = async () => {
    const storedEmail = await getStoredEmail();
    if (storedEmail) {
      setEmail(storedEmail);
    }
  };

  const animateOut = (callback: () => void) => {
    Animated.parallel([
      Animated.timing(fadeAnim, {
        toValue: 0,
        duration: 200,
        useNativeDriver: true,
      }),
      Animated.timing(scaleAnim, {
        toValue: 0.95,
        duration: 200,
        useNativeDriver: true,
      }),
    ]).start(callback);
  };

  const handleResponse = async (approved: boolean) => {
    if (!approved) {
      // Deny without authentication
      await handleDeny();
      return;
    }

    // For approval, check if biometric is enabled
    try {
      const biometricAuthEnabled = await getBiometricAuthRequests();

      if (biometricAuthEnabled) {
        // Try biometric authentication first
        const result = await authenticateWithBiometric('Verify your identity');

        if (result) {
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
      console.error('Auth error:', error);
      // On error, fallback to PIN entry
      setShowPinEntry(true);
    }
  };

  const approveRequest = async () => {
    setLoading(true);
    try {
      const authToken = await getStoredToken();
      if (!authToken) {
        Alert.alert('Error', 'Not logged in. Please login first.');
        setLoading(false);
        navigation.goBack();
        return;
      }

      await respondToCIBA(authReqId, true, true, authToken);

      // Log the approval activity
      await addActivityLog({
        type: 'auth_approved',
        title: 'Sign-in Approved',
        description: `Approved CIBA authentication request`,
        metadata: { authReqId },
      });

      setLoading(false);
      animateOut(() => {
        Alert.alert('✓ Approved', 'Request approved successfully', [
          {text: 'Done', onPress: () => navigation.goBack()}
        ]);
      });
    } catch (error: any) {
      Alert.alert('Error', error.message);
      setLoading(false);
    }
  };

  const handleDeny = async () => {
    setLoading(true);
    try {
      const authToken = await getStoredToken();
      if (!authToken) {
        Alert.alert('Error', 'Not logged in. Please login first.');
        setLoading(false);
        navigation.goBack();
        return;
      }

      await respondToCIBA(authReqId, false, false, authToken);

      // Log the denial activity
      await addActivityLog({
        type: 'auth_denied',
        title: 'Sign-in Denied',
        description: `Denied CIBA authentication request`,
        metadata: { authReqId },
      });

      setLoading(false);
      animateOut(() => {
        Alert.alert('✗ Denied', 'Request has been denied', [
          {text: 'Done', onPress: () => navigation.goBack()}
        ]);
      });
    } catch (error: any) {
      Alert.alert('Error', error.message);
      setLoading(false);
    }
  };

  const handlePinSubmit = async () => {
    if (pin.length !== 6) {
      Alert.alert('Invalid PIN', 'Please enter your 6-digit PIN');
      return;
    }

    setLoading(true);
    const storedPin = await getAppPin();

    if (pin === storedPin) {
      setPin('');
      setShowPinEntry(false);
      await approveRequest();
    } else {
      Alert.alert('Incorrect PIN', 'Please try again');
      setPin('');
      setLoading(false);
    }
  };

  const handlePinChange = (text: string) => {
    if (/^\d*$/.test(text) && text.length <= 6) {
      setPin(text);
      // Auto-submit when 6 digits entered
      if (text.length === 6) {
        setTimeout(() => handlePinSubmit(), 100);
      }
    }
  };

  const styles = createStyles(colors, isDark);

  return (
    <View style={styles.overlay}>
      <StatusBar barStyle="light-content" backgroundColor="rgba(0,0,0,0.95)" />

      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboardAvoidingView}>
        <ScrollView
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}>
          <Animated.View
            style={[
              styles.container,
              {
                opacity: fadeAnim,
                transform: [
                  {translateY: slideAnim},
                  {scale: scaleAnim},
                ],
              }
            ]}>
        
        {/* Close Button */}
        <TouchableOpacity 
          style={styles.closeButton}
          onPress={() => navigation.goBack()}>
          <Text style={styles.closeButtonText}>×</Text>
        </TouchableOpacity>

        {/* Icon */}
        <View style={styles.iconContainer}>
          <View style={styles.iconRing}>
            <Ionicons name="lock-closed" size={44} color={colors.primary} />
          </View>
        </View>

        {/* Title */}
        <Text style={styles.title}>Sign-in Request</Text>
        <Text style={styles.subtitle}>Someone is trying to access your account</Text>
        
        {/* Info Cards */}
        <View style={styles.infoSection}>
          <View style={styles.infoCard}>
            <View style={styles.infoRow}>
              <Ionicons name="person" size={22} color={colors.info} style={{marginRight: 14}} />
              <View style={styles.infoContent}>
                <Text style={styles.infoLabel}>Account</Text>
                <Text style={styles.infoValue} numberOfLines={1}>{email || 'Unknown'}</Text>
              </View>
            </View>
          </View>

          <View style={styles.infoCard}>
            <View style={styles.infoRow}>
              <Ionicons name="desktop" size={22} color={colors.success} style={{marginRight: 14}} />
              <View style={styles.infoContent}>
                <Text style={styles.infoLabel}>Application</Text>
                <Text style={styles.infoValue}>CIBA Authentication</Text>
              </View>
            </View>
          </View>

          <View style={styles.infoCard}>
            <View style={styles.infoRow}>
              <Ionicons name="time" size={22} color={colors.warning} style={{marginRight: 14}} />
              <View style={styles.infoContent}>
                <Text style={styles.infoLabel}>Time</Text>
                <Text style={styles.infoValue}>{new Date().toLocaleTimeString()}</Text>
              </View>
            </View>
          </View>
        </View>

        {/* Warning */}
        <View style={styles.warningBanner}>
          <Ionicons name="warning" size={20} color={colors.warning} style={{marginRight: 12}} />
          <Text style={styles.warningText}>
            Only approve if you initiated this sign-in
          </Text>
        </View>

        {/* Action Buttons or PIN Entry */}
        {loading ? (
          <View style={styles.loadingContainer}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={styles.loadingText}>Processing...</Text>
          </View>
        ) : showPinEntry ? (
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
            />

            <View style={styles.pinButtonContainer}>
              <TouchableOpacity
                style={styles.pinCancelButton}
                onPress={() => {
                  setShowPinEntry(false);
                  setPin('');
                }}>
                <Text style={styles.pinCancelText}>Cancel</Text>
              </TouchableOpacity>

              <TouchableOpacity
                style={[styles.pinSubmitButton, pin.length !== 6 && styles.pinSubmitButtonDisabled]}
                onPress={handlePinSubmit}
                disabled={pin.length !== 6}>
                <Text style={styles.pinSubmitText}>Submit</Text>
              </TouchableOpacity>
            </View>
          </View>
        ) : (
          <View style={styles.buttonContainer}>
            <TouchableOpacity
              style={styles.approveButton}
              onPress={() => handleResponse(true)}
              activeOpacity={0.85}>
              <Ionicons name="checkmark" size={20} color="#FFFFFF" style={{marginRight: 8}} />
              <Text style={styles.approveText}>Approve</Text>
            </TouchableOpacity>

            <TouchableOpacity
              style={styles.denyButton}
              onPress={() => handleResponse(false)}
              activeOpacity={0.85}>
              <Ionicons name="close" size={20} color={colors.error} style={{marginRight: 8}} />
              <Text style={styles.denyText}>Deny</Text>
            </TouchableOpacity>
          </View>
        )}
          </Animated.View>
        </ScrollView>
      </KeyboardAvoidingView>
    </View>
  );
};

const createStyles = (colors: any, isDark: boolean) =>
  StyleSheet.create({
    overlay: {
      flex: 1,
      backgroundColor: 'rgba(0, 0, 0, 0.95)',
      justifyContent: 'center',
      alignItems: 'center',
      padding: 20,
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
    container: {
      width: '100%',
      maxWidth: 380,
      backgroundColor: colors.surface,
      borderRadius: 28,
      padding: 28,
      shadowColor: '#000',
      shadowOffset: {width: 0, height: 20},
      shadowOpacity: 0.5,
      shadowRadius: 30,
      elevation: 20,
    },
    closeButton: {
      position: 'absolute',
      top: 16,
      right: 16,
      width: 36,
      height: 36,
      borderRadius: 18,
      backgroundColor: colors.background,
      justifyContent: 'center',
      alignItems: 'center',
      zIndex: 10,
    },
    closeButtonText: {
      fontSize: 24,
      color: colors.textMuted,
      marginTop: -2,
    },
    iconContainer: {
      alignItems: 'center',
      marginBottom: 20,
    },
    iconRing: {
      width: 88,
      height: 88,
      borderRadius: 44,
      backgroundColor: colors.primaryLight,
      justifyContent: 'center',
      alignItems: 'center',
      borderWidth: 3,
      borderColor: colors.primary,
    },
    title: {
      fontSize: 26,
      fontWeight: '700',
      color: colors.text,
      textAlign: 'center',
      marginBottom: 6,
    },
    subtitle: {
      fontSize: 14,
      color: colors.textSecondary,
      textAlign: 'center',
      marginBottom: 24,
    },
    infoSection: {
      marginBottom: 20,
    },
    infoCard: {
      backgroundColor: colors.background,
      borderRadius: 14,
      padding: 14,
      marginBottom: 10,
    },
    infoRow: {
      flexDirection: 'row',
      alignItems: 'center',
    },
    infoContent: {
      flex: 1,
    },
    infoLabel: {
      fontSize: 12,
      color: colors.textMuted,
      marginBottom: 2,
      textTransform: 'uppercase',
      letterSpacing: 0.5,
    },
    infoValue: {
      fontSize: 15,
      color: colors.text,
      fontWeight: '600',
    },
    warningBanner: {
      flexDirection: 'row',
      backgroundColor: colors.warningLight,
      borderRadius: 12,
      padding: 14,
      marginBottom: 24,
      alignItems: 'center',
    },
    warningText: {
      flex: 1,
      fontSize: 13,
      color: colors.warning,
      fontWeight: '500',
      lineHeight: 18,
    },
    loadingContainer: {
      alignItems: 'center',
      paddingVertical: 24,
    },
    loadingText: {
      marginTop: 12,
      fontSize: 14,
      color: colors.textSecondary,
    },
    buttonContainer: {
      flexDirection: 'row',
      gap: 12,
    },
    approveButton: {
      flex: 1,
      flexDirection: 'row',
      backgroundColor: colors.success,
      borderRadius: 14,
      paddingVertical: 18,
      alignItems: 'center',
      justifyContent: 'center',
      shadowColor: colors.success,
      shadowOffset: {width: 0, height: 4},
      shadowOpacity: 0.3,
      shadowRadius: 8,
      elevation: 4,
    },
    approveText: {
      fontSize: 17,
      fontWeight: '700',
      color: '#FFFFFF',
    },
    denyButton: {
      flex: 1,
      flexDirection: 'row',
      backgroundColor: colors.surface,
      borderRadius: 14,
      paddingVertical: 18,
      alignItems: 'center',
      justifyContent: 'center',
      borderWidth: 2,
      borderColor: colors.error,
    },
    denyText: {
      fontSize: 17,
      fontWeight: '700',
      color: colors.error,
    },
    pinEntryContainer: {
      width: '100%',
    },
    pinEntryTitle: {
      fontSize: 20,
      fontWeight: '700',
      color: colors.text,
      textAlign: 'center',
      marginBottom: 8,
    },
    pinEntrySubtitle: {
      fontSize: 14,
      color: colors.textSecondary,
      textAlign: 'center',
      marginBottom: 24,
    },
    pinInput: {
      backgroundColor: colors.background,
      borderWidth: 2,
      borderColor: colors.primary,
      borderRadius: 14,
      padding: 18,
      fontSize: 26,
      fontWeight: '600',
      color: colors.text,
      textAlign: 'center',
      letterSpacing: 10,
      marginBottom: 24,
    },
    pinButtonContainer: {
      flexDirection: 'row',
      gap: 12,
    },
    pinCancelButton: {
      flex: 1,
      paddingVertical: 16,
      borderRadius: 14,
      backgroundColor: colors.surface,
      alignItems: 'center',
      borderWidth: 2,
      borderColor: colors.error,
    },
    pinCancelText: {
      fontSize: 17,
      fontWeight: '700',
      color: colors.error,
    },
    pinSubmitButton: {
      flex: 1,
      paddingVertical: 16,
      borderRadius: 14,
      backgroundColor: colors.success,
      alignItems: 'center',
    },
    pinSubmitButtonDisabled: {
      opacity: 0.5,
    },
    pinSubmitText: {
      fontSize: 17,
      fontWeight: '700',
      color: '#FFFFFF',
    },
  });

export default ApprovalScreen;
