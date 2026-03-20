import React, { useEffect, useState } from 'react';
import { NavigationContainer, DefaultTheme, DarkTheme, Theme } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import * as Linking from 'expo-linking';
import { Alert, Platform, View, TouchableOpacity } from 'react-native';

// Theme
import { ThemeProvider, useTheme } from './src/context/ThemeContext';
import { Ionicons } from '@expo/vector-icons';

// Components
import AuthRequestPopup from './src/components/AuthRequestPopup';

// Screens
import LoginScreen from './src/screens/LoginScreen';
import ClientIDScreen from './src/screens/ClientIDScreen';
import EndUserLoginScreen from './src/screens/EndUserLoginScreen';
import OIDCSAMLLoginScreen from './src/screens/OIDCSAMLLoginScreen';
import AppPinSetupScreen from './src/screens/AppPinSetupScreen';
import AppPinEntryScreen from './src/screens/AppPinEntryScreen';
import ApprovalScreen from './src/screens/ApprovalScreen';
import DeviceRegistrationScreen from './src/screens/DeviceRegistrationScreen';
import TOTPSetupScreen from './src/screens/TOTPSetupScreen';
import AuthenticatorScreen from './src/screens/AuthenticatorScreen';
import NotificationsScreen from './src/screens/NotificationsScreen';

// Navigation
import MainTabs from './src/navigation/MainTabs';

// Services
import { registerDeviceToken, respondToCIBA } from './src/services/api';
import { getStoredToken, storeDeviceToken, isPinSetupCompleted, getStoredClientId, addActivityLog, getAppLockEnabled, addNotification } from './src/services/storage';

const Stack = createNativeStackNavigator();

// Configure notification behavior - show alerts even when app is in foreground
Notifications.setNotificationHandler({
  handleNotification: async (notification) => {
    console.log('Handling notification:', notification);
    return {
      shouldShowAlert: true,
      shouldPlaySound: true,
      shouldSetBadge: false,
      shouldShowBanner: true,
      shouldShowList: true,
    };
  },
});

// Configure notification categories with action buttons
Notifications.setNotificationCategoryAsync('auth_request', [
  {
    identifier: 'approve',
    buttonTitle: '✓ Approve',
    options: {
      opensAppToForeground: true, // Opens app for biometric verification
      isAuthenticationRequired: true, // Requires device unlock
    },
  },
  {
    identifier: 'deny',
    buttonTitle: '✗ Deny',
    options: {
      isDestructive: true,
      opensAppToForeground: false, // Can deny without opening app
    },
  },
]);

