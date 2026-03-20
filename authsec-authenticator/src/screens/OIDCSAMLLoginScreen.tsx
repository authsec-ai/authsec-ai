import React, { useState, useRef, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StatusBar,
  Image,
  Animated,
} from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import { getOIDCAuthURL, registerDeviceToken } from '../services/api';
import { storeToken, storeEmail, storeDeviceToken, isPinSetupCompleted } from '../services/storage';
import { useTheme } from '../context/ThemeContext';
import { Ionicons } from '@expo/vector-icons';

const OIDCSAMLLoginScreen = ({ navigation, route }: any) => {
  const { colors, isDark } = useTheme();
  const [loading, setLoading] = useState(false);
  const [jwtToken, setJwtToken] = useState('');
  const [clientId, setClientId] = useState('');

  // Animations
  const fadeAnim = useRef(new Animated.Value(0)).current;
  const slideAnim = useRef(new Animated.Value(30)).current;

  useEffect(() => {
    if (route.params?.clientId) {
      setClientId(route.params.clientId);
    }

    Animated.parallel([
      Animated.timing(fadeAnim, {
        toValue: 1,
        duration: 600,
        useNativeDriver: true,
      }),
      Animated.timing(slideAnim, {
        toValue: 0,
        duration: 600,
        useNativeDriver: true,
      }),
    ]).start();
  }, [route.params]);

  const registerDeviceForPush = async (authToken: string) => {
    try {
      if (!Device.isDevice) {
        console.log('Push notifications only work on physical devices');
        return;
      }

      const { status: existingStatus } = await Notifications.getPermissionsAsync();
      let finalStatus = existingStatus;

      if (existingStatus !== 'granted') {
        const { status } = await Notifications.requestPermissionsAsync();
        finalStatus = status;
      }

      if (finalStatus !== 'granted') {
        console.log('Push notification permission not granted');
        return;
      }

      const token = (await Notifications.getExpoPushTokenAsync({
        projectId: 'YOUR_EAS_PROJECT_ID'
      })).data;
      console.log('Expo Push Token obtained:', token.substring(0, 30) + '...');
      await storeDeviceToken(token);

      await registerDeviceToken(token, authToken);
      console.log('Device registered with backend');
    } catch (error) {
      console.error('Auto device registration failed:', error);
    }
  };

  const handleBrowserLogin = async () => {
    try {
      console.log('Opening browser for full OAuth flow');
      console.log('Client ID:', clientId);

      setLoading(true);

      // Get proper auth URL from API
      console.log('Fetching OIDC auth URL from API...');
      const authUrlResponse = await getOIDCAuthURL(clientId);
      let authUrl = authUrlResponse.auth_url;

      console.log('Auth URL received:', authUrl);

      // Add mobile flag to the URL
      const separator = authUrl.includes('?') ? '&' : '?';
      authUrl = `${authUrl}${separator}mobile=true`;

      console.log('========================================');
      console.log('🌐 OPENING BROWSER FOR OAUTH');
      console.log('========================================');
      console.log('Auth URL:', authUrl.substring(0, 150) + '...');
      console.log('Mobile flag: ✅ added');
      console.log('========================================');

      // Open in system browser
      const result = await WebBrowser.openBrowserAsync(authUrl);
      
      console.log('Browser result:', result);
      
      setLoading(false);
      
      Alert.alert(
        'Continue in Browser',
        'Complete authentication in your browser. After successful login, copy the JWT token from the web page and paste it here.',
        [{ text: 'OK' }]
      );

    } catch (error: any) {
      console.error('Browser login error:', error);
      Alert.alert('Error', error.message || 'Failed to open browser');
      setLoading(false);
    }
  };

  const handleManualTokenLogin = async () => {
    if (!jwtToken.trim()) {
      Alert.alert('Error', 'Please enter a JWT token');
      return;
    }

    setLoading(true);

    try {
      console.log('Processing manual JWT token...');
      
      // Decode JWT to extract user info
      const decoded = decodeJWT(jwtToken.trim());
      console.log('Decoded JWT:', decoded);

      const email = decoded.email_id || decoded.email || '';
      const tenantId = decoded.tenant_id || decoded.project_id || clientId;

      if (!email) {
        throw new Error('Invalid token: email not found');
      }

      console.log('Email from token:', email);
      console.log('Tenant ID:', tenantId);

      // Store token and email
      await storeToken(jwtToken.trim());
      await storeEmail(email);

      // Register device for push notifications
      console.log('Registering device for push notifications...');
      await registerDeviceForPush(jwtToken.trim());

      // Check if PIN setup is completed
      const pinSetupDone = await isPinSetupCompleted();

      console.log('=== Manual Token Login Complete! ===');
      console.log('Navigating to', pinSetupDone ? 'home' : 'PIN setup');

      if (pinSetupDone) {
        navigation.replace('Home');
      } else {
        navigation.replace('AppPinSetup');
      }
    } catch (error: any) {
      console.error('Manual token login error:', error);
      Alert.alert('Login Failed', error.message || 'Invalid JWT token');
    } finally {
      setLoading(false);
    }
  };

  const decodeJWT = (token: string): any => {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) {
        throw new Error('Invalid JWT format');
      }
      const payload = parts[1];
      const decoded = JSON.parse(atob(payload));
      return decoded;
    } catch (error) {
      throw new Error('Failed to decode JWT token');
    }
  };

  const styles = createStyles(colors, isDark);

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}>
      <StatusBar
        barStyle={isDark ? 'light-content' : 'dark-content'}
        backgroundColor={colors.background}
      />
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}>

        {/* Header with Logo */}
        <Animated.View style={[
          styles.header,
          {
            opacity: fadeAnim,
            transform: [{ translateY: slideAnim }],
          }
        ]}>
          <TouchableOpacity
            style={styles.backButton}
            onPress={() => navigation.goBack()}>
            <Ionicons name="arrow-back" size={24} color={colors.text} />
          </TouchableOpacity>
          <View style={styles.logoContainer}>
            <Image
              source={isDark ? require('../../appicon_dark.png') : require('../../appicon.png')}
              style={styles.logo}
              resizeMode="contain"
            />
          </View>
          <Text style={styles.title}>OIDC / SAML</Text>
          <Text style={styles.subtitle}>Enterprise Authentication</Text>
        </Animated.View>

        {/* Form */}
        <View style={styles.form}>
          <View style={styles.infoBox}>
            <Text style={styles.infoText}>
              Use OIDC/SAML Authentication
            </Text>
          </View>

          {/* Continue in Browser Button */}
          <TouchableOpacity
            style={[styles.primaryButton, loading && styles.buttonDisabled]}
            onPress={handleBrowserLogin}
            disabled={loading}
            activeOpacity={0.8}>
            <Ionicons name="globe-outline" size={24} color={isDark ? '#000000' : '#FFFFFF'} />
            <Text style={styles.primaryButtonText}>Continue in Browser</Text>
          </TouchableOpacity>

          <View style={styles.divider}>
            <View style={styles.dividerLine} />
            <Text style={styles.dividerText}>OR</Text>
            <View style={styles.dividerLine} />
          </View>

          {/* Manual JWT Token Input */}
          <View style={styles.inputGroup}>
            <Text style={styles.label}>Paste JWT Token</Text>
            <TextInput
              style={[styles.input, styles.tokenInput]}
              placeholder="Paste your JWT token here..."
              placeholderTextColor={colors.inputPlaceholder}
              value={jwtToken}
              onChangeText={setJwtToken}
              autoCapitalize="none"
              autoCorrect={false}
              editable={!loading}
              multiline={true}
              numberOfLines={4}
              textAlignVertical="top"
            />
            <Text style={styles.helpText}>
              After authenticating in the browser, copy the token from the web page and paste it here.
            </Text>
          </View>

          <TouchableOpacity
            style={[styles.button, loading && styles.buttonDisabled]}
            onPress={handleManualTokenLogin}
            disabled={loading || !jwtToken.trim()}
            activeOpacity={0.8}>
            {loading ? (
              <ActivityIndicator color={isDark ? '#000000' : '#FFFFFF'} />
            ) : (
              <Text style={styles.buttonText}>Login with Token</Text>
            )}
          </TouchableOpacity>
        </View>

        {/* Footer */}
        <Text style={styles.footer}>
          Secure • Private • Trusted
        </Text>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const createStyles = (colors: any, isDark: boolean) =>
  StyleSheet.create({
    container: {
      flex: 1,
      backgroundColor: colors.background,
    },
    scrollContent: {
      flexGrow: 1,
      justifyContent: 'center',
      padding: 34,
      paddingTop: 54,
    },
    header: {
      alignItems: 'center',
      marginBottom: 40,
      position: 'relative',
    },
    backButton: {
      position: 'absolute',
      left: 0,
      top: 0,
      padding: 8,
      zIndex: 10,
    },
    logoContainer: {
      width: 80,
      height: 80,
      borderRadius: 28,
      backgroundColor: colors.card,
      justifyContent: 'center',
      alignItems: 'center',
      marginBottom: 4,
      shadowColor: colors.primary,
      shadowOffset: { width: 0, height: 8 },
      shadowOpacity: 0.2,
      shadowRadius: 16,
      elevation: 8,
      overflow: 'hidden',
    },
    logo: {
      width: 70,
      height: 70,
    },
    title: {
      fontSize: 40,
      fontWeight: '800',
      color: colors.primary,
      marginBottom: 10,
      letterSpacing: -0.5,
      textAlign: 'center',
    },
    subtitle: {
      fontSize: 16,
      color: colors.textSecondary,
      fontWeight: '500',
      textAlign: 'center',
    },
    form: {
      width: '100%',
    },
    infoBox: {
      backgroundColor: colors.card,
      borderRadius: 12,
      padding: 16,
      marginBottom: 20,
      borderLeftWidth: 4,
      borderLeftColor: colors.primary,
    },
    infoText: {
      fontSize: 14,
      color: colors.text,
      fontWeight: '600',
      textAlign: 'center',
    },
    primaryButton: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: colors.primary,
      borderRadius: 14,
      padding: 18,
      marginBottom: 20,
      shadowColor: colors.primary,
      shadowOffset: { width: 0, height: 4 },
      shadowOpacity: 0.3,
      shadowRadius: 8,
      elevation: 6,
    },
    primaryButtonText: {
      fontSize: 18,
      fontWeight: '700',
      color: isDark ? '#000000' : '#FFFFFF',
      marginLeft: 12,
    },
    divider: {
      flexDirection: 'row',
      alignItems: 'center',
      marginVertical: 24,
    },
    dividerLine: {
      flex: 1,
      height: 1,
      backgroundColor: colors.border,
    },
    dividerText: {
      marginHorizontal: 16,
      fontSize: 12,
      color: colors.textSecondary,
      fontWeight: '500',
    },
    inputGroup: {
      marginBottom: 20,
    },
    label: {
      fontSize: 14,
      fontWeight: '600',
      color: colors.text,
      marginBottom: 8,
      marginLeft: 4,
    },
    input: {
      backgroundColor: colors.card,
      borderWidth: 1,
      borderColor: colors.border,
      borderRadius: 14,
      padding: 16,
      fontSize: 16,
      color: colors.text,
    },
    tokenInput: {
      minHeight: 100,
      paddingTop: 12,
    },
    helpText: {
      fontSize: 12,
      color: colors.textSecondary,
      marginTop: 8,
      lineHeight: 16,
    },
    button: {
      backgroundColor: colors.primary,
      borderRadius: 14,
      padding: 18,
      alignItems: 'center',
      shadowColor: colors.primary,
      shadowOffset: { width: 0, height: 4 },
      shadowOpacity: 0.3,
      shadowRadius: 8,
      elevation: 6,
    },
    buttonDisabled: {
      opacity: 0.6,
    },
    buttonText: {
      color: isDark ? '#000000' : '#FFFFFF',
      fontSize: 18,
      fontWeight: '700',
    },
    footer: {
      textAlign: 'center',
      color: colors.textSecondary,
      fontSize: 14,
      marginTop: 40,
      fontWeight: '500',
    },
  });

export default OIDCSAMLLoginScreen;
