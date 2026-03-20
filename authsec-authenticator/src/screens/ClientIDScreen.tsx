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
    Keyboard,
    Platform,
    ScrollView,
    StatusBar,
    Image,
    Animated,
    Modal,
    AppState,
} from 'react-native';
import WebView from 'react-native-webview';
import { storeClientId } from '../services/storage';
import { getOIDCAuthURL, getOIDCPageData } from '../services/api';
import { useTheme } from '../context/ThemeContext';
import { Ionicons } from '@expo/vector-icons';

const ClientIDScreen = ({ navigation }: any) => {
    const { colors, isDark } = useTheme();
    const [clientId, setClientId] = useState('');
    const [loading, setLoading] = useState(false);
    const [showWebView, setShowWebView] = useState(false);
    const [authUrl, setAuthUrl] = useState('');
    const [webViewLoading, setWebViewLoading] = useState(true);

    // Animations
    const fadeAnim = useRef(new Animated.Value(0)).current;
    const slideAnim = useRef(new Animated.Value(30)).current;
    const scaleAnim = useRef(new Animated.Value(0.9)).current;

    useEffect(() => {
        const subscription = AppState.addEventListener('change', (nextAppState) => {
            if (nextAppState === 'active') {
                Keyboard.dismiss();
            }
        });
        return () => subscription.remove();
    }, []);

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
            Animated.spring(scaleAnim, {
                toValue: 1,
                tension: 50,
                friction: 7,
                useNativeDriver: true,
            }),
        ]).start();
    }, []);

    const handleContinue = async () => {
        if (!clientId.trim()) {
            Alert.alert('Error', 'Please enter a Client ID or Hydra URL');
            return;
        }

        setLoading(true);

        try {
            const input = clientId.trim();

            // Store the client ID
            await storeClientId(input);
            console.log('Client ID stored successfully');

            // Navigate directly to simplified login screen
            // No provider fetching, no WebView - just browser + manual token flow
            setLoading(false);
            navigation.replace('EndUserLogin', {
                clientId: input,
                simpleFlow: true, // Flag to show simplified UI
            });

        } catch (error: any) {
            console.error('Client ID verification error:', error);
            Alert.alert('Error', error.message || 'Invalid Client ID or URL');
            setLoading(false);
        }
    };

    const handleWebViewNavigationChange = async (navState: any) => {
        const { url } = navState;
        console.log('WebView navigated to:', url);

        // Check if the URL contains login_challenge
        if (url.includes('login_challenge=')) {
            try {
                // Extract login_challenge from URL
                const urlObj = new URL(url);
                const loginChallenge = urlObj.searchParams.get('login_challenge');

                if (loginChallenge) {
                    console.log('Login challenge found:', loginChallenge.substring(0, 50) + '...');

                    // Close WebView
                    setShowWebView(false);
                    setLoading(true);

                    // Step 3: Get page data with detailed provider info
                    console.log('Getting page data...');
                    const pageData = await getOIDCPageData(loginChallenge);

                    console.log('Page data received (FULL):', JSON.stringify(pageData, null, 2));
                    console.log('Page data received:', {
                        client_id: pageData.client_id,
                        tenant_name: pageData.tenant_name,
                        provider_count: pageData.providers?.length || 0,
                        base_url: pageData.base_url,
                    });

                    // Step 4: Filter and format providers
                    // Filter out 'authsec' provider and format for display
                    const formattedProviders = pageData.providers
                        .filter(p => p.provider_name.toLowerCase() !== 'authsec' && p.is_active)
                        .map(p => ({
                            name: p.display_name,
                            provider_name: p.provider_name,
                            auth_url: authUrl, // Use the original auth_url
                            logo_url: undefined, // Can be added later if needed
                            config: p.config,
                        }));

                    console.log('Filtered providers:', formattedProviders.map(p => p.name));

                    // Use base_url from response, or construct it as fallback
                    const baseUrl = pageData.base_url || 'https://prod.api.authsec.ai/hmgr';
                    console.log('Base URL (final):', baseUrl);

                    // Navigate to login screen with OIDC providers, login_challenge, and base_url
                    navigation.replace('EndUserLogin', {
                        clientId: clientId.trim(),
                        oidcProviders: formattedProviders,
                        loginChallenge: loginChallenge, // Pass login_challenge for OIDC initiate calls
                        baseUrl: baseUrl, // Pass base_url from page-data or fallback
                    });
                }
            } catch (error: any) {
                console.error('Error processing login_challenge:', error);
                Alert.alert('Error', error.message || 'Failed to get provider information');
                setShowWebView(false);
                setLoading(false);
            }
        }
    };

    const closeWebView = () => {
        setShowWebView(false);
        setAuthUrl('');
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
                    <Text style={styles.subtitle}>Secure Authentication</Text>
                </Animated.View>

                {/* Form */}
                <View style={styles.form}>
                    <View style={styles.infoBox}>
                        <Text style={styles.infoText}>
                            Enter your organization's Client ID to continue
                        </Text>
                    </View>

                    <View style={styles.inputGroup}>
                        <Text style={styles.label}>Client ID</Text>
                        <TextInput
                            style={styles.input}
                            placeholder="e.g., xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                            placeholderTextColor={colors.inputPlaceholder}
                            value={clientId}
                            onChangeText={setClientId}
                            autoCapitalize="none"
                            autoCorrect={false}
                            editable={!loading}
                            onSubmitEditing={handleContinue}
                        />
                    </View>

                    <TouchableOpacity
                        style={[styles.button, loading && styles.buttonDisabled]}
                        onPress={handleContinue}
                        disabled={loading}
                        activeOpacity={0.8}>
                        {loading ? (
                            <ActivityIndicator color={isDark ? '#000000' : '#FFFFFF'} />
                        ) : (
                            <Text style={styles.buttonText}>Continue</Text>
                        )}
                    </TouchableOpacity>
                </View>

                {/* Footer */}
                <Text style={styles.footer}>
                    Secure • Private • Trusted
                </Text>
            </ScrollView>

            {/* WebView Modal */}
            <Modal
                visible={showWebView}
                animationType="slide"
                onRequestClose={closeWebView}>
                <View style={styles.webViewContainer}>
                    {/* Header */}
                    <View style={styles.webViewHeader}>
                        <Text style={styles.webViewTitle}>Loading Providers...</Text>
                        <TouchableOpacity onPress={closeWebView} style={styles.closeButton}>
                            <Ionicons name="close" size={28} color={colors.text} />
                        </TouchableOpacity>
                    </View>

                    {/* Loading Indicator */}
                    {webViewLoading && (
                        <View style={styles.loadingContainer}>
                            <ActivityIndicator size="large" color={colors.primary} />
                            <Text style={styles.loadingText}>Please wait...</Text>
                        </View>
                    )}

                    {/* WebView */}
                    {authUrl ? (
                        <WebView
                            source={{ uri: authUrl }}
                            onNavigationStateChange={handleWebViewNavigationChange}
                            onLoadStart={() => setWebViewLoading(true)}
                            onLoadEnd={() => setWebViewLoading(false)}
                            style={styles.webView}
                            javaScriptEnabled={true}
                            domStorageEnabled={true}
                        />
                    ) : null}
                </View>
            </Modal>
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
            backgroundColor: `${colors.primary}15`,
            borderLeftWidth: 3,
            borderLeftColor: colors.primary,
            borderRadius: 12,
            padding: 16,
            marginBottom: 24,
        },
        infoText: {
            fontSize: 14,
            color: colors.text,
            lineHeight: 20,
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
        footer: {
            textAlign: 'center',
            fontSize: 12,
            color: colors.textMuted,
            marginTop: 40,
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

export default ClientIDScreen;