function App({ navigation }: any): React.JSX.Element {
  const [initialRoute, setInitialRoute] = useState<string>('ClientID');
  const [isReady, setIsReady] = useState(false);
  const [needsPinVerification, setNeedsPinVerification] = useState(false);
  const navigationRef = React.useRef<any>(null);
  const [pendingAuthRequest, setPendingAuthRequest] = useState<string | null>(null);
  const [showAuthPopup, setShowAuthPopup] = useState(false);
  const [currentRouteName, setCurrentRouteName] = useState<string | undefined>();
  const { colors, isDark } = useTheme();

  // Custom navigation theme
  const navigationTheme: Theme = {
    dark: isDark,
    colors: {
      primary: colors.primary,
      background: colors.background,
      card: colors.card,
      text: colors.text,
      border: colors.border,
      notification: colors.error,
    },
    fonts: DefaultTheme.fonts,
  };

  useEffect(() => {
    // 1. Initial check: Was app opened by a notification?
    Notifications.getLastNotificationResponseAsync().then(response => {
      if (response) {
        console.log('App opened via notification (cold start):', response);
        const { actionIdentifier } = response;
        const { title, body, data } = response.notification.request.content;

        // Store notification for the Notifications screen
        addNotification({
          title: title || 'Notification',
          body: body || '',
          data,
        });

        // Handle action button press from notification
        if (actionIdentifier === 'approve' && (data as any)?.auth_req_id) {
          handleNotificationAction('approve', String((data as any).auth_req_id));
        } else if (actionIdentifier === 'deny' && (data as any)?.auth_req_id) {
          handleNotificationAction('deny', String((data as any).auth_req_id));
        } else if ((data as any)?.auth_req_id) {
          // Just tapped the notification, not an action button
          console.log('Notification tapped, queuing request:', (data as any).auth_req_id);
          setPendingAuthRequest(String((data as any).auth_req_id));
        }
      }
    });

    setupPushNotifications();
    checkAuthStatus();

    // 2. Handle notifications received while app is running (foreground)
    const notificationListener = Notifications.addNotificationReceivedListener(notification => {
      console.log('Notification received (foreground):', notification);
      const { title, body, data } = notification.request.content;

      // Store notification for the Notifications screen
      addNotification({
        title: title || 'Notification',
        body: body || '',
        data,
      });

      // Show approval popup immediately when notification arrives on ANY screen
      if ((data as any)?.type === 'auth_request' && (data as any)?.auth_req_id) {
        setPendingAuthRequest(String((data as any).auth_req_id));
        setShowAuthPopup(true);
      }
    });

    // 3. Handle notification taps while app is in background/memory
    const responseListener = Notifications.addNotificationResponseReceivedListener(response => {
      console.log('Notification tapped (background):', response);
      const { actionIdentifier } = response;
      const { title, body, data } = response.notification.request.content;

      // Store notification for the Notifications screen
      addNotification({
        title: title || 'Notification',
        body: body || '',
        data,
      });

      if (actionIdentifier === 'approve' && (data as any)?.auth_req_id) {
        handleNotificationAction('approve', String((data as any).auth_req_id));
      } else if (actionIdentifier === 'deny' && (data as any)?.auth_req_id) {
        handleNotificationAction('deny', String((data as any).auth_req_id));
      } else if ((data as any)?.auth_req_id) {
        // User tapped the notification itself (not action button)
        setPendingAuthRequest(String((data as any).auth_req_id));
      }
    });

    // 4. Handle deep links for OAuth callback from web app
    const linkingListener = Linking.addEventListener('url', ({ url }) => {
      console.log('Deep link received:', url);
      
      if (url.startsWith('authsec://callback')) {
        try {
          const urlObj = new URL(url);
          const code = urlObj.searchParams.get('code');
          const state = urlObj.searchParams.get('state');
          const token = urlObj.searchParams.get('token');
          
          if (token) {
            // Web app completed full flow and sent token
            console.log('Received token from web app');
            storeToken(token).then(() => {
              Alert.alert('Success', 'Login completed successfully!');
              navigationRef.current?.navigate('Home');
            }).catch(err => {
              console.error('Failed to store token:', err);
              Alert.alert('Error', 'Failed to complete login');
            });
          } else if (code && state) {
            // Web app sent code for mobile to exchange
            console.log('Received code from web app:', code.substring(0, 10) + '...');
            // Navigate to EndUserLogin which will handle the code exchange
            navigationRef.current?.navigate('EndUserLogin', { oauthCode: code, oauthState: state });
          }
        } catch (err) {
          console.error('Failed to parse deep link:', err);
        }
      }
    });

    return () => {
      notificationListener.remove();
      responseListener.remove();
      linkingListener.remove();
    };
  }, []);

  const handleNotificationAction = async (action: 'approve' | 'deny', authReqId: string) => {
    console.log(`Handling ${action} action for request ${authReqId}`);

    try {
      const authToken = await getStoredToken();
      if (!authToken) {
        Alert.alert('Error', 'Not logged in. Please login first.');
        return;
      }

      // Import the necessary functions
      const { respondToCIBA } = require('./src/services/api');
      const { authenticateWithBiometric } = require('./src/services/biometric');

      if (action === 'approve') {
        // Require biometric authentication for approval
        const result = await authenticateWithBiometric('Approve authentication request');

        if (!result) {
          Alert.alert('Authentication Failed', 'Biometric authentication required to approve.');
          return;
        }

        await respondToCIBA(authReqId, true, true, authToken);

        // Log the approval activity
        await addActivityLog({
          type: 'auth_approved',
          title: 'Sign-in Approved',
          description: `Approved authentication request via notification action`,
          metadata: { authReqId },
        });

        Alert.alert('✓ Approved', 'Authentication request has been approved.');
      } else {
        // Deny doesn't require biometric - works from notification bar
        await respondToCIBA(authReqId, false, false, authToken);

        // Log the denial activity
        await addActivityLog({
          type: 'auth_denied',
          title: 'Sign-in Denied',
          description: `Denied authentication request via notification action`,
          metadata: { authReqId },
        });

        // Show local notification that request was denied
        await Notifications.scheduleNotificationAsync({
          content: {
            title: '✗ Request Denied',
            body: 'Authentication request has been denied.',
          },
          trigger: null,
        });
      }
    } catch (error: any) {
      console.error('Error handling notification action:', error);
      Alert.alert('Error', error.message || 'Failed to process request');
    }
  };


  const checkAuthStatus = async () => {
    const token = await getStoredToken();
    const clientId = await getStoredClientId();

    // If no client ID, always start with ClientID screen
    if (!clientId) {
      setInitialRoute('ClientID');
    } else if (token) {
      // Check if PIN setup is completed AND app lock is enabled
      const pinSetupDone = await isPinSetupCompleted();
      const appLockEnabled = await getAppLockEnabled();

      if (pinSetupDone && appLockEnabled) {
        setNeedsPinVerification(true);
        setInitialRoute('AppPinEntry');
      } else {
        setInitialRoute('Home');
      }
    } else {
      // Has client ID but no token, go to login
      setInitialRoute('EndUserLogin');
    }
    setIsReady(true);
  };

  const setupPushNotifications = async () => {
    if (Platform.OS === 'android') {
      // Create high-priority channel for authentication requests
      await Notifications.setNotificationChannelAsync('auth_request', {
        name: 'Authentication Requests',
        importance: Notifications.AndroidImportance.MAX, // Maximum importance
        vibrationPattern: [0, 250, 250, 250],
        lightColor: '#2196F3',
        showBadge: true,
        sound: 'default',
        enableVibrate: true,
        enableLights: true,
        bypassDnd: true, // Bypass Do Not Disturb mode
      });
    }

    if (!Device.isDevice) {
      console.log('Push notifications only work on physical devices');
      return;
    }

    // Request permission
    const { status: existingStatus } = await Notifications.getPermissionsAsync();
    let finalStatus = existingStatus;

    if (existingStatus !== 'granted') {
      const { status } = await Notifications.requestPermissionsAsync();
      finalStatus = status;
    }

    if (finalStatus !== 'granted') {
      Alert.alert('Permission Required', 'Please enable push notifications');
      return;
    }

    // Get Expo Push token for Expo Push Service
    let token;
    try {
      token = (await Notifications.getExpoPushTokenAsync({
        projectId: 'YOUR_EAS_PROJECT_ID'
      })).data;
      console.log('Expo Push Token:', token);
      await storeDeviceToken(token);
    } catch (error) {
      console.error('Failed to get Expo push token:', error);
      return;
    }

    // Register with backend if user is logged in
    const authToken = await getStoredToken();
    if (authToken && token) {
      try {
        await registerDeviceToken(token, authToken);
        console.log('Device registered with backend automatically');
      } catch (error) {
        console.error('Failed to auto-register device:', error);
      }
    }
  };

  // Watch for pending auth requests and show popup immediately on any screen
  useEffect(() => {
    if (pendingAuthRequest && !showAuthPopup) {
      // Small delay to ensure smooth transition
      const timer = setTimeout(() => {
        setShowAuthPopup(true);
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [pendingAuthRequest, showAuthPopup]);

  const handlePushNotification = (data: any) => {
    if (data?.type === 'auth_request' && data?.auth_req_id) {
      setPendingAuthRequest(String(data.auth_req_id));
    }
  };

  if (!isReady) {
    return <View style={{ flex: 1 }} />;
  }

  return (
    <NavigationContainer
      ref={navigationRef}
      theme={navigationTheme}
      onReady={() => {
        setIsReady(true);
        setCurrentRouteName(navigationRef.current?.getCurrentRoute()?.name);
      }}
      onStateChange={() => {
        setCurrentRouteName(navigationRef.current?.getCurrentRoute()?.name);
      }}
    >
      <Stack.Navigator
        initialRouteName={initialRoute}
        screenOptions={{
          headerStyle: {
            backgroundColor: colors.card,
          },
          headerTintColor: colors.text,
          headerTitleStyle: {
            fontWeight: '600',
          },
          contentStyle: {
            backgroundColor: colors.background,
          },
        }}>
        <Stack.Screen
          name="ClientID"
          component={ClientIDScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="EndUserLogin"
          component={EndUserLoginScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="OIDCSAMLLogin"
          component={OIDCSAMLLoginScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="Login"
          component={LoginScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="AppPinSetup"
          component={AppPinSetupScreen}
          options={{ headerShown: false, gestureEnabled: false }}
        />
        <Stack.Screen
          name="AppPinEntry"
          component={AppPinEntryScreen}
          options={{ headerShown: false, gestureEnabled: false }}
        />
        <Stack.Screen
          name="Home"
          component={MainTabs}
          options={{ headerShown: false, gestureEnabled: false }}
        />
        <Stack.Screen
          name="Notifications"
          component={NotificationsScreen}
          options={{ headerShown: false }}
        />
        <Stack.Screen
          name="Approval"
          component={ApprovalScreen}
          options={{
            title: 'Authentication Request',
            presentation: 'fullScreenModal',
            headerShown: false,
            gestureEnabled: false,
            animation: 'fade',
          }}
        />
        <Stack.Screen
          name="DeviceRegistration"
          component={DeviceRegistrationScreen}
          options={{
            title: 'Device Registration',
          }}
        />
        <Stack.Screen
          name="TOTPSetup"
          component={TOTPSetupScreen}
          options={{
            title: 'TOTP Setup',
          }}
        />
        <Stack.Screen
          name="Authenticator"
          component={AuthenticatorScreen}
          options={{
            headerShown: false,
          }}
        />
      </Stack.Navigator>

      {/* Auth Request Popup - Microsoft Authenticator style */}
      <AuthRequestPopup
        visible={showAuthPopup}
        authReqId={pendingAuthRequest || ''}
        onClose={() => {
          setShowAuthPopup(false);
          setPendingAuthRequest(null);
        }}
        onResponse={(approved) => {
          setShowAuthPopup(false);
          setPendingAuthRequest(null);
        }}
      />
    </NavigationContainer>
  );
}

// Wrap the app with ThemeProvider
const AppWrapper = () => {
  return (
    <ThemeProvider>
      <App />
    </ThemeProvider>
  );
};

export default AppWrapper;
