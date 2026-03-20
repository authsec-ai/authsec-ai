import React, {useState, useRef, useEffect} from 'react';
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
  ToastAndroid,
  Image,
  Animated,
  StatusBar,
} from 'react-native';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import {
  loginPrecheck,
  login,
  webauthnCallback,
  registerDeviceToken,
} from '../services/api';
import {storeToken, storeEmail, storeDeviceToken, isPinSetupCompleted} from '../services/storage';
import {useTheme} from '../context/ThemeContext';

const showToast = (message: string) => {
  if (Platform.OS === 'android') {
    ToastAndroid.show(message, ToastAndroid.SHORT);
  }
};

const LoginScreen = ({navigation}: any) => {
  const {colors, isDark} = useTheme();
  
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  
  // Animations
  const fadeAnim = useRef(new Animated.Value(0)).current;
  const slideAnim = useRef(new Animated.Value(30)).current;

  useEffect(() => {
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
  }, []);

  const styles = createStyles(colors, isDark);

  const registerDeviceForPush = async (authToken: string) => {
    try {
      if (!Device.isDevice) {
        console.log('Push notifications only work on physical devices');
        return;
      }

      const {status: existingStatus} = await Notifications.getPermissionsAsync();
      let finalStatus = existingStatus;

      if (existingStatus !== 'granted') {
        const {status} = await Notifications.requestPermissionsAsync();
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

    setLoading(true);

    try {
      console.log('Step 1: Checking tenant for email:', email);
      const precheckResponse = await loginPrecheck(email);
      const tenantDomain = precheckResponse.tenant_domain;
      const tenantId = precheckResponse.tenant_id;
      console.log('Tenant domain:', tenantDomain);
      console.log('Tenant ID:', tenantId);

      console.log('Step 2: Validating credentials...');
      await login(email, password, tenantDomain);
      console.log('Credentials validated successfully');

      console.log('Step 3: Getting JWT token via webauthn callback...');
      const tokenResponse = await webauthnCallback(email, tenantId);
      const token = tokenResponse.access_token;

      if (!token) {
        throw new Error('No access token received from webauthn callback');
      }

      console.log('Step 4: Storing token:', token.substring(0, 20) + '...');
      await storeToken(token);
      await storeEmail(email);

      console.log('Step 5: Auto-registering device for push notifications...');
      await registerDeviceForPush(token);

      // Check if PIN setup is completed
      const pinSetupDone = await isPinSetupCompleted();
      
      console.log('Step 6: Login complete! Navigating to', pinSetupDone ? 'home' : 'PIN setup');
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
        <Animated.View style={[
          styles.header,
          {opacity: fadeAnim, transform: [{translateY: slideAnim}]}
        ]}>
          <View style={styles.logoContainer}>
            <Image
              source={isDark ? require('../../appicon_dark.png') : require('../../appicon.png')}
              style={styles.logo}
              resizeMode="contain"
            />
          </View>
          <Text style={styles.title}>AuthSec Authenticator</Text>
          <Text style={styles.subtitle}>Admin Portal</Text>
        </Animated.View>

        <Animated.View style={[
          styles.card,
          {opacity: fadeAnim, transform: [{translateY: slideAnim}]}
        ]}>
          <Text style={styles.cardTitle}>Admin Login</Text>
          
          <View style={styles.inputContainer}>
            <Text style={styles.label}>Email Address</Text>
            <TextInput
              style={styles.input}
              placeholder="you@example.com"
              placeholderTextColor={colors.textSecondary}
              value={email}
              onChangeText={setEmail}
              autoCapitalize="none"
              keyboardType="email-address"
              autoCorrect={false}
              editable={!loading}
            />
          </View>

          <View style={styles.inputContainer}>
            <Text style={styles.label}>Password</Text>
            <TextInput
              style={styles.input}
              placeholder="Enter your password"
              placeholderTextColor={colors.textSecondary}
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
            disabled={loading}>
            {loading ? (
              <ActivityIndicator color={isDark ? '#000' : '#fff'} />
            ) : (
              <Text style={styles.buttonText}>Sign In</Text>
            )}
          </TouchableOpacity>
        </Animated.View>

        <TouchableOpacity
          style={styles.linkButton}
          onPress={() => navigation.navigate('EndUserLogin')}
          disabled={loading}>
          <Text style={styles.linkButtonText}>← Switch to End-User Login</Text>
        </TouchableOpacity>

        <Text style={styles.footer}>
          Secure • Private • Trusted
        </Text>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const createStyles = (colors: any, isDark: boolean) => StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: 'center',
    padding: 24,
    paddingTop: 54,
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
    shadowOffset: {width: 0, height: 8},
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
    textAlign : 'center'
  },
  subtitle: {
    fontSize: 16,
    color: colors.textSecondary,
    fontWeight: '500',
    textAlign: 'center',
  },
  card: {
    backgroundColor: colors.card,
    borderRadius: 20,
    padding: 24,
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: isDark ? 0.3 : 0.1,
    shadowRadius: 12,
    elevation: 5,
  },
  cardTitle: {
    fontSize: 24,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 24,
    textAlign: 'center',
  },
  inputContainer: {
    marginBottom: 16,
  },
  label: {
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
    padding: 16,
    fontSize: 16,
    color: colors.text,
  },
  button: {
    backgroundColor: colors.primary,
    borderRadius: 12,
    padding: 18,
    alignItems: 'center',
    marginTop: 8,
    shadowColor: colors.primary,
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 8,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: isDark ? '#000' : '#fff',
    fontSize: 18,
    fontWeight: '700',
  },
  linkButton: {
    marginTop: 8,
    alignItems: 'center',
    padding: 14,
    backgroundColor: colors.card,
    borderRadius: 12,
    marginBottom: 16,
  },
  linkButtonText: {
    color: colors.primary,
    fontSize: 15,
    fontWeight: '600',
  },
  footer: {
    textAlign: 'center',
    fontSize: 12,
    color: colors.textMuted,
    marginTop: 24,
    letterSpacing: 1,
  },
});

export default LoginScreen;
