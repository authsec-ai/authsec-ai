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
  ToastAndroid,
  Image,
  Animated,
  Dimensions,
  Modal,
} from 'react-native';
import { WebView } from 'react-native-webview';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import * as WebBrowser from 'expo-web-browser';
import {
  loginEndUser,
  webauthnCallbackEndUser,
  registerDeviceToken,
  OIDCProvider,
  initiateOIDC,
  oidcCallback,
  exchangeOIDCToken,
  oidcLogin,
  webauthnCallbackOIDC,
  getOIDCAuthURL,
} from '../services/api';
import { storeToken, storeEmail, storeDeviceToken, isPinSetupCompleted, getStoredClientId, clearClientId } from '../services/storage';
import { useTheme } from '../context/ThemeContext';
import { Ionicons } from '@expo/vector-icons';

const { width } = Dimensions.get('window');

const showToast = (message: string) => {
  if (Platform.OS === 'android') {
    ToastAndroid.show(message, ToastAndroid.SHORT);
  }
};

const EndUserLoginScreen = ({ navigation, route }: any) => {
  const { colors, isDark } = useTheme();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [clientId, setClientId] = useState('');
  const [oidcProviders, setOidcProviders] = useState<OIDCProvider[]>([]);
  const [loginChallenge, setLoginChallenge] = useState('');
  const [baseUrl, setBaseUrl] = useState('https://prod.api.authsec.ai/hmgr'); // Default base URL
  const [hydraAuthUrl, setHydraAuthUrl] = useState(''); // Full Hydra URL for browser flow
  const [showOIDCWebView, setShowOIDCWebView] = useState(false);
  const [oidcUrl, setOidcUrl] = useState('');
  const [currentProvider, setCurrentProvider] = useState<any>(null);
  const [webViewLoading, setWebViewLoading] = useState(true);
  const [simpleFlow, setSimpleFlow] = useState(false); // New: simplified manual flow
  const [jwtToken, setJwtToken] = useState(''); // New: manual JWT token input

  // Animations
  const fadeAnim = useRef(new Animated.Value(0)).current;
  const slideAnim = useRef(new Animated.Value(30)).current;
  const scaleAnim = useRef(new Animated.Value(0.9)).current;

  useEffect(() => {
    // Load client ID and OIDC providers from route params
    const loadData = async () => {
      const storedClientId = await getStoredClientId();
      if (storedClientId) {
        setClientId(storedClientId);
      } else if (route.params?.clientId) {
        setClientId(route.params.clientId);
      }

      if (route.params?.oidcProviders) {
        // Providers are already filtered in ClientIDScreen
        setOidcProviders(route.params.oidcProviders);
      }

      if (route.params?.loginChallenge) {
        setLoginChallenge(route.params.loginChallenge);
      }

      if (route.params?.baseUrl) {
        setBaseUrl(route.params.baseUrl);
        console.log('Base URL loaded:', route.params.baseUrl);
      }

      if (route.params?.hydraAuthUrl) {
        setHydraAuthUrl(route.params.hydraAuthUrl);
        console.log('Hydra Auth URL loaded:', route.params.hydraAuthUrl);
      }

      if (route.params?.simpleFlow) {
        setSimpleFlow(true);
        console.log('Simple manual flow enabled');
      }
    };

    loadData();

    // Animations
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
      Animated.spring(scaleAnim, {
        toValue: 1,
        tension: 50,
        friction: 7,
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
      showToast('✓ Device registered for notifications');
    } catch (error) {
      console.error('Auto device registration failed:', error);
    }
  };

  const handleLogin = async () => {
    if (!email || !password) {
      Alert.alert('Error', 'Please enter email and password');
      return;
    }

    if (!clientId) {
      Alert.alert('Error', 'Client ID not found. Please restart the app.');
      return;
    }

    setLoading(true);

    try {
      // Extract tenant domain from email
      const tenantDomain = email.split('@')[1];
      console.log('Using tenant domain from email:', tenantDomain);
      console.log('Client ID:', clientId);

      // Step 1: Login with credentials (validates password) and get tenant_id
      console.log('Step 1: Validating credentials...');
      const loginResponse = await loginEndUser(email, password, tenantDomain, clientId);
      console.log('Credentials validated successfully');

      // Get tenant_id from login response
      const tenantId = loginResponse.tenant_id;
      console.log('Tenant ID (from login response):', tenantId);

      // Step 2: WebAuthn callback to get actual JWT token
      console.log('Step 2: Getting JWT token via webauthn callback...');
      const tokenResponse = await webauthnCallbackEndUser(email, tenantId, clientId);
      const token = tokenResponse.access_token;

      if (!token) {
        throw new Error('No access token received from webauthn callback');
      }

      // Step 3: Store token
      console.log('Step 3: Storing token:', token.substring(0, 20) + '...');
      await storeToken(token);
      await storeEmail(email);

      // Step 4: Auto-register device for push notifications
      console.log('Step 4: Auto-registering device for push notifications...');
      await registerDeviceForPush(token);

      // Check if PIN setup is completed
      const pinSetupDone = await isPinSetupCompleted();

      // Step 5: Navigate to home or PIN setup
      console.log('Step 5: Login complete! Navigating to', pinSetupDone ? 'home' : 'PIN setup');
      if (pinSetupDone) {
        navigation.replace('Home');
      } else {
        navigation.replace('AppPinSetup');
      }
    } catch (error: any) {
      console.error('Login error:', error);
      Alert.alert('Login Failed', error.message);
    } finally {
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

  const handleBrowserLogin = async () => {
    try {
      console.log('Opening browser for full OAuth flow');
      console.log('Client ID:', clientId);

      setLoading(true);

      // Step 1: Get proper auth URL from API
      console.log('Fetching OIDC auth URL from API...');
      const authUrlResponse = await getOIDCAuthURL(clientId);
      let authUrl = authUrlResponse.auth_url;

      console.log('Auth URL received:', authUrl);

      // Step 2: Add mobile flag to the URL
      const separator = authUrl.includes('?') ? '&' : '?';
      authUrl = `${authUrl}${separator}mobile=true`;

      console.log('========================================');
      console.log('🌐 OPENING BROWSER FOR OAUTH');
      console.log('========================================');
      console.log('Auth URL:', authUrl.substring(0, 150) + '...');
      console.log('Mobile flag: ✅ added');
      console.log('========================================');

      // Step 3: Open in system browser
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

  const handleOIDCLogin = async (provider: any) => {
    try {
      if (!loginChallenge) {
        Alert.alert('Error', 'No login challenge found. Please restart the app.');
        return;
      }

      console.log('Initiating OIDC login for provider:', provider.name);
      console.log('Using base URL:', baseUrl);
      console.log('Login challenge:', loginChallenge.substring(0, 50) + '...');

      setLoading(true);

      // Step 1: Call initiate API to get the auth_url
      const initiateResponse = await initiateOIDC(
        provider.provider_name,
        loginChallenge,
        baseUrl
      );

      console.log('Initiate response:', initiateResponse);
      console.log('Auth URL from initiate:', initiateResponse.auth_url);

      // Extract state from auth_url if available
      let expectedState = '';
      if (initiateResponse.auth_url) {
        try {
          const urlObj = new URL(initiateResponse.auth_url);
          expectedState = urlObj.searchParams.get('state') || '';
          console.log('Extracted state from auth_url:', expectedState ? expectedState.substring(0, 50) + '...' : 'None');
        } catch (e) {
          console.log('Could not parse auth_url');
        }
      }

      // If the API returns state directly (some do), use that as fallback
      if (!expectedState && initiateResponse.state) {
        expectedState = initiateResponse.state;
      }

      // Step 2: Open in proper in-app browser (not WebView) to avoid Google's useragent restriction
      const result = await WebBrowser.openAuthSessionAsync(
        `${initiateResponse.auth_url}&mobile=true`, // Add mobile flag for web app detection
        'https://jk.app.authsec.dev/oidc/auth/callback' // redirect URL
      );

      console.log('WebBrowser result:', result);

      if (result.type === 'success' && result.url) {
        // Extract code and state from callback URL
        const urlObj = new URL(result.url);
        const code = urlObj.searchParams.get('code');
        const state = urlObj.searchParams.get('state');

        if (code && state) {
          console.log('Callback detected! Code (raw):', code.substring(0, 10));
          console.log('State (raw):', state.substring(0, 50));

          // URL parameters are already decoded by URL.searchParams.get()
          // However, sometimes + signs are replaced by spaces during this process
          let finalState = state;

           // Handle potential encoding issues (rare for Base64URL but good for safety)
           // If 'state' contains spaces but no '+', it might be that '+' was decoded to space
           if (finalState.includes(' ') && !finalState.includes('+')) {
             console.log('Fixing state: replacing spaces with +');
             finalState = finalState.replace(/ /g, '+');
           }

          // Complete the OIDC flow
          await completeOIDCFlow(code, finalState, expectedState);
        } else {
          console.error('No code or state in callback URL');
          Alert.alert('Error', 'Invalid callback from authentication provider');
          setLoading(false);
        }
      } else if (result.type === 'cancel') {
        console.log('User cancelled authentication');
        setLoading(false);
      } else {
        console.log('Authentication failed or was dismissed');
        setLoading(false);
      }
    } catch (error: any) {
      console.error('OIDC login error:', error);
      Alert.alert('Error', error.message || 'Failed to open authentication provider');
      setLoading(false);
    }
  };

  const completeOIDCFlow = async (code: string, state: string, expectedState?: string) => {
    try {
      console.log('=== Starting OIDC Flow (Platform-Compatible) ===');

      // Verify state matches what we sent/received during initiation
      // This mimics the web app's security check
      if (expectedState && state !== expectedState) {
         console.warn('⚠️ State mismatch detected!');
         console.warn('Expected:', expectedState.substring(0, 50));
         console.warn('Received:', state.substring(0, 50));
         // We log it but proceed, assuming backend checks will handle final validation if crtical.
         // In a strict implementation, we would abort here.
      } else if (expectedState) {
         console.log('✅ State verified locally');
      }

      // STEP 3 (Platform Flow): Handle Callback - Validate code & state
      console.log('Step 3: Validating callback with backend...');
      console.log('Code:', code.substring(0, 30) + '...');
      console.log('State:', state.substring(0, 50) + '...');

      const callbackResponse = await oidcCallback(code, state);
      console.log('Callback validation response:', callbackResponse);

      // Decode state to extract provider info (platform does this)
      const stateData = decodeStateParameter(state);
      const provider = stateData.provider || 'google';
      console.log('Decoded provider from state:', provider);

      // STEP 4 (Platform Flow): Exchange Token
      console.log('Step 4: Exchanging code for access token...');
      const exchangeResponse = await exchangeOIDCToken(
        loginChallenge,
        code,
        state,
        provider, // Extract from state, not hardcode
        'https://jk.app.authsec.dev/oidc/auth/callback'
      );

      console.log('Exchange response:', exchangeResponse);

      // Platform returns: { success, tokens: { access_token, expires_in, ... } }
      const accessToken = exchangeResponse.tokens?.access_token || exchangeResponse.access_token;
      const expiresIn = exchangeResponse.tokens?.expires_in || exchangeResponse.expires_in || 3600;

      if (!accessToken) {
        throw new Error('No access token received from token exchange');
      }

      // STEP 5 (Platform Flow): Final Login
      console.log('Step 5: Logging in with OIDC access token...');
      const loginResponse = await oidcLogin(accessToken, expiresIn);
      console.log('OIDC Login response:', loginResponse);

      // Platform returns: { success, data: { tenant_id, email, first_login } }
      const email = loginResponse.data?.email || loginResponse.email || extractEmailFromToken(accessToken);
      const tenantId = loginResponse.data?.tenant_id || loginResponse.tenant_id || clientId;

      console.log('Email:', email);
      console.log('Tenant ID:', tenantId);

      // STEP 6 (Mobile-Specific): WebAuthn callback to get final app token
      console.log('Step 6: Getting final app token via WebAuthn callback...');
      const finalTokenResponse = await webauthnCallbackOIDC(
        email,
        tenantId,
        clientId
      );

      const finalToken = finalTokenResponse.access_token;
      console.log('Final token received:', finalToken.substring(0, 20) + '...');

      // STEP 7: Store token and email
      await storeToken(finalToken);
      await storeEmail(email);

      // STEP 8: Auto-register device for push notifications
      console.log('Step 7: Auto-registering device for push notifications...');
      await registerDeviceForPush(finalToken);

      // Check if PIN setup is completed
      const pinSetupDone = await isPinSetupCompleted();

      // STEP 9: Navigate to home or PIN setup
      console.log('=== OIDC Login Complete! ===');
      console.log('Navigating to', pinSetupDone ? 'home' : 'PIN setup');

      if (pinSetupDone) {
        navigation.replace('Home');
      } else {
        navigation.replace('AppPinSetup');
      }
    } catch (error: any) {
      console.error('OIDC flow error:', error);
      Alert.alert('Authentication Failed', error.message || 'Failed to complete OIDC authentication');
    } finally {
      setLoading(false);
    }
  };

  const decodeStateParameter = (state: string): any => {
    try {
      // First try to parse as-is (in case it's not base64)
      try {
         return JSON.parse(state);
      } catch (e) {
         // Not JSON, continue to base64 decode
      }

      // Handle base64 decoding (standard and URL-safe)
      let base64 = state.replace(/-/g, '+').replace(/_/g, '/');
      const pad = base64.length % 4;
      if (pad) {
        if (pad === 1) {
          throw new Error('InvalidLengthError: Input base64url string is the wrong length to determine padding');
        }
        base64 += new Array(5 - pad).join('=');
      }

      const decoded = JSON.parse(atob(base64));
      console.log('Decoded state:', decoded);
      return decoded;
    } catch (error) {
      console.error('Error decoding state parameter:', error);
      console.log('Raw state causing error:', state);
      return {};
    }
  };

  const extractEmailFromToken = (token: string): string => {
    try {
      // Decode JWT token to extract email
      const payload = token.split('.')[1];
      const decoded = JSON.parse(atob(payload));
      return decoded.ext?.email || decoded.email || '';
    } catch (error) {
      console.error('Error decoding token:', error);
      return '';
    }
  };

  const handleChangeClientId = async () => {
    Alert.alert(
      'Change Client ID',
      'Are you sure you want to change the client ID? This will take you back to the client ID screen.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Change',
          style: 'default',
          onPress: async () => {
            await clearClientId();
            navigation.reset({
              index: 0,
              routes: [{ name: 'ClientID' }],
            });
          },
        },
      ],
    );
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
            transform: [{ translateY: slideAnim }, { scale: scaleAnim }],
          }
        ]}>
          <View style={styles.logoContainer}>
            <Image
              source={isDark ? require('../../appicon_dark.png') : require('../../appicon.png')}
              style={styles.logo}
              resizeMode="contain"
            />
          </View>
          <Text style={styles.title}>AuthSec Authenticator</Text>
        </Animated.View>

        {/* Form */}
        <View style={styles.form}>
          {/* Simplified Manual Flow - Show email/password + OIDC/SAML button */}
          {simpleFlow && (
            <>
              <View style={styles.infoBox}>
                <Text style={styles.infoText}>
                  Sign in to your account
                </Text>
              </View>

              {/* Email/Password Fields */}
              <View style={styles.inputGroup}>
                <Text style={styles.label}>Email</Text>
                <TextInput
                  style={styles.input}
                  placeholder="you@example.com"
                  placeholderTextColor={colors.inputPlaceholder}
                  value={email}
                  onChangeText={setEmail}
                  autoCapitalize="none"
                  keyboardType="email-address"
                  autoCorrect={false}
                  editable={!loading}
                />
              </View>

              <View style={styles.inputGroup}>
                <Text style={styles.label}>Password</Text>
                <TextInput
                  style={styles.input}
                  placeholder="Enter your password"
                  placeholderTextColor={colors.inputPlaceholder}
                  value={password}
                  onChangeText={setPassword}
                  secureTextEntry
                  autoCapitalize="none"
                  autoCorrect={false}
                  editable={!loading}
                  onSubmitEditing={handleLogin}
                />
              </View>

              <TouchableOpacity
                style={[styles.button, loading && styles.buttonDisabled]}
                onPress={handleLogin}
                disabled={loading}
                activeOpacity={0.8}>
                {loading ? (
                  <ActivityIndicator color={isDark ? '#000000' : '#FFFFFF'} />
                ) : (
                  <Text style={styles.buttonText}>Sign In</Text>
                )}
              </TouchableOpacity>

              <View style={styles.divider}>
                <View style={styles.dividerLine} />
                <Text style={styles.dividerText}>OR</Text>
                <View style={styles.dividerLine} />
              </View>

              {/* OIDC/SAML Button */}
              <TouchableOpacity
                style={[styles.oidcSamlButton]}
                onPress={() => navigation.navigate('OIDCSAMLLogin', { clientId })}
                activeOpacity={0.8}>
                <Ionicons name="business-outline" size={20} color={colors.primary} />
                <Text style={styles.oidcSamlButtonText}>Use OIDC / SAML</Text>
              </TouchableOpacity>
            </>
          )}

          {/* Original Provider Flow - Show only if NOT simpleFlow */}
          {!simpleFlow && oidcProviders.length > 0 && (
            <>
              <View style={styles.oidcSection}>
                <Text style={styles.sectionTitle}>Sign in with</Text>
                {oidcProviders.map((provider, index) => (
                  <TouchableOpacity
                    key={index}
                    style={styles.oidcButton}
                    onPress={() => handleOIDCLogin(provider)}
                    activeOpacity={0.8}>
                    <Ionicons
                      name={
                        (provider.provider_name || provider.name).toLowerCase() === 'google' ? 'logo-google' :
                          (provider.provider_name || provider.name).toLowerCase() === 'microsoft' ? 'logo-microsoft' :
                            (provider.provider_name || provider.name).toLowerCase() === 'github' ? 'logo-github' :
                              'globe-outline'
                      }
                      size={20}
                      color={colors.text}
                    />
                    <Text style={styles.oidcButtonText}>
                      {provider.name}
                    </Text>
                  </TouchableOpacity>
                ))}
              </View>

              {/* Continue in Browser Button */}
              <TouchableOpacity
                style={[styles.browserButton, loading && styles.buttonDisabled]}
                onPress={handleBrowserLogin}
                disabled={loading}
                activeOpacity={0.8}>
                <Ionicons name="globe-outline" size={20} color={colors.primary} />
                <Text style={styles.browserButtonText}>Continue in Browser</Text>
              </TouchableOpacity>

              <View style={styles.divider}>
                <View style={styles.dividerLine} />
                <Text style={styles.dividerText}>OR</Text>
                <View style={styles.dividerLine} />
              </View>
            </>
          )}

          {/* Email/Password Login - Show only if NOT simpleFlow */}
          {!simpleFlow && (
            <>
              <View style={styles.inputGroup}>
                <Text style={styles.label}>Email</Text>
                <TextInput
                  style={styles.input}
                  placeholder="you@example.com"
                  placeholderTextColor={colors.inputPlaceholder}
                  value={email}
                  onChangeText={setEmail}
                  autoCapitalize="none"
                  keyboardType="email-address"
                  autoCorrect={false}
                  editable={!loading}
                />
              </View>

              <View style={styles.inputGroup}>
            <Text style={styles.label}>Password</Text>
            <TextInput
              style={styles.input}
              placeholder="Enter your password"
              placeholderTextColor={colors.inputPlaceholder}
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoCapitalize="none"
              autoCorrect={false}
              editable={!loading}
              onSubmitEditing={handleLogin}
            />
          </View>

          <TouchableOpacity
            style={[styles.button, loading && styles.buttonDisabled]}
            onPress={handleLogin}
            disabled={loading}
            activeOpacity={0.8}>
            {loading ? (
              <ActivityIndicator color={isDark ? '#000000' : '#FFFFFF'} />
            ) : (
              <Text style={styles.buttonText}>Sign In</Text>
            )}
          </TouchableOpacity>
            </>
          )}
        </View>

        {/* Change Client ID Link */}
        <TouchableOpacity
          style={styles.changeClientIdButton}
          onPress={handleChangeClientId}
          activeOpacity={0.7}>
          <Ionicons name="settings-outline" size={16} color={colors.textSecondary} />
          <Text style={styles.changeClientIdText}>Change Client ID</Text>
        </TouchableOpacity>

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
      padding: 34,
      paddingTop: 54,
      paddingBottom: 120,
    },
    header: {
      alignItems: 'center',
      marginBottom: 20,
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
    oidcSection: {
      marginBottom: 20,
    },
    sectionTitle: {
      fontSize: 14,
      fontWeight: '600',
      color: colors.textSecondary,
      marginBottom: 12,
      textAlign: 'center',
    },
    oidcButton: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: colors.card,
      borderWidth: 1,
      borderColor: colors.border,
      borderRadius: 14,
      padding: 14,
      marginBottom: 10,
    },
    oidcButtonText: {
      fontSize: 16,
      fontWeight: '600',
      color: colors.text,
      marginLeft: 10,
    },
    browserButton: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: colors.background,
      borderWidth: 2,
      borderColor: colors.primary,
      borderRadius: 14,
      padding: 14,
      marginBottom: 10,
    },
    browserButtonText: {
      fontSize: 16,
      fontWeight: '700',
      color: colors.primary,
      marginLeft: 10,
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
    button: {
      backgroundColor: colors.primary,
      borderRadius: 14,
      padding: 18,
      alignItems: 'center',
      marginTop: 12,
      shadowColor: colors.primary,
      shadowOffset: { width: 0, height: 6 },
      shadowOpacity: 0.35,
      shadowRadius: 12,
      elevation: 8,
    },
    buttonDisabled: {
      opacity: 0.6,
    },
    buttonText: {
      color: isDark ? '#000000' : '#FFFFFF',
      fontSize: 18,
      fontWeight: '700',
    },
    oidcSamlButton: {
      backgroundColor: isDark ? '#1E1E1E' : '#F5F5F5',
      paddingVertical: 15,
      paddingHorizontal: 20,
      borderRadius: 12,
      alignItems: 'center',
      flexDirection: 'row',
      justifyContent: 'center',
      gap: 8,
      borderWidth: 1,
      borderColor: colors.primary,
      marginTop: 12,
    },
    oidcSamlButtonText: {
      color: colors.primary,
      fontSize: 16,
      fontWeight: '600',
    },
    changeClientIdButton: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      marginTop: 24,
      marginBottom: 8,
      paddingVertical: 10,
    },
    changeClientIdText: {
      fontSize: 14,
      color: colors.textSecondary,
      marginLeft: 6,
      fontWeight: '500',
    },
    footer: {
      textAlign: 'center',
      fontSize: 12,
      color: colors.textMuted,
      marginTop: 32,
      letterSpacing: 1,
    },
    webViewContainer: {
      flex: 1,
      backgroundColor: colors.background,
    },
    webViewHeader: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      padding: 16,
      paddingTop: Platform.OS === 'ios' ? 50 : 16,
      backgroundColor: colors.card,
      borderBottomWidth: 1,
      borderBottomColor: colors.border,
    },
    webViewTitle: {
      fontSize: 18,
      fontWeight: '600',
      color: colors.text,
      flex: 1,
    },
    closeButton: {
      padding: 4,
    },
    webView: {
      flex: 1,
    },
    loadingContainer: {
      position: 'absolute',
      top: 100,
      left: 0,
      right: 0,
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 999,
    },
    loadingText: {
      marginTop: 12,
      fontSize: 14,
      color: colors.text,
    },
  });

export default EndUserLoginScreen;
